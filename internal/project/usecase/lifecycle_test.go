package usecase

import (
	"context"
	"errors"
	"testing"

	"project-srv/internal/model"
	"project-srv/internal/project"
	"project-srv/internal/project/repository"
	"project-srv/pkg/microservice"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestActivate(t *testing.T) {
	ctx := userContext("user-1")

	type mockRepoDetail struct {
		output model.Project
		err    error
	}
	type mockReadiness struct {
		output microservice.ActivationReadiness
		err    error
	}
	type mockIngestActivate struct {
		isCalled bool
		err      error
	}
	type mockRepoUpdateStatus struct {
		isCalled bool
		err      error
		output   model.Project
	}
	type mockPublisher struct {
		isCalled bool
		err      error
	}
	type mockData struct {
		repoDetail        mockRepoDetail
		readiness         mockReadiness
		ingestActivate    mockIngestActivate
		repoUpdateStatus  mockRepoUpdateStatus
		publisher         mockPublisher
		nilIngest         bool
		nilAfterReadiness bool
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output project.ActivateOutput
		err    error
	}{
		"success": {
			input: "project-1",
			mock: mockData{
				repoDetail:       mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}},
				readiness:        mockReadiness{output: microservice.ActivationReadiness{CanProceed: true}},
				ingestActivate:   mockIngestActivate{isCalled: true},
				repoUpdateStatus: mockRepoUpdateStatus{isCalled: true, output: model.Project{ID: "project-1", Status: model.ProjectStatusActive}},
				publisher:        mockPublisher{isCalled: true},
			},
			output: project.ActivateOutput{Project: model.Project{ID: "project-1", Status: model.ProjectStatusActive}},
		},
		"detail_error": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{err: repository.ErrNotFound}},
			err:   project.ErrNotFound,
		},
		"status_not_allowed": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusActive}}},
			err:   project.ErrActivateNotAllowed,
		},
		"readiness_error": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}}, readiness: mockReadiness{err: microservice.ErrForbidden}},
			err:   project.ErrLifecycleManagerForbidden,
		},
		"readiness_blocked": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}}, readiness: mockReadiness{output: microservice.ActivationReadiness{CanProceed: false, Errors: ingestReadiness().Errors}}},
			err:   project.ErrReadinessDatasourceRequired,
		},
		"nil_ingest": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}}, readiness: mockReadiness{output: microservice.ActivationReadiness{CanProceed: true}}, nilIngest: true},
			err:   project.ErrLifecycleManagerFailed,
		},
		"nil_ingest_after_readiness": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}}, readiness: mockReadiness{output: microservice.ActivationReadiness{CanProceed: true}}, nilAfterReadiness: true},
			err:   project.ErrLifecycleManagerFailed,
		},
		"ingest_error": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}}, readiness: mockReadiness{output: microservice.ActivationReadiness{CanProceed: true}}, ingestActivate: mockIngestActivate{isCalled: true, err: microservice.ErrUnauthorized}},
			err:   project.ErrLifecycleManagerUnauthorized,
		},
		"repo_not_found": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}}, readiness: mockReadiness{output: microservice.ActivationReadiness{CanProceed: true}}, ingestActivate: mockIngestActivate{isCalled: true}, repoUpdateStatus: mockRepoUpdateStatus{isCalled: true, err: repository.ErrNotFound}},
			err:   project.ErrNotFound,
		},
		"status_conflict": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}}, readiness: mockReadiness{output: microservice.ActivationReadiness{CanProceed: true}}, ingestActivate: mockIngestActivate{isCalled: true}, repoUpdateStatus: mockRepoUpdateStatus{isCalled: true, err: repository.ErrStatusConflict}},
			err:   project.ErrActivateNotAllowed,
		},
		"repo_error": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}}, readiness: mockReadiness{output: microservice.ActivationReadiness{CanProceed: true}}, ingestActivate: mockIngestActivate{isCalled: true}, repoUpdateStatus: mockRepoUpdateStatus{isCalled: true, err: repository.ErrFailedToUpdate}},
			err:   project.ErrUpdateFailed,
		},
		"publisher_error_ignored": {
			input: "project-1",
			mock: mockData{
				repoDetail:       mockRepoDetail{output: model.Project{ID: "project-1", Status: model.ProjectStatusPending}},
				readiness:        mockReadiness{output: microservice.ActivationReadiness{CanProceed: true}},
				ingestActivate:   mockIngestActivate{isCalled: true},
				repoUpdateStatus: mockRepoUpdateStatus{isCalled: true, output: model.Project{ID: "project-1", Status: model.ProjectStatusActive}},
				publisher:        mockPublisher{isCalled: true, err: errors.New("publish error")},
			},
			output: project.ActivateOutput{Project: model.Project{ID: "project-1", Status: model.ProjectStatusActive}},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.nilIngest {
				uc.ingest = nil
			}
			deps.repo.EXPECT().Detail(ctx, "project-1").Return(tc.mock.repoDetail.output, tc.mock.repoDetail.err)
			if tc.mock.repoDetail.err == nil && tc.mock.repoDetail.output.Status == model.ProjectStatusPending {
				expect := deps.ingest.EXPECT().GetActivationReadiness(ctx, mock.Anything)
				if tc.mock.nilAfterReadiness {
					expect.Run(func(context.Context, microservice.ActivationReadinessInput) {
						uc.ingest = nil
					})
				}
				expect.Return(tc.mock.readiness.output, tc.mock.readiness.err).Maybe()
			}
			if tc.mock.ingestActivate.isCalled {
				deps.ingest.EXPECT().Activate(ctx, "project-1").Return(tc.mock.ingestActivate.err)
			}
			if tc.mock.repoUpdateStatus.isCalled {
				deps.repo.EXPECT().UpdateStatus(ctx, repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive), ExpectedStatuses: []string{string(model.ProjectStatusPending)}}).Return(tc.mock.repoUpdateStatus.output, tc.mock.repoUpdateStatus.err)
			}
			if tc.mock.publisher.isCalled {
				deps.publisher.EXPECT().PublishLifecycleEvent(ctx, mock.Anything).Return(tc.mock.publisher.err)
			}

			output, err := uc.Activate(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestPause(t *testing.T) {
	testSimpleLifecycle(t, "pause")
}

func TestArchive(t *testing.T) {
	testSimpleLifecycle(t, "archive")
}

func TestUnarchive(t *testing.T) {
	testSimpleLifecycle(t, "unarchive")
}

func testSimpleLifecycle(t *testing.T, method string) {
	t.Helper()
	ctx := userContext("user-1")

	type mockData struct {
		detailErr       error
		nilIngest       bool
		ingestErr       error
		updateErr       error
		publisherErr    error
		expectIngest    bool
		expectedNewStat model.ProjectStatus
	}

	tcs := map[string]struct {
		input  string
		status model.ProjectStatus
		mock   mockData
		output model.Project
		err    error
	}{
		"success": {
			input: "project-1",
			status: map[string]model.ProjectStatus{
				"pause":     model.ProjectStatusActive,
				"archive":   model.ProjectStatusPending,
				"unarchive": model.ProjectStatusArchived,
			}[method],
			mock:   mockData{expectIngest: method == "pause", expectedNewStat: map[string]model.ProjectStatus{"pause": model.ProjectStatusPaused, "archive": model.ProjectStatusArchived, "unarchive": model.ProjectStatusPaused}[method]},
			output: model.Project{ID: "project-1", Status: map[string]model.ProjectStatus{"pause": model.ProjectStatusPaused, "archive": model.ProjectStatusArchived, "unarchive": model.ProjectStatusPaused}[method]},
		},
		"publisher_error_ignored": {
			input: "project-1",
			status: map[string]model.ProjectStatus{
				"pause":     model.ProjectStatusActive,
				"archive":   model.ProjectStatusPending,
				"unarchive": model.ProjectStatusArchived,
			}[method],
			mock: mockData{
				expectIngest:    method == "pause",
				expectedNewStat: map[string]model.ProjectStatus{"pause": model.ProjectStatusPaused, "archive": model.ProjectStatusArchived, "unarchive": model.ProjectStatusPaused}[method],
				publisherErr:    errors.New("publish error"),
			},
			output: model.Project{ID: "project-1", Status: map[string]model.ProjectStatus{"pause": model.ProjectStatusPaused, "archive": model.ProjectStatusArchived, "unarchive": model.ProjectStatusPaused}[method]},
		},
		"archive_active_success": {
			input:  "project-1",
			status: model.ProjectStatusActive,
			mock:   mockData{expectIngest: method == "archive", expectedNewStat: model.ProjectStatusArchived},
			output: model.Project{ID: "project-1", Status: model.ProjectStatusArchived},
		},
		"detail_error": {
			input:  "project-1",
			status: model.ProjectStatusPending,
			mock:   mockData{detailErr: repository.ErrNotFound},
			err:    project.ErrNotFound,
		},
		"not_allowed": {
			input:  "project-1",
			status: map[string]model.ProjectStatus{"pause": model.ProjectStatusPending, "archive": model.ProjectStatusArchived, "unarchive": model.ProjectStatusPending}[method],
			err:    map[string]error{"pause": project.ErrPauseNotAllowed, "archive": project.ErrInvalidTransition, "unarchive": project.ErrUnarchiveNotAllowed}[method],
		},
		"nil_ingest": {
			input:  "project-1",
			status: model.ProjectStatusActive,
			mock:   mockData{nilIngest: true, expectIngest: method == "pause" || method == "archive"},
			err: func() error {
				if method == "unarchive" {
					return nil
				}
				return project.ErrLifecycleManagerFailed
			}(),
		},
		"ingest_error": {
			input:  "project-1",
			status: model.ProjectStatusActive,
			mock:   mockData{expectIngest: method == "pause" || method == "archive", ingestErr: microservice.ErrBadRequest},
			err: func() error {
				if method == "unarchive" {
					return nil
				}
				return project.ErrLifecycleManagerRejected
			}(),
		},
		"repo_not_found": {
			input:  "project-1",
			status: map[string]model.ProjectStatus{"pause": model.ProjectStatusActive, "archive": model.ProjectStatusPending, "unarchive": model.ProjectStatusArchived}[method],
			mock:   mockData{expectIngest: method == "pause", updateErr: repository.ErrNotFound, expectedNewStat: map[string]model.ProjectStatus{"pause": model.ProjectStatusPaused, "archive": model.ProjectStatusArchived, "unarchive": model.ProjectStatusPaused}[method]},
			err:    project.ErrNotFound,
		},
		"repo_error": {
			input:  "project-1",
			status: map[string]model.ProjectStatus{"pause": model.ProjectStatusActive, "archive": model.ProjectStatusPending, "unarchive": model.ProjectStatusArchived}[method],
			mock:   mockData{expectIngest: method == "pause", updateErr: repository.ErrFailedToUpdate, expectedNewStat: map[string]model.ProjectStatus{"pause": model.ProjectStatusPaused, "archive": model.ProjectStatusArchived, "unarchive": model.ProjectStatusPaused}[method]},
			err:    project.ErrUpdateFailed,
		},
		"status_conflict_pause": {
			input:  "project-1",
			status: model.ProjectStatusActive,
			mock:   mockData{expectIngest: method == "pause", updateErr: repository.ErrStatusConflict, expectedNewStat: model.ProjectStatusPaused},
			err: func() error {
				if method == "pause" {
					return project.ErrPauseNotAllowed
				}
				return nil
			}(),
		},
	}

	for name, tc := range tcs {
		if method != "archive" && name == "archive_active_success" {
			continue
		}
		if method != "pause" && name == "status_conflict_pause" {
			continue
		}
		if method == "unarchive" && (name == "nil_ingest" || name == "ingest_error") {
			continue
		}
		if tc.err == nil && tc.output.ID == "" && name != "nil_ingest" && name != "ingest_error" && name != "status_conflict_pause" {
			continue
		}
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.nilIngest {
				uc.ingest = nil
			}
			deps.repo.EXPECT().Detail(ctx, "project-1").Return(model.Project{ID: "project-1", Status: tc.status}, tc.mock.detailErr)
			transitionAllowed := (method == "pause" && tc.status == model.ProjectStatusActive) ||
				(method == "archive" && (tc.status == model.ProjectStatusPending || tc.status == model.ProjectStatusActive || tc.status == model.ProjectStatusPaused)) ||
				(method == "unarchive" && tc.status == model.ProjectStatusArchived)
			shouldCallIngest := tc.mock.expectIngest && !tc.mock.nilIngest && tc.mock.detailErr == nil && transitionAllowed
			if shouldCallIngest {
				if method == "pause" || method == "archive" {
					deps.ingest.EXPECT().Pause(ctx, "project-1").Return(tc.mock.ingestErr)
				}
			}
			shouldUpdate := tc.mock.detailErr == nil && transitionAllowed && !tc.mock.nilIngest && tc.mock.ingestErr == nil
			if shouldUpdate {
				status := string(tc.mock.expectedNewStat)
				opts := repository.UpdateStatusOptions{ID: "project-1", Status: status}
				if method == "pause" {
					opts.ExpectedStatuses = []string{string(model.ProjectStatusActive)}
				}
				deps.repo.EXPECT().UpdateStatus(ctx, opts).Return(model.Project{ID: "project-1", Status: tc.mock.expectedNewStat}, tc.mock.updateErr)
			}
			if shouldUpdate && tc.mock.updateErr == nil {
				deps.publisher.EXPECT().PublishLifecycleEvent(ctx, mock.Anything).Return(tc.mock.publisherErr).Maybe()
			}

			var outputProject model.Project
			var err error
			switch method {
			case "pause":
				var output project.PauseOutput
				output, err = uc.Pause(ctx, tc.input)
				outputProject = output.Project
			case "archive":
				var output project.ArchiveOutput
				output, err = uc.Archive(ctx, tc.input)
				outputProject = output.Project
			case "unarchive":
				var output project.UnarchiveOutput
				output, err = uc.Unarchive(ctx, tc.input)
				outputProject = output.Project
			}

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, outputProject)
		})
	}
}

