package usecase

import (
	"context"
	"errors"
	"testing"

	"project-srv/internal/crisis"
	"project-srv/internal/crisis/repository"
	"project-srv/internal/model"
	"project-srv/internal/project"
	"project-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

type mockDeps struct {
	repo      *repository.MockRepository
	projectUC *project.MockUseCase
	ingest    *microservice.MockIngestUseCase
}

func initUseCase(t *testing.T) (*implUseCase, mockDeps) {
	t.Helper()

	l := log.NewZapLogger(log.ZapConfig{
		Level:        log.LevelFatal,
		Mode:         log.ModeProduction,
		Encoding:     log.EncodingJSON,
		ColorEnabled: false,
	})
	repo := repository.NewMockRepository(t)
	projectUC := project.NewMockUseCase(t)
	ingest := microservice.NewMockIngestUseCase(t)
	uc := New(l, repo, projectUC, ingest).(*implUseCase)

	return uc, mockDeps{repo: repo, projectUC: projectUC, ingest: ingest}
}

func TestUpsert(t *testing.T) {
	ctx := context.Background()
	trigger := &model.KeywordsTrigger{Enabled: true, Logic: "OR", Groups: []model.KeywordGroup{{Name: "brand", Keywords: []string{"smap"}, Weight: 1}}}
	badStatus := model.CrisisStatus("BAD")

	type mockProjectDetail struct {
		isCalled bool
		input    string
		output   project.DetailOutput
		err      error
	}
	type mockRepoUpsert struct {
		isCalled bool
		input    repository.UpsertOptions
		output   model.CrisisConfig
		err      error
	}
	type mock struct {
		projectDetail mockProjectDetail
		repoUpsert    mockRepoUpsert
	}

	tcs := map[string]struct {
		input  crisis.UpsertInput
		mock   mock
		output crisis.UpsertOutput
		err    error
	}{
		"success": {
			input: crisis.UpsertInput{ProjectID: "project-1", KeywordsTrigger: trigger},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoUpsert: mockRepoUpsert{
					isCalled: true,
					input:    repository.UpsertOptions{ProjectID: "project-1", KeywordsTrigger: trigger},
					output:   model.CrisisConfig{ProjectID: "project-1", KeywordsTrigger: *trigger},
				},
			},
			output: crisis.UpsertOutput{CrisisConfig: model.CrisisConfig{ProjectID: "project-1", KeywordsTrigger: *trigger}},
		},
		"empty_project_id": {
			err: crisis.ErrProjectInvalid,
		},
		"invalid_status": {
			input: crisis.UpsertInput{ProjectID: "project-1", Status: &badStatus},
			err:   crisis.ErrInvalidStatus,
		},
		"project_invalid": {
			input: crisis.UpsertInput{ProjectID: "project-1"},
			mock:  mock{projectDetail: mockProjectDetail{isCalled: true, input: "project-1", err: project.ErrNotFound}},
			err:   crisis.ErrProjectInvalid,
		},
		"repo_error": {
			input: crisis.UpsertInput{ProjectID: "project-1"},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoUpsert:    mockRepoUpsert{isCalled: true, input: repository.UpsertOptions{ProjectID: "project-1"}, err: errors.New("repo error")},
			},
			err: crisis.ErrUpsertFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.projectDetail.isCalled {
				deps.projectUC.EXPECT().Detail(ctx, tc.mock.projectDetail.input).Return(tc.mock.projectDetail.output, tc.mock.projectDetail.err)
			}
			if tc.mock.repoUpsert.isCalled {
				deps.repo.EXPECT().Upsert(ctx, tc.mock.repoUpsert.input).Return(tc.mock.repoUpsert.output, tc.mock.repoUpsert.err)
			}

			output, err := uc.Upsert(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDetail(t *testing.T) {
	ctx := context.Background()

	type mockRepoDetail struct {
		isCalled bool
		input    string
		output   model.CrisisConfig
		err      error
	}
	type mock struct {
		repo mockRepoDetail
	}

	tcs := map[string]struct {
		input  string
		mock   mock
		output crisis.DetailOutput
		err    error
	}{
		"success": {
			input:  "project-1",
			mock:   mock{repo: mockRepoDetail{isCalled: true, input: "project-1", output: model.CrisisConfig{ProjectID: "project-1"}}},
			output: crisis.DetailOutput{CrisisConfig: model.CrisisConfig{ProjectID: "project-1"}},
		},
		"empty_project_id": {
			err: crisis.ErrNotFound,
		},
		"repo_error": {
			input: "project-1",
			mock:  mock{repo: mockRepoDetail{isCalled: true, input: "project-1", err: repository.ErrFailedToGet}},
			err:   crisis.ErrNotFound,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repo.isCalled {
				deps.repo.EXPECT().Detail(ctx, tc.mock.repo.input).Return(tc.mock.repo.output, tc.mock.repo.err)
			}

			output, err := uc.Detail(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	type mockRepoDelete struct {
		isCalled bool
		input    string
		err      error
	}
	type mock struct {
		repo mockRepoDelete
	}

	tcs := map[string]struct {
		input  string
		mock   mock
		output error
		err    error
	}{
		"success": {
			input: "project-1",
			mock:  mock{repo: mockRepoDelete{isCalled: true, input: "project-1"}},
		},
		"empty_project_id": {
			err: crisis.ErrNotFound,
		},
		"repo_not_found": {
			input: "project-1",
			mock:  mock{repo: mockRepoDelete{isCalled: true, input: "project-1", err: repository.ErrFailedToGet}},
			err:   crisis.ErrNotFound,
		},
		"repo_error": {
			input: "project-1",
			mock:  mock{repo: mockRepoDelete{isCalled: true, input: "project-1", err: repository.ErrFailedToDelete}},
			err:   crisis.ErrDeleteFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repo.isCalled {
				deps.repo.EXPECT().Delete(ctx, tc.mock.repo.input).Return(tc.mock.repo.err)
			}

			err := uc.Delete(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestApplyRuntime(t *testing.T) {
	ctx := context.Background()
	warning := model.CrisisStatusWarning
	lowerCritical := model.CrisisStatus(" critical ")

	type mockProjectDetail struct {
		isCalled bool
		input    string
		output   project.DetailOutput
		err      error
	}
	type mockRepoDetail struct {
		isCalled bool
		input    string
		output   model.CrisisConfig
		err      error
	}
	type mockIngestUpdate struct {
		isCalled bool
		input    microservice.UpdateProjectCrawlModeInput
		output   microservice.UpdateProjectCrawlModeOutput
		err      error
	}
	type mock struct {
		projectDetail mockProjectDetail
		repoDetail    mockRepoDetail
		ingestUpdate  mockIngestUpdate
		ingestNil     bool
	}

	tcs := map[string]struct {
		input  crisis.ApplyRuntimeInput
		mock   mock
		output crisis.ApplyRuntimeOutput
		err    error
	}{
		"success_status_from_detail_default_reason_normal": {
			input: crisis.ApplyRuntimeInput{ProjectID: " project-1 "},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoDetail:    mockRepoDetail{isCalled: true, input: "project-1", output: model.CrisisConfig{ProjectID: "project-1", Status: model.CrisisStatusNormal}},
				ingestUpdate: mockIngestUpdate{
					isCalled: true,
					input:    microservice.UpdateProjectCrawlModeInput{ProjectID: "project-1", CrawlMode: "NORMAL", TriggerType: "CRISIS_EVENT", Reason: "crisis runtime apply status=NORMAL"},
					output:   microservice.UpdateProjectCrawlModeOutput{ProjectID: "project-1", AffectedDataSourceCount: 2},
				},
			},
			output: crisis.ApplyRuntimeOutput{ProjectID: "project-1", CrisisStatus: model.CrisisStatusNormal, AppliedCrawlMode: "NORMAL", AffectedDataSourceCount: 2},
		},
		"success_input_status_trimmed_reason_event_ref_crisis": {
			input: crisis.ApplyRuntimeInput{ProjectID: "project-1", Status: &lowerCritical, Reason: " reason ", EventRef: " event-1 "},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoDetail:    mockRepoDetail{isCalled: true, input: "project-1", output: model.CrisisConfig{ProjectID: "project-1", Status: model.CrisisStatusNormal}},
				ingestUpdate: mockIngestUpdate{
					isCalled: true,
					input:    microservice.UpdateProjectCrawlModeInput{ProjectID: "project-1", CrawlMode: "CRISIS", TriggerType: "CRISIS_EVENT", Reason: "reason", EventRef: "event-1"},
					output:   microservice.UpdateProjectCrawlModeOutput{ProjectID: "project-1", AffectedDataSourceCount: 3},
				},
			},
			output: crisis.ApplyRuntimeOutput{ProjectID: "project-1", CrisisStatus: model.CrisisStatusCritical, AppliedCrawlMode: "CRISIS", AffectedDataSourceCount: 3},
		},
		"empty_project_id": {
			err: crisis.ErrProjectInvalid,
		},
		"project_invalid": {
			input: crisis.ApplyRuntimeInput{ProjectID: "project-1"},
			mock:  mock{projectDetail: mockProjectDetail{isCalled: true, input: "project-1", err: project.ErrNotFound}},
			err:   crisis.ErrProjectInvalid,
		},
		"detail_error": {
			input: crisis.ApplyRuntimeInput{ProjectID: "project-1"},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoDetail:    mockRepoDetail{isCalled: true, input: "project-1", err: repository.ErrFailedToGet},
			},
			err: crisis.ErrNotFound,
		},
		"invalid_detail_status": {
			input: crisis.ApplyRuntimeInput{ProjectID: "project-1"},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoDetail:    mockRepoDetail{isCalled: true, input: "project-1", output: model.CrisisConfig{ProjectID: "project-1", Status: model.CrisisStatus("BAD")}},
			},
			err: crisis.ErrInvalidStatus,
		},
		"invalid_input_status": {
			input: crisis.ApplyRuntimeInput{ProjectID: "project-1", Status: func() *model.CrisisStatus { s := model.CrisisStatus("BAD"); return &s }()},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoDetail:    mockRepoDetail{isCalled: true, input: "project-1", output: model.CrisisConfig{ProjectID: "project-1", Status: model.CrisisStatusNormal}},
			},
			err: crisis.ErrInvalidStatus,
		},
		"nil_ingest": {
			input: crisis.ApplyRuntimeInput{ProjectID: "project-1", Status: &warning},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoDetail:    mockRepoDetail{isCalled: true, input: "project-1", output: model.CrisisConfig{ProjectID: "project-1", Status: model.CrisisStatusNormal}},
				ingestNil:     true,
			},
			err: crisis.ErrApplyFailed,
		},
		"ingest_bad_request": {
			input: crisis.ApplyRuntimeInput{ProjectID: "project-1", Status: &warning},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoDetail:    mockRepoDetail{isCalled: true, input: "project-1", output: model.CrisisConfig{ProjectID: "project-1", Status: model.CrisisStatusNormal}},
				ingestUpdate:  mockIngestUpdate{isCalled: true, input: microservice.UpdateProjectCrawlModeInput{ProjectID: "project-1", CrawlMode: "CRISIS", TriggerType: "CRISIS_EVENT", Reason: "crisis runtime apply status=WARNING"}, err: microservice.ErrBadRequest},
			},
			err: crisis.ErrInvalidStatus,
		},
		"ingest_generic_error": {
			input: crisis.ApplyRuntimeInput{ProjectID: "project-1", Status: &warning},
			mock: mock{
				projectDetail: mockProjectDetail{isCalled: true, input: "project-1", output: project.DetailOutput{Project: model.Project{ID: "project-1"}}},
				repoDetail:    mockRepoDetail{isCalled: true, input: "project-1", output: model.CrisisConfig{ProjectID: "project-1", Status: model.CrisisStatusNormal}},
				ingestUpdate:  mockIngestUpdate{isCalled: true, input: microservice.UpdateProjectCrawlModeInput{ProjectID: "project-1", CrawlMode: "CRISIS", TriggerType: "CRISIS_EVENT", Reason: "crisis runtime apply status=WARNING"}, err: microservice.ErrRequestFailed},
			},
			err: crisis.ErrApplyFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.ingestNil {
				uc.ingest = nil
			}
			if tc.mock.projectDetail.isCalled {
				deps.projectUC.EXPECT().Detail(ctx, tc.mock.projectDetail.input).Return(tc.mock.projectDetail.output, tc.mock.projectDetail.err)
			}
			if tc.mock.repoDetail.isCalled {
				deps.repo.EXPECT().Detail(ctx, tc.mock.repoDetail.input).Return(tc.mock.repoDetail.output, tc.mock.repoDetail.err)
			}
			if tc.mock.ingestUpdate.isCalled {
				deps.ingest.EXPECT().UpdateProjectCrawlMode(ctx, tc.mock.ingestUpdate.input).Return(tc.mock.ingestUpdate.output, tc.mock.ingestUpdate.err)
			}

			output, err := uc.ApplyRuntime(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}
