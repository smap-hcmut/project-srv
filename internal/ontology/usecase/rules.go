package usecase

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"

	"project-srv/internal/model"
	"project-srv/internal/ontology"
	repo "project-srv/internal/ontology/repository"

	"github.com/smap-hcmut/shared-libs/go/auth"
)

const (
	maxOntologyRulesPerProject = 80
	maxPhrasesPerRule          = 30
	maxPatternsPerRule         = 10
	maxPhraseLen               = 120
	maxPatternLen              = 200
)

func (uc *implUseCase) Upsert(ctx context.Context, input ontology.UpsertInput) (ontology.UpsertOutput, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return ontology.UpsertOutput{}, ontology.ErrProjectInvalid
	}
	if _, err := uc.projectUC.Detail(ctx, projectID); err != nil {
		uc.l.Warnf(ctx, "ontology.usecase.Upsert.validateProject: project_id=%s err=%v", projectID, err)
		return ontology.UpsertOutput{}, ontology.ErrProjectInvalid
	}

	rules, err := normalizeRules(input.Rules)
	if err != nil {
		uc.l.Warnf(ctx, "ontology.usecase.Upsert.normalizeRules: project_id=%s err=%v", projectID, err)
		return ontology.UpsertOutput{}, ontology.ErrInvalidRules
	}

	updatedBy := strings.TrimSpace(input.UpdatedBy)
	if updatedBy == "" {
		updatedBy, _ = auth.GetUserIDFromContext(ctx)
	}

	config, err := uc.repo.Upsert(ctx, repo.UpsertOptions{
		ProjectID: projectID,
		Enabled:   input.Enabled,
		Rules:     rules,
		UpdatedBy: updatedBy,
	})
	if err != nil {
		uc.l.Errorf(ctx, "ontology.usecase.Upsert.repo.Upsert: project_id=%s err=%v", projectID, err)
		return ontology.UpsertOutput{}, ontology.ErrUpsertFailed
	}

	return ontology.UpsertOutput{Config: config}, nil
}

func (uc *implUseCase) Detail(ctx context.Context, projectID string) (ontology.DetailOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ontology.DetailOutput{}, ontology.ErrProjectInvalid
	}
	if _, err := uc.projectUC.Detail(ctx, projectID); err != nil {
		uc.l.Warnf(ctx, "ontology.usecase.Detail.validateProject: project_id=%s err=%v", projectID, err)
		return ontology.DetailOutput{}, ontology.ErrProjectInvalid
	}

	config, err := uc.repo.Detail(ctx, projectID)
	if err != nil {
		if err == repo.ErrFailedToGet {
			return ontology.DetailOutput{Config: model.DefaultProjectOntologyRules(projectID)}, nil
		}
		uc.l.Errorf(ctx, "ontology.usecase.Detail.repo.Detail: project_id=%s err=%v", projectID, err)
		return ontology.DetailOutput{}, ontology.ErrNotFound
	}

	return ontology.DetailOutput{Config: config}, nil
}

func (uc *implUseCase) Runtime(ctx context.Context, projectID string) (ontology.RuntimeOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ontology.RuntimeOutput{}, ontology.ErrProjectInvalid
	}

	config, err := uc.repo.Detail(ctx, projectID)
	if err != nil {
		if err == repo.ErrFailedToGet {
			return ontology.RuntimeOutput{ProjectID: projectID, Config: model.DefaultProjectOntologyRules(projectID)}, nil
		}
		uc.l.Errorf(ctx, "ontology.usecase.Runtime.repo.Detail: project_id=%s err=%v", projectID, err)
		return ontology.RuntimeOutput{}, ontology.ErrNotFound
	}

	return ontology.RuntimeOutput{ProjectID: projectID, Config: config}, nil
}

func (uc *implUseCase) Test(ctx context.Context, input ontology.TestInput) (ontology.TestOutput, error) {
	projectID := strings.TrimSpace(input.ProjectID)
	if projectID == "" {
		return ontology.TestOutput{}, ontology.ErrProjectInvalid
	}
	if _, err := uc.projectUC.Detail(ctx, projectID); err != nil {
		uc.l.Warnf(ctx, "ontology.usecase.Test.validateProject: project_id=%s err=%v", projectID, err)
		return ontology.TestOutput{}, ontology.ErrProjectInvalid
	}

	rules, err := normalizeRules(input.Rules)
	if err != nil {
		return ontology.TestOutput{}, ontology.ErrInvalidRules
	}

	text := strings.TrimSpace(input.Text)
	matches := make([]ontology.TestMatch, 0, len(rules))
	for _, rule := range rules {
		matched, evidence := matchRule(rule, text)
		matches = append(matches, ontology.TestMatch{
			RuleID:     rule.ID,
			Label:      rule.Label,
			TargetKind: string(rule.TargetKind),
			TargetKey:  rule.TargetKey,
			Matched:    matched,
			Evidence:   evidence,
		})
	}
	return ontology.TestOutput{Matches: matches}, nil
}

func (uc *implUseCase) Delete(ctx context.Context, projectID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return ontology.ErrProjectInvalid
	}
	if err := uc.repo.Delete(ctx, projectID); err != nil {
		if err == repo.ErrFailedToGet {
			return ontology.ErrNotFound
		}
		return ontology.ErrDeleteFailed
	}
	return nil
}