func TestResume(t *testing.T) {
	ctx := userContext("user-1")

	tcs := map[string]struct {
		input string
		mock  struct {
			detailOutput model.Project
			detailErr    error
			readiness    microservice.ActivationReadiness
			readinessErr error
			nilIngest    bool
			resumeErr    error
			updateErr    error
		}
		output project.ResumeOutput
		err    error
	}{
		"success": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: true}},
			output: project.ResumeOutput{Project: model.Project{ID: "project-1", Status: model.ProjectStatusActive}},
		},
		"not_allowed": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusActive}},
			err: project.ErrResumeNotAllowed,
		},
		"detail_error": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailErr: repository.ErrNotFound},
			err: project.ErrNotFound,
		},
		"readiness_error": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readinessErr: microservice.ErrUnauthorized},
			err: project.ErrLifecycleManagerUnauthorized,
		},
		"readiness_blocked": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: false, Errors: ingestReadiness().Errors}},
			err: project.ErrReadinessDatasourceRequired,
		},
		"nil_ingest": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: true}, nilIngest: true},
			err: project.ErrLifecycleManagerFailed,
		},
		"nil_ingest_after_readiness": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: true}},
			err: project.ErrLifecycleManagerFailed,
		},
		"resume_error": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: true}, resumeErr: microservice.ErrForbidden},
			err: project.ErrLifecycleManagerForbidden,
		},
		"status_conflict": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: true}, updateErr: repository.ErrStatusConflict},
			err: project.ErrResumeNotAllowed,
		},
		"repo_not_found": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: true}, updateErr: repository.ErrNotFound},
			err: project.ErrNotFound,
		},
		"repo_error": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: true}, updateErr: repository.ErrFailedToUpdate},
			err: project.ErrUpdateFailed,
		},
		"publisher_error_ignored": {
			input: "project-1",
			mock: struct {
				detailOutput         model.Project
				detailErr            error
				readiness            microservice.ActivationReadiness
				readinessErr         error
				nilIngest            bool
				resumeErr, updateErr error
			}{detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused}, readiness: microservice.ActivationReadiness{CanProceed: true}},
			output: project.ResumeOutput{Project: model.Project{ID: "project-1", Status: model.ProjectStatusActive}},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.nilIngest {
				uc.ingest = nil
			}
			deps.repo.EXPECT().Detail(ctx, "project-1").Return(tc.mock.detailOutput, tc.mock.detailErr)
			if tc.mock.detailErr == nil && tc.mock.detailOutput.Status == model.ProjectStatusPaused {
				expect := deps.ingest.EXPECT().GetActivationReadiness(ctx, mock.Anything)
				if name == "nil_ingest_after_readiness" {
					expect.Run(func(context.Context, microservice.ActivationReadinessInput) {
						uc.ingest = nil
					})
				}
				expect.Return(tc.mock.readiness, tc.mock.readinessErr).Maybe()
			}
			if tc.mock.readiness.CanProceed && !tc.mock.nilIngest {
				deps.ingest.EXPECT().Resume(ctx, "project-1").Return(tc.mock.resumeErr).Maybe()
			}
			if tc.mock.readiness.CanProceed && !tc.mock.nilIngest && tc.mock.resumeErr == nil {
				deps.repo.EXPECT().UpdateStatus(ctx, repository.UpdateStatusOptions{ID: "project-1", Status: string(model.ProjectStatusActive), ExpectedStatuses: []string{string(model.ProjectStatusPaused)}}).Return(model.Project{ID: "project-1", Status: model.ProjectStatusActive}, tc.mock.updateErr).Maybe()
			}
			if tc.mock.readiness.CanProceed && !tc.mock.nilIngest && tc.mock.resumeErr == nil && tc.mock.updateErr == nil {
				publisherErr := error(nil)
				if name == "publisher_error_ignored" {
					publisherErr = errors.New("publish error")
				}
				deps.publisher.EXPECT().PublishLifecycleEvent(ctx, mock.Anything).Return(publisherErr).Maybe()
			}

			output, err := uc.Resume(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestGetActivationReadiness(t *testing.T) {
	ctx := userContext("user-1")

	tcs := map[string]struct {
		input project.ActivationReadinessInput
		mock  struct {
			detailOutput model.Project
			detailErr    error
			nilIngest    bool
			ingestInput  microservice.ActivationReadinessInput
			ingestOutput microservice.ActivationReadiness
			ingestErr    error
		}
		output project.ActivationReadiness
		err    error
	}{
		"success": {
			input: project.ActivationReadinessInput{ProjectID: "project-1", Command: project.ActivationReadinessCommandResume},
			mock: struct {
				detailOutput model.Project
				detailErr    error
				nilIngest    bool
				ingestInput  microservice.ActivationReadinessInput
				ingestOutput microservice.ActivationReadiness
				ingestErr    error
			}{
				detailOutput: model.Project{ID: "project-1", Status: model.ProjectStatusPaused},
				ingestInput:  microservice.ActivationReadinessInput{ProjectID: "project-1", Command: microservice.ActivationReadinessCommandResume},
				ingestOutput: ingestReadiness(),
			},
			output: project.ActivationReadiness{
				ProjectID:       "project-1",
				ProjectStatus:   model.ProjectStatusPaused,
				DataSourceCount: 1,
				HasDatasource:   true,
				CanProceed:      true,
				Errors:          []project.ActivationReadinessError{{Code: project.ActivationReadinessCodeDatasourceRequired, Message: "missing datasource"}},
			},
		},
		"detail_error": {
			input: project.ActivationReadinessInput{ProjectID: "project-1"},
			mock: struct {
				detailOutput model.Project
				detailErr    error
				nilIngest    bool
				ingestInput  microservice.ActivationReadinessInput
				ingestOutput microservice.ActivationReadiness
				ingestErr    error
			}{detailErr: repository.ErrNotFound},
			err: project.ErrNotFound,
		},
		"nil_ingest": {
			input: project.ActivationReadinessInput{ProjectID: "project-1"},
			mock: struct {
				detailOutput model.Project
				detailErr    error
				nilIngest    bool
				ingestInput  microservice.ActivationReadinessInput
				ingestOutput microservice.ActivationReadiness
				ingestErr    error
			}{detailOutput: model.Project{ID: "project-1"}, nilIngest: true},
			err: project.ErrLifecycleManagerFailed,
		},
		"ingest_error": {
			input: project.ActivationReadinessInput{ProjectID: "project-1"},
			mock: struct {
				detailOutput model.Project
				detailErr    error
				nilIngest    bool
				ingestInput  microservice.ActivationReadinessInput
				ingestOutput microservice.ActivationReadiness
				ingestErr    error
			}{detailOutput: model.Project{ID: "project-1"}, ingestInput: microservice.ActivationReadinessInput{ProjectID: "project-1", Command: microservice.ActivationReadinessCommandActivate}, ingestErr: microservice.ErrBadRequest},
			err: project.ErrLifecycleManagerRejected,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.nilIngest {
				uc.ingest = nil
			}
			deps.repo.EXPECT().Detail(ctx, "project-1").Return(tc.mock.detailOutput, tc.mock.detailErr)
			if tc.mock.detailErr == nil && !tc.mock.nilIngest {
				deps.ingest.EXPECT().GetActivationReadiness(ctx, tc.mock.ingestInput).Return(tc.mock.ingestOutput, tc.mock.ingestErr)
			}

			output, err := uc.GetActivationReadiness(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}
