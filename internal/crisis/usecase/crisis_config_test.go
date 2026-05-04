package usecase

import (
	"context"
	"errors"
	"testing"

	"project-srv/internal/crisis"
	"project-srv/internal/crisis/repository"
	"project-srv/internal/model"
	"project-srv/internal/project"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

type mockDeps struct {
	repo      *repository.MockRepository
	projectUC *project.MockUseCase
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
	uc := New(l, repo, projectUC).(*implUseCase)

	return uc, mockDeps{repo: repo, projectUC: projectUC}
}

func TestUpsert(t *testing.T) {
	ctx := context.Background()
	trigger := &model.KeywordsTrigger{Enabled: true, Logic: "OR", Groups: []model.KeywordGroup{{Name: "brand", Keywords: []string{"smap"}, Weight: 1}}}

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
