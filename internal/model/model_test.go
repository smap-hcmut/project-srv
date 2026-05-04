package model

import (
	"testing"
	"time"

	"project-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
	"github.com/stretchr/testify/require"
)

func TestIsValidProjectStatus(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output bool
		err    error
	}{
		"pending":  {input: string(ProjectStatusPending), output: true},
		"active":   {input: string(ProjectStatusActive), output: true},
		"paused":   {input: string(ProjectStatusPaused), output: true},
		"archived": {input: string(ProjectStatusArchived), output: true},
		"invalid":  {input: "BAD", output: false},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := IsValidProjectStatus(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestIsValidEntityType(t *testing.T) {
	tcs := map[string]struct {
		input  string
		mock   struct{}
		output bool
		err    error
	}{
		"product":    {input: string(EntityTypeProduct), output: true},
		"campaign":   {input: string(EntityTypeCampaign), output: true},
		"service":    {input: string(EntityTypeService), output: true},
		"competitor": {input: string(EntityTypeCompetitor), output: true},
		"topic":      {input: string(EntityTypeTopic), output: true},
		"invalid":    {input: "BAD", output: false},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := IsValidEntityType(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestProjectLifecycleStatusGuards(t *testing.T) {
	tcs := map[string]struct {
		input ProjectStatus
		mock  struct {
			fn func(ProjectStatus) bool
		}
		output bool
		err    error
	}{
		"can_activate":       {input: ProjectStatusPending, mock: struct{ fn func(ProjectStatus) bool }{fn: CanActivateProjectStatus}, output: true},
		"cannot_activate":    {input: ProjectStatusActive, mock: struct{ fn func(ProjectStatus) bool }{fn: CanActivateProjectStatus}, output: false},
		"can_pause":          {input: ProjectStatusActive, mock: struct{ fn func(ProjectStatus) bool }{fn: CanPauseProjectStatus}, output: true},
		"cannot_pause":       {input: ProjectStatusPending, mock: struct{ fn func(ProjectStatus) bool }{fn: CanPauseProjectStatus}, output: false},
		"can_resume":         {input: ProjectStatusPaused, mock: struct{ fn func(ProjectStatus) bool }{fn: CanResumeProjectStatus}, output: true},
		"cannot_resume":      {input: ProjectStatusActive, mock: struct{ fn func(ProjectStatus) bool }{fn: CanResumeProjectStatus}, output: false},
		"can_archive_active": {input: ProjectStatusActive, mock: struct{ fn func(ProjectStatus) bool }{fn: CanArchiveProjectStatus}, output: true},
		"cannot_archive":     {input: ProjectStatusArchived, mock: struct{ fn func(ProjectStatus) bool }{fn: CanArchiveProjectStatus}, output: false},
		"can_unarchive":      {input: ProjectStatusArchived, mock: struct{ fn func(ProjectStatus) bool }{fn: CanUnarchiveProjectStatus}, output: true},
		"cannot_unarchive":   {input: ProjectStatusPaused, mock: struct{ fn func(ProjectStatus) bool }{fn: CanUnarchiveProjectStatus}, output: false},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := tc.mock.fn(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestNewCampaignFromDB(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  *sqlboiler.Campaign
		mock   struct{}
		output *Campaign
		err    error
	}{
		"nil": {},
		"success": {
			input: &sqlboiler.Campaign{
				ID:          "campaign-1",
				Name:        "Campaign A",
				Description: null.StringFrom("Desc"),
				Status:      sqlboiler.CampaignStatusPENDING,
				StartDate:   null.TimeFrom(now),
				EndDate:     null.TimeFrom(now),
				CreatedBy:   "user-1",
				CreatedAt:   null.TimeFrom(now),
				UpdatedAt:   null.TimeFrom(now),
			},
			output: &Campaign{ID: "campaign-1", Name: "Campaign A", Description: "Desc", Status: CampaignStatusPending, StartDate: &now, EndDate: &now, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := NewCampaignFromDB(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestNewProjectFromDB(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  *sqlboiler.Project
		mock   struct{}
		output *Project
		err    error
	}{
		"nil": {},
		"success": {
			input: &sqlboiler.Project{
				ID:             "project-1",
				CampaignID:     "campaign-1",
				Name:           "Project A",
				Description:    null.StringFrom("Desc"),
				Brand:          null.StringFrom("Brand"),
				EntityType:     sqlboiler.EntityTypeProduct,
				EntityName:     "VF8",
				DomainTypeCode: "ev",
				Status:         sqlboiler.ProjectStatusPENDING,
				ConfigStatus:   sqlboiler.NullProjectConfigStatusFrom(sqlboiler.ProjectConfigStatusDRAFT),
				CreatedBy:      "user-1",
				CreatedAt:      null.TimeFrom(now),
				UpdatedAt:      null.TimeFrom(now),
			},
			output: &Project{ID: "project-1", CampaignID: "campaign-1", Name: "Project A", Description: "Desc", Brand: "Brand", EntityType: EntityTypeProduct, EntityName: "VF8", DomainTypeCode: "ev", Status: ProjectStatusPending, ConfigStatus: ConfigStatusDraft, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := NewProjectFromDB(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestNewCrisisConfigFromDB(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  *sqlboiler.ProjectsCrisisConfig
		mock   struct{}
		output *CrisisConfig
		err    error
	}{
		"nil": {},
		"success": {
			input: &sqlboiler.ProjectsCrisisConfig{
				ProjectID:       "project-1",
				Status:          sqlboiler.NullCrisisStatusFrom(sqlboiler.CrisisStatusCRITICAL),
				KeywordsRules:   null.JSONFrom([]byte(`{"enabled":true,"logic":"AND","groups":[{"name":"Pin","keywords":["pin"],"weight":10}]}`)),
				VolumeRules:     null.JSONFrom([]byte(`{"enabled":true,"metric":"MENTIONS","rules":[{"level":"CRITICAL","threshold_percent_growth":150,"comparison_window_hours":1,"baseline":"PREVIOUS_PERIOD"}]}`)),
				SentimentRules:  null.JSONFrom([]byte(`{"enabled":true,"min_sample_size":10,"rules":[{"type":"NEGATIVE_SPIKE","threshold_percent":25}]}`)),
				InfluencerRules: null.JSONFrom([]byte(`{"enabled":true,"logic":"OR","rules":[{"type":"HIGH_REACH","min_followers":1000}]}`)),
				CreatedAt:       null.TimeFrom(now),
				UpdatedAt:       null.TimeFrom(now),
			},
			output: &CrisisConfig{
				ProjectID:         "project-1",
				Status:            CrisisStatusCritical,
				KeywordsTrigger:   KeywordsTrigger{Enabled: true, Logic: "AND", Groups: []KeywordGroup{{Name: "Pin", Keywords: []string{"pin"}, Weight: 10}}},
				VolumeTrigger:     VolumeTrigger{Enabled: true, Metric: "MENTIONS", Rules: []VolumeRule{{Level: "CRITICAL", ThresholdPercentGrowth: 150, ComparisonWindowHours: 1, Baseline: "PREVIOUS_PERIOD"}}},
				SentimentTrigger:  SentimentTrigger{Enabled: true, MinSampleSize: 10, Rules: []SentimentRule{{Type: "NEGATIVE_SPIKE", ThresholdPercent: 25}}},
				InfluencerTrigger: InfluencerTrigger{Enabled: true, Logic: "OR", Rules: []InfluencerRule{{Type: "HIGH_REACH", MinFollowers: 1000}}},
				CreatedAt:         now,
				UpdatedAt:         now,
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := NewCrisisConfigFromDB(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestScopeRoles(t *testing.T) {
	tcs := map[string]struct {
		input Scope
		mock  struct {
			fn func(Scope) bool
		}
		output bool
		err    error
	}{
		"admin":       {input: Scope{Role: RoleAdmin}, mock: struct{ fn func(Scope) bool }{fn: Scope.IsAdmin}, output: true},
		"not_admin":   {input: Scope{Role: RoleViewer}, mock: struct{ fn func(Scope) bool }{fn: Scope.IsAdmin}, output: false},
		"analyst":     {input: Scope{Role: RoleAnalyst}, mock: struct{ fn func(Scope) bool }{fn: Scope.IsAnalyst}, output: true},
		"not_analyst": {input: Scope{Role: RoleAdmin}, mock: struct{ fn func(Scope) bool }{fn: Scope.IsAnalyst}, output: false},
		"viewer":      {input: Scope{Role: RoleViewer}, mock: struct{ fn func(Scope) bool }{fn: Scope.IsViewer}, output: true},
		"not_viewer":  {input: Scope{Role: RoleAdmin}, mock: struct{ fn func(Scope) bool }{fn: Scope.IsViewer}, output: false},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := tc.mock.fn(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}
