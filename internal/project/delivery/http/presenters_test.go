package http

import (
	"errors"
	"testing"
	"time"

	"project-srv/internal/model"
	"project-srv/internal/project"

	"github.com/stretchr/testify/require"
)

func TestMapError(t *testing.T) {
	h, _ := newTestHandler(t)

	tcs := map[string]struct {
		input  error
		mock   struct{}
		output error
		err    error
	}{
		"not_found":                       {input: project.ErrNotFound, output: errNotFound},
		"name_required":                   {input: project.ErrNameRequired, output: errNameRequired},
		"campaign_required":               {input: project.ErrCampaignRequired, output: errCampaignRequired},
		"campaign_not_found":              {input: project.ErrCampaignNotFound, output: errCampaignNotFound},
		"invalid_status":                  {input: project.ErrInvalidStatus, output: errInvalidStatus},
		"invalid_entity":                  {input: project.ErrInvalidEntity, output: errInvalidEntity},
		"domain_required":                 {input: project.ErrDomainTypeRequired, output: errDomainTypeRequired},
		"invalid_domain":                  {input: project.ErrInvalidDomainType, output: errInvalidDomainType},
		"invalid_sort":                    {input: project.ErrInvalidSort, output: errInvalidSort},
		"create_failed":                   {input: project.ErrCreateFailed, output: errCreateFailed},
		"detail_failed":                   {input: project.ErrDetailFailed, output: errDetailFailed},
		"update_failed":                   {input: project.ErrUpdateFailed, output: errUpdateFailed},
		"delete_failed":                   {input: project.ErrDeleteFailed, output: errDeleteFailed},
		"list_failed":                     {input: project.ErrListFailed, output: errListFailed},
		"invalid_transition":              {input: project.ErrInvalidTransition, output: errInvalidTransition},
		"activate_not_allowed":            {input: project.ErrActivateNotAllowed, output: errActivateNotAllowed},
		"pause_not_allowed":               {input: project.ErrPauseNotAllowed, output: errPauseNotAllowed},
		"resume_not_allowed":              {input: project.ErrResumeNotAllowed, output: errResumeNotAllowed},
		"unarchive_not_allowed":           {input: project.ErrUnarchiveNotAllowed, output: errUnarchiveNotAllowed},
		"readiness_datasource_required":   {input: project.ErrReadinessDatasourceRequired, output: errReadinessDatasourceRequired},
		"readiness_passive_unconfirmed":   {input: project.ErrReadinessPassiveUnconfirmed, output: errReadinessPassiveUnconfirmed},
		"readiness_target_dryrun_missing": {input: project.ErrReadinessTargetDryrunMissing, output: errReadinessTargetDryrunMissing},
		"readiness_target_dryrun_failed":  {input: project.ErrReadinessTargetDryrunFailed, output: errReadinessTargetDryrunFailed},
		"readiness_active_target_missing": {input: project.ErrReadinessActiveTargetMissing, output: errReadinessActiveTargetMissing},
		"readiness_datasource_status":     {input: project.ErrReadinessDatasourceStatus, output: errReadinessDatasourceStatus},
		"readiness_failed":                {input: project.ErrReadinessFailed, output: errReadinessFailed},
		"lifecycle_failed":                {input: project.ErrLifecycleManagerFailed, output: errLifecycleManagerFailed},
		"lifecycle_rejected":              {input: project.ErrLifecycleManagerRejected, output: errLifecycleManagerRejected},
		"lifecycle_unauthorized":          {input: project.ErrLifecycleManagerUnauthorized, output: errLifecycleManagerUnauthorized},
		"lifecycle_forbidden":             {input: project.ErrLifecycleManagerForbidden, output: errLifecycleManagerForbidden},
		"delete_requires_archived":        {input: project.ErrDeleteRequiresArchived, output: errDeleteRequiresArchived},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := h.mapError(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestMapErrorDefault(t *testing.T) {
	h, _ := newTestHandler(t)

	tcs := map[string]struct {
		input  error
		mock   struct{}
		output struct{}
		err    error
	}{
		"panic_unknown_error": {input: errors.New("unknown")},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			require.Panics(t, func() {
				_ = h.mapError(tc.input)
			})
		})
	}
}

func TestCreateReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  createReq
		mock   struct{}
		output struct{}
		err    error
	}{
		"success":              {input: createReq{CampaignID: testCampaignID, Name: "Project A", EntityType: "product", EntityName: "VF8", DomainTypeCode: " ev "}},
		"name_required":        {input: createReq{CampaignID: testCampaignID}, err: errNameRequired},
		"campaign_required":    {input: createReq{Name: "Project A"}, err: errCampaignRequired},
		"bad_campaign_id":      {input: createReq{CampaignID: "bad", Name: "Project A"}, err: errWrongBody},
		"entity_type_required": {input: createReq{CampaignID: testCampaignID, Name: "Project A"}, err: errEntityTypeRequired},
		"invalid_entity":       {input: createReq{CampaignID: testCampaignID, Name: "Project A", EntityType: "bad"}, err: errInvalidEntity},
		"entity_name_required": {input: createReq{CampaignID: testCampaignID, Name: "Project A", EntityType: "product"}, err: errEntityNameRequired},
		"domain_required":      {input: createReq{CampaignID: testCampaignID, Name: "Project A", EntityType: "product", EntityName: "VF8", DomainTypeCode: " "}, err: errDomainTypeRequired},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUpdateReqValidate(t *testing.T) {
	tcs := map[string]struct {
		input  updateReq
		mock   struct{}
		output struct{}
		err    error
	}{
		"success":        {input: updateReq{ID: testProjectID, EntityType: "product"}},
		"bad_id":         {input: updateReq{ID: "bad"}, err: errWrongBody},
		"invalid_entity": {input: updateReq{ID: testProjectID, EntityType: "bad"}, err: errInvalidEntity},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input.validate()

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestProjectRequestValidate(t *testing.T) {
	tcs := map[string]struct {
		input  func() error
		mock   struct{}
		output struct{}
		err    error
	}{
		"list_bad_id":                {input: func() error { return listReq{CampaignID: "bad"}.validate() }, err: errWrongBody},
		"list_bad_sort":              {input: func() error { return listReq{CampaignID: testCampaignID, Sort: "bad"}.validate() }, err: errWrongQuery},
		"favorite_list_bad_campaign": {input: func() error { return favoriteListReq{CampaignID: "bad"}.validate() }, err: errWrongQuery},
		"favorite_list_bad_sort":     {input: func() error { return favoriteListReq{Sort: "bad"}.validate() }, err: errWrongQuery},
		"readiness_bad_id":           {input: func() error { return activationReadinessReq{ID: "bad"}.validate() }, err: errWrongBody},
		"readiness_bad_command":      {input: func() error { return activationReadinessReq{ID: testProjectID, Command: "bad"}.validate() }, err: errWrongBody},
		"readiness_default_command":  {input: func() error { return activationReadinessReq{ID: testProjectID}.validate() }},
		"readiness_activate_command": {input: func() error {
			return activationReadinessReq{ID: testProjectID, Command: string(project.ActivationReadinessCommandActivate)}.validate()
		}},
		"readiness_resume_command": {input: func() error {
			return activationReadinessReq{ID: testProjectID, Command: string(project.ActivationReadinessCommandResume)}.validate()
		}},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := tc.input()

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestProjectMappers(t *testing.T) {
	h, _ := newTestHandler(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  model.Project
		mock   struct{}
		output projectResp
		err    error
	}{
		"with_optional_fields": {
			input:  model.Project{ID: testProjectID, CampaignID: testCampaignID, Name: "Project A", Description: "Desc", Brand: "Brand", EntityType: model.EntityTypeProduct, EntityName: "VF8", DomainTypeCode: "ev", Status: model.ProjectStatusActive, IsFavorite: true, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now},
			output: projectResp{ID: testProjectID, CampaignID: testCampaignID, Name: "Project A", Description: "Desc", Brand: "Brand", EntityType: "product", EntityName: "VF8", DomainTypeCode: "ev", Status: "ACTIVE", IsFavorite: true, CreatedBy: "user-1", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := h.toProjectResp(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestNewActivationReadinessResp(t *testing.T) {
	h, _ := newTestHandler(t)

	tcs := map[string]struct {
		input  project.ActivationReadiness
		mock   struct{}
		output activationReadinessResp
		err    error
	}{
		"with_errors": {
			input: project.ActivationReadiness{
				ProjectID:                testProjectID,
				ProjectStatus:            model.ProjectStatusPending,
				DataSourceCount:          2,
				HasDatasource:            true,
				PassiveUnconfirmedCount:  1,
				MissingTargetDryrunCount: 1,
				FailedTargetDryrunCount:  1,
				CanProceed:               false,
				Errors: []project.ActivationReadinessError{{
					Code:         project.ActivationReadinessCodeDatasourceRequired,
					Message:      "missing datasource",
					DataSourceID: "datasource-1",
					TargetID:     "target-1",
				}},
			},
			output: activationReadinessResp{
				ProjectID:                testProjectID,
				ProjectStatus:            "PENDING",
				DataSourceCount:          2,
				HasDatasource:            true,
				PassiveUnconfirmedCount:  1,
				MissingTargetDryrunCount: 1,
				FailedTargetDryrunCount:  1,
				CanProceed:               false,
				Errors: []activationReadinessErrorResp{{
					Code:         string(project.ActivationReadinessCodeDatasourceRequired),
					Message:      "missing datasource",
					DataSourceID: "datasource-1",
					TargetID:     "target-1",
				}},
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := h.newActivationReadinessResp(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}
