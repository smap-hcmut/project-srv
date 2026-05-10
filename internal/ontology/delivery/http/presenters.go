package http

import (
	"strings"

	"project-srv/internal/model"
	"project-srv/internal/ontology"
)

type ruleReq struct {
	ID              string   `json:"id,omitempty"`
	Label           string   `json:"label"`
	Description     string   `json:"description,omitempty"`
	TargetKind      string   `json:"target_kind"`
	TargetKey       string   `json:"target_key"`
	MatchMode       string   `json:"match_mode"`
	Phrases         []string `json:"phrases"`
	Patterns        []string `json:"patterns"`
	NegativePhrases []string `json:"negative_phrases,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
	Weight          int      `json:"weight,omitempty"`
	SampleText      string   `json:"sample_text,omitempty"`
}

type upsertReq struct {
	Enabled *bool     `json:"enabled,omitempty"`
	Rules   []ruleReq `json:"rules"`
}

type testReq struct {
	Enabled *bool     `json:"enabled,omitempty"`
	Rules   []ruleReq `json:"rules"`
	Text    string    `json:"text"`
}

type ruleResp struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Description     string   `json:"description,omitempty"`
	TargetKind      string   `json:"target_kind"`
	TargetKey       string   `json:"target_key"`
	MatchMode       string   `json:"match_mode"`
	Phrases         []string `json:"phrases"`
	Patterns        []string `json:"patterns"`
	NegativePhrases []string `json:"negative_phrases,omitempty"`
	Enabled         bool     `json:"enabled"`
	Weight          int      `json:"weight"`
	SampleText      string   `json:"sample_text,omitempty"`
}

type configResp struct {
	ProjectID string     `json:"project_id"`
	Enabled   bool       `json:"enabled"`
	Rules     []ruleResp `json:"rules"`
	CreatedAt string     `json:"created_at,omitempty"`
	UpdatedAt string     `json:"updated_at,omitempty"`
}

type detailResp struct {
	OntologyRules configResp `json:"ontology_rules"`
}

type upsertResp struct {
	OntologyRules configResp `json:"ontology_rules"`
}

type runtimeResp struct {
	ProjectID     string     `json:"project_id"`
	OntologyRules configResp `json:"ontology_rules"`
}

type testMatchResp struct {
	RuleID     string   `json:"rule_id"`
	Label      string   `json:"label"`
	TargetKind string   `json:"target_kind"`
	TargetKey  string   `json:"target_key"`
	Matched    bool     `json:"matched"`
	Evidence   []string `json:"evidence"`
}

type testResp struct {
	Matches []testMatchResp `json:"matches"`
}

func (r upsertReq) toInput(projectID string) ontology.UpsertInput {
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return ontology.UpsertInput{
		ProjectID: strings.TrimSpace(projectID),
		Enabled:   enabled,
		Rules:     toModelRules(r.Rules),
	}
}

func (r testReq) toInput(projectID string) ontology.TestInput {
	enabled := true
	if r.Enabled != nil {
		enabled = *r.Enabled
	}
	return ontology.TestInput{
		ProjectID: strings.TrimSpace(projectID),
		Enabled:   enabled,
		Rules:     toModelRules(r.Rules),
		Text:      r.Text,
	}
}

func toModelRules(rules []ruleReq) []model.OntologySignalRule {
	out := make([]model.OntologySignalRule, len(rules))
	for i, rule := range rules {
		enabled := true
		if rule.Enabled != nil {
			enabled = *rule.Enabled
		}
		out[i] = model.OntologySignalRule{
			ID:              rule.ID,
			Label:           rule.Label,
			Description:     rule.Description,
			TargetKind:      model.OntologyTargetKind(rule.TargetKind),
			TargetKey:       rule.TargetKey,
			MatchMode:       model.OntologyMatchMode(rule.MatchMode),
			Phrases:         append([]string(nil), rule.Phrases...),
			Patterns:        append([]string(nil), rule.Patterns...),
			NegativePhrases: append([]string(nil), rule.NegativePhrases...),
			Enabled:         enabled,
			Weight:          rule.Weight,
			SampleText:      rule.SampleText,
		}
	}
	return out
}

func (h *handler) newDetailResp(o ontology.DetailOutput) detailResp {
	return detailResp{OntologyRules: toConfigResp(o.Config)}
}

func (h *handler) newUpsertResp(o ontology.UpsertOutput) upsertResp {
	return upsertResp{OntologyRules: toConfigResp(o.Config)}
}

func (h *handler) newRuntimeResp(o ontology.RuntimeOutput) runtimeResp {
	return runtimeResp{ProjectID: o.ProjectID, OntologyRules: toConfigResp(o.Config)}
}

func (h *handler) newTestResp(o ontology.TestOutput) testResp {
	matches := make([]testMatchResp, len(o.Matches))
	for i, item := range o.Matches {
		matches[i] = testMatchResp{
			RuleID:     item.RuleID,
			Label:      item.Label,
			TargetKind: item.TargetKind,
			TargetKey:  item.TargetKey,
			Matched:    item.Matched,
			Evidence:   item.Evidence,
		}
	}
	return testResp{Matches: matches}
}

func toConfigResp(config model.ProjectOntologyRules) configResp {
	rules := make([]ruleResp, len(config.Rules))
	for i, rule := range config.Rules {
		rules[i] = ruleResp{
			ID:              rule.ID,
			Label:           rule.Label,
			Description:     rule.Description,
			TargetKind:      string(rule.TargetKind),
			TargetKey:       rule.TargetKey,
			MatchMode:       string(rule.MatchMode),
			Phrases:         cloneStrings(rule.Phrases),
			Patterns:        cloneStrings(rule.Patterns),
			NegativePhrases: cloneStrings(rule.NegativePhrases),
			Enabled:         rule.Enabled,
			Weight:          rule.Weight,
			SampleText:      rule.SampleText,
		}
	}
	return configResp{
		ProjectID: config.ProjectID,
		Enabled:   config.Enabled,
		Rules:     rules,
		CreatedAt: config.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: config.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}