func normalizeRules(rules []model.OntologySignalRule) ([]model.OntologySignalRule, error) {
	if len(rules) > maxOntologyRulesPerProject {
		return nil, ontology.ErrInvalidRules
	}

	result := make([]model.OntologySignalRule, 0, len(rules))
	seenIDs := map[string]struct{}{}
	for _, rule := range rules {
		normalized, err := normalizeRule(rule)
		if err != nil {
			return nil, err
		}
		if _, ok := seenIDs[normalized.ID]; ok {
			suffix := shortHash(normalized.Label + ":" + normalized.TargetKey)
			normalized.ID = normalized.ID + "-" + suffix
		}
		seenIDs[normalized.ID] = struct{}{}
		result = append(result, normalized)
	}
	return result, nil
}

func normalizeRule(rule model.OntologySignalRule) (model.OntologySignalRule, error) {
	rule.Label = strings.TrimSpace(rule.Label)
	rule.Description = strings.TrimSpace(rule.Description)
	rule.TargetKey = strings.TrimSpace(rule.TargetKey)
	rule.SampleText = strings.TrimSpace(rule.SampleText)
	if rule.Label == "" || rule.TargetKey == "" {
		return model.OntologySignalRule{}, ontology.ErrInvalidRules
	}

	rule.TargetKind = model.OntologyTargetKind(strings.ToUpper(strings.TrimSpace(string(rule.TargetKind))))
	switch rule.TargetKind {
	case model.OntologyTargetKindAspect, model.OntologyTargetKindIssue, model.OntologyTargetKindTopic:
	default:
		return model.OntologySignalRule{}, ontology.ErrInvalidRules
	}

	rule.MatchMode = model.OntologyMatchMode(strings.ToUpper(strings.TrimSpace(string(rule.MatchMode))))
	if rule.MatchMode == "" {
		rule.MatchMode = model.OntologyMatchModeAny
	}
	switch rule.MatchMode {
	case model.OntologyMatchModeAny, model.OntologyMatchModeAll, model.OntologyMatchModeRegex:
	default:
		return model.OntologySignalRule{}, ontology.ErrInvalidRules
	}

	rule.Phrases = normalizePhraseList(rule.Phrases, maxPhrasesPerRule)
	rule.NegativePhrases = normalizePhraseList(rule.NegativePhrases, maxPhrasesPerRule)
	if len(rule.Phrases) == 0 && len(rule.Patterns) == 0 {
		return model.OntologySignalRule{}, ontology.ErrInvalidRules
	}
	if len(rule.Patterns) > maxPatternsPerRule {
		return model.OntologySignalRule{}, ontology.ErrInvalidRules
	}
	patterns := make([]string, 0, len(rule.Patterns))
	for _, pattern := range rule.Patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if len([]rune(pattern)) > maxPatternLen || !isSafeMarketingRegex(pattern) {
			return model.OntologySignalRule{}, ontology.ErrInvalidRules
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return model.OntologySignalRule{}, ontology.ErrInvalidRules
		}
		patterns = append(patterns, pattern)
	}
	rule.Patterns = uniqueStrings(patterns)
	if rule.Patterns == nil {
		rule.Patterns = []string{}
	}
	if rule.NegativePhrases == nil {
		rule.NegativePhrases = []string{}
	}
	if len(rule.Phrases) == 0 && len(rule.Patterns) == 0 {
		return model.OntologySignalRule{}, ontology.ErrInvalidRules
	}

	if strings.TrimSpace(rule.ID) == "" {
		rule.ID = slugify(rule.Label)
	}
	rule.ID = slugify(rule.ID)
	if rule.ID == "" {
		rule.ID = "rule-" + shortHash(rule.Label+rule.TargetKey)
	}
	if rule.Weight <= 0 {
		rule.Weight = 1
	}
	if rule.Weight > 100 {
		rule.Weight = 100
	}
	if !rule.Enabled {
		// Preserve explicit false. A blank draft from older clients still becomes true.
		rule.Enabled = false
	}
	return rule, nil
}

func normalizePhraseList(values []string, maxItems int) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
		if value == "" || len([]rune(value)) > maxPhraseLen {
			continue
		}
		out = append(out, value)
		if len(out) >= maxItems {
			break
		}
	}
	return uniqueStrings(out)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isSafeMarketingRegex(pattern string) bool {
	blocked := []string{"(?=", "(?!", "(?<=", "(?<!", "\\1", "\\2", "\\3", "\\k", "(?P"}
	for _, token := range blocked {
		if strings.Contains(pattern, token) {
			return false
		}
	}
	// Deny obvious nested quantifier shapes that Python's backtracking engine
	// would handle poorly. Go RE2 is safe, but analysis-srv applies the same
	// saved pattern in Python.
	nested := regexp.MustCompile(`\([^)]*[+*][^)]*\)[+*{]`)
	return !nested.MatchString(pattern)
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}

func matchRule(rule model.OntologySignalRule, text string) (bool, []string) {
	if !rule.Enabled {
		return false, nil
	}
	if strings.TrimSpace(text) == "" {
		return false, nil
	}
	lower := strings.ToLower(text)
	for _, negative := range rule.NegativePhrases {
		if strings.Contains(lower, strings.ToLower(negative)) {
			return false, nil
		}
	}

	evidence := []string{}
	required := 0
	matchedCount := 0
	if rule.MatchMode == model.OntologyMatchModeRegex {
		required = len(rule.Patterns)
	} else {
		required = len(rule.Phrases) + len(rule.Patterns)
	}
	for _, phrase := range rule.Phrases {
		if strings.Contains(lower, strings.ToLower(phrase)) {
			evidence = append(evidence, phrase)
			matchedCount++
		}
	}
	for _, pattern := range rule.Patterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}
		found := re.FindString(text)
		if found != "" {
			evidence = append(evidence, found)
			matchedCount++
		}
	}
	if rule.MatchMode == model.OntologyMatchModeAll {
		return required > 0 && matchedCount >= required, evidence
	}
	return matchedCount > 0, evidence
}
