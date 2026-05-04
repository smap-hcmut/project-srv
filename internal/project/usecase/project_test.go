package usecase

import (
	"context"
	"errors"
	"testing"

	"project-srv/internal/campaign"
	"project-srv/internal/domain"
	"project-srv/internal/model"
	"project-srv/internal/project"
	projectproducer "project-srv/internal/project/delivery/kafka/producer"
	"project-srv/internal/project/repository"
	"project-srv/pkg/microservice"

	"github.com/golang-jwt/jwt"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/contracts"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDeps struct {
	repo       *repository.MockRepository
	domainRepo *domain.MockRepository
	campaignUC *campaign.MockUseCase
	ingest     *microservice.MockIngestUseCase
	publisher  *projectproducer.MockProducer
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
	domainRepo := domain.NewMockRepository(t)
	campaignUC := campaign.NewMockUseCase(t)
	ingest := microservice.NewMockIngestUseCase(t)
	publisher := projectproducer.NewMockProducer(t)
	uc := New(l, repo, domainRepo, campaignUC, ingest, publisher).(*implUseCase)

	return uc, mockDeps{
		repo:       repo,
		domainRepo: domainRepo,
		campaignUC: campaignUC,
		ingest:     ingest,
		publisher:  publisher,
	}
}

func userContext(userID string) context.Context {
	return auth.SetPayloadToContext(context.Background(), auth.Payload{
		UserID:         userID,
		StandardClaims: jwt.StandardClaims{Subject: userID},
	})
}

func TestFavoriteProjectForUser(t *testing.T) {
	tcs := map[string]struct {
		input struct {
			item   model.Project
			userID string
		}
		mock   struct{}
		output model.Project
		err    error
	}{
		"empty_user": {
			input: struct {
				item   model.Project
				userID string
			}{item: model.Project{ID: "project-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}},
			output: model.Project{ID: "project-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: false},
		},
		"favorite": {
			input: struct {
				item   model.Project
				userID string
			}{item: model.Project{ID: "project-1", FavoriteUserIDs: []string{"user-1"}}, userID: "user-1"},
			output: model.Project{ID: "project-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true},
		},
		"not_favorite": {
			input: struct {
				item   model.Project
				userID string
			}{item: model.Project{ID: "project-1", FavoriteUserIDs: []string{"user-2"}}, userID: "user-1"},
			output: model.Project{ID: "project-1", FavoriteUserIDs: []string{"user-2"}, IsFavorite: false},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _ := initUseCase(t)

			output := uc.favoriteProjectForUser(tc.input.item, tc.input.userID)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := userContext("user-1")
	repoErr := errors.New("repo error")

	type mockDomainExists struct {
		isCalled bool
		input    string
		output   bool
		err      error
	}
	type mockCampaignDetail struct {
		isCalled bool
		input    string
		output   campaign.DetailOutput
		err      error
	}
	type mockRepoCreate struct {
		isCalled bool
		input    repository.CreateOptions
		output   model.Project
		err      error
	}
	type mockData struct {
		domainExists   mockDomainExists
		campaignDetail mockCampaignDetail
		repoCreate     mockRepoCreate
	}

	tcs := map[string]struct {
		input  project.CreateInput
		mock   mockData
		output project.CreateOutput
		err    error
	}{
		"success": {
			input: project.CreateInput{CampaignID: "campaign-1", Name: "Project A", Description: "Desc", Brand: "Brand", EntityType: string(model.EntityTypeProduct), EntityName: "Product", DomainTypeCode: "retail"},
			mock: mockData{
				domainExists:   mockDomainExists{isCalled: true, input: "retail", output: true},
				campaignDetail: mockCampaignDetail{isCalled: true, input: "campaign-1", output: campaign.DetailOutput{Campaign: model.Campaign{ID: "campaign-1"}}},
				repoCreate: mockRepoCreate{
					isCalled: true,
					input: repository.CreateOptions{
						CampaignID:     "campaign-1",
						Name:           "Project A",
						Description:    "Desc",
						Brand:          "Brand",
						EntityType:     string(model.EntityTypeProduct),
						EntityName:     "Product",
						DomainTypeCode: "retail",
						CreatedBy:      "user-1",
					},
					output: model.Project{ID: "project-1", Name: "Project A", FavoriteUserIDs: []string{"user-1"}},
				},
			},
			output: project.CreateOutput{Project: model.Project{ID: "project-1", Name: "Project A", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}},
		},
		"campaign_required": {
			input: project.CreateInput{Name: "Project A", DomainTypeCode: "retail"},
			err:   project.ErrCampaignRequired,
		},
		"name_required": {
			input: project.CreateInput{CampaignID: "campaign-1", DomainTypeCode: "retail"},
			err:   project.ErrNameRequired,
		},
		"invalid_entity": {
			input: project.CreateInput{CampaignID: "campaign-1", Name: "Project A", EntityType: "bad", DomainTypeCode: "retail"},
			err:   project.ErrInvalidEntity,
		},
		"domain_required": {
			input: project.CreateInput{CampaignID: "campaign-1", Name: "Project A"},
			err:   project.ErrDomainTypeRequired,
		},
		"domain_repo_error": {
			input: project.CreateInput{CampaignID: "campaign-1", Name: "Project A", DomainTypeCode: "retail"},
			mock:  mockData{domainExists: mockDomainExists{isCalled: true, input: "retail", err: errors.New("redis error")}},
			err:   project.ErrCreateFailed,
		},
		"invalid_domain": {
			input: project.CreateInput{CampaignID: "campaign-1", Name: "Project A", DomainTypeCode: "retail"},
			mock:  mockData{domainExists: mockDomainExists{isCalled: true, input: "retail", output: false}},
			err:   project.ErrInvalidDomainType,
		},
		"campaign_not_found": {
			input: project.CreateInput{CampaignID: "campaign-1", Name: "Project A", DomainTypeCode: "retail"},
			mock: mockData{
				domainExists:   mockDomainExists{isCalled: true, input: "retail", output: true},
				campaignDetail: mockCampaignDetail{isCalled: true, input: "campaign-1", err: campaign.ErrNotFound},
			},
			err: project.ErrCampaignNotFound,
		},
		"repo_error": {
			input: project.CreateInput{CampaignID: "campaign-1", Name: "Project A", DomainTypeCode: "retail"},
			mock: mockData{
				domainExists:   mockDomainExists{isCalled: true, input: "retail", output: true},
				campaignDetail: mockCampaignDetail{isCalled: true, input: "campaign-1", output: campaign.DetailOutput{}},
				repoCreate:     mockRepoCreate{isCalled: true, input: repository.CreateOptions{CampaignID: "campaign-1", Name: "Project A", DomainTypeCode: "retail", CreatedBy: "user-1"}, err: repoErr},
			},
			err: project.ErrCreateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.domainExists.isCalled {
				deps.domainRepo.EXPECT().Exists(ctx, tc.mock.domainExists.input).Return(tc.mock.domainExists.output, tc.mock.domainExists.err)
			}
			if tc.mock.campaignDetail.isCalled {
				deps.campaignUC.EXPECT().Detail(ctx, tc.mock.campaignDetail.input).Return(tc.mock.campaignDetail.output, tc.mock.campaignDetail.err)
			}
			if tc.mock.repoCreate.isCalled {
				deps.repo.EXPECT().Create(ctx, tc.mock.repoCreate.input).Return(tc.mock.repoCreate.output, tc.mock.repoCreate.err)
			}

			output, err := uc.Create(ctx, tc.input)

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
	ctx := userContext("user-1")

	type mockRepoDetail struct {
		isCalled bool
		input    string
		output   model.Project
		err      error
	}
	type mockData struct {
		repoDetail mockRepoDetail
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output project.DetailOutput
		err    error
	}{
		"success": {
			input:  " project-1 ",
			mock:   mockData{repoDetail: mockRepoDetail{isCalled: true, input: "project-1", output: model.Project{ID: "project-1", FavoriteUserIDs: []string{"user-1"}}}},
			output: project.DetailOutput{Project: model.Project{ID: "project-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}},
		},
		"empty_id": {
			err: project.ErrNotFound,
		},
		"repo_not_found": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{isCalled: true, input: "project-1", err: repository.ErrNotFound}},
			err:   project.ErrNotFound,
		},
		"repo_error": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{isCalled: true, input: "project-1", err: repository.ErrFailedToGet}},
			err:   project.ErrDetailFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repoDetail.isCalled {
				deps.repo.EXPECT().Detail(ctx, tc.mock.repoDetail.input).Return(tc.mock.repoDetail.output, tc.mock.repoDetail.err)
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

func TestList(t *testing.T) {
	ctx := userContext("user-1")
	pagQuery := paginator.PaginateQuery{Page: 1, Limit: 10}
	pag := paginator.Paginator{Total: 1, Count: 1, PerPage: 10, CurrentPage: 1}

	type mockRepoGet struct {
		isCalled  bool
		input     repository.GetOptions
		output    []model.Project
		paginator paginator.Paginator
		err       error
	}
	type mockData struct {
		repoGet mockRepoGet
	}

	tcs := map[string]struct {
		input  project.ListInput
		mock   mockData
		output project.ListOutput
		err    error
	}{
		"success": {
			input: project.ListInput{CampaignID: "campaign-1", Status: string(model.ProjectStatusActive), Name: "Project", Brand: "Brand", EntityType: string(model.EntityTypeProduct), FavoriteOnly: true, Sort: projectSortCreatedAtDesc, Paginator: pagQuery},
			mock: mockData{repoGet: mockRepoGet{
				isCalled: true,
				input: repository.GetOptions{
					CampaignID:    "campaign-1",
					Status:        string(model.ProjectStatusActive),
					Name:          "Project",
					Brand:         "Brand",
					EntityType:    string(model.EntityTypeProduct),
					FavoriteOnly:  true,
					Sort:          projectSortCreatedAtDesc,
					CurrentUserID: "user-1",
					Paginator:     pagQuery,
				},
				output:    []model.Project{{ID: "project-1", FavoriteUserIDs: []string{"user-1"}}},
				paginator: pag,
			}},
			output: project.ListOutput{Projects: []model.Project{{ID: "project-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}}, Paginator: pag},
		},
		"invalid_status": {
			input: project.ListInput{Status: "DONE"},
			err:   project.ErrInvalidStatus,
		},
		"invalid_entity": {
			input: project.ListInput{EntityType: "bad"},
			err:   project.ErrInvalidEntity,
		},
		"invalid_sort": {
			input: project.ListInput{Sort: "bad"},
			err:   project.ErrInvalidSort,
		},
		"repo_error": {
			mock: mockData{repoGet: mockRepoGet{isCalled: true, input: repository.GetOptions{CurrentUserID: "user-1"}, err: errors.New("repo error")}},
			err:  project.ErrListFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repoGet.isCalled {
				deps.repo.EXPECT().Get(ctx, tc.mock.repoGet.input).Return(tc.mock.repoGet.output, tc.mock.repoGet.paginator, tc.mock.repoGet.err)
			}

			output, err := uc.List(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestUpdate(t *testing.T) {
	ctx := userContext("user-1")

	type mockDomainExists struct {
		isCalled bool
		input    string
		output   bool
		err      error
	}
	type mockRepoUpdate struct {
		isCalled bool
		input    repository.UpdateOptions
		output   model.Project
		err      error
	}
	type mockData struct {
		domainExists mockDomainExists
		repoUpdate   mockRepoUpdate
	}

	tcs := map[string]struct {
		input  project.UpdateInput
		mock   mockData
		output project.UpdateOutput
		err    error
	}{
		"success": {
			input: project.UpdateInput{ID: "project-1", Name: "Project B", EntityType: string(model.EntityTypeService), DomainTypeCode: "retail"},
			mock: mockData{
				domainExists: mockDomainExists{isCalled: true, input: "retail", output: true},
				repoUpdate: mockRepoUpdate{
					isCalled: true,
					input:    repository.UpdateOptions{ID: "project-1", Name: "Project B", EntityType: string(model.EntityTypeService), DomainTypeCode: "retail"},
					output:   model.Project{ID: "project-1", Name: "Project B", FavoriteUserIDs: []string{"user-1"}},
				},
			},
			output: project.UpdateOutput{Project: model.Project{ID: "project-1", Name: "Project B", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}},
		},
		"empty_id": {
			err: project.ErrNotFound,
		},
		"invalid_entity": {
			input: project.UpdateInput{ID: "project-1", EntityType: "bad"},
			err:   project.ErrInvalidEntity,
		},
		"domain_error": {
			input: project.UpdateInput{ID: "project-1", DomainTypeCode: "retail"},
			mock:  mockData{domainExists: mockDomainExists{isCalled: true, input: "retail", err: errors.New("redis error")}},
			err:   project.ErrUpdateFailed,
		},
		"invalid_domain": {
			input: project.UpdateInput{ID: "project-1", DomainTypeCode: "retail"},
			mock:  mockData{domainExists: mockDomainExists{isCalled: true, input: "retail", output: false}},
			err:   project.ErrInvalidDomainType,
		},
		"repo_not_found": {
			input: project.UpdateInput{ID: "project-1"},
			mock:  mockData{repoUpdate: mockRepoUpdate{isCalled: true, input: repository.UpdateOptions{ID: "project-1"}, err: repository.ErrNotFound}},
			err:   project.ErrNotFound,
		},
		"repo_error": {
			input: project.UpdateInput{ID: "project-1"},
			mock:  mockData{repoUpdate: mockRepoUpdate{isCalled: true, input: repository.UpdateOptions{ID: "project-1"}, err: repository.ErrFailedToUpdate}},
			err:   project.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.domainExists.isCalled {
				deps.domainRepo.EXPECT().Exists(ctx, tc.mock.domainExists.input).Return(tc.mock.domainExists.output, tc.mock.domainExists.err)
			}
			if tc.mock.repoUpdate.isCalled {
				deps.repo.EXPECT().Update(ctx, tc.mock.repoUpdate.input).Return(tc.mock.repoUpdate.output, tc.mock.repoUpdate.err)
			}

			output, err := uc.Update(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestFavorite(t *testing.T) {
	testFavoriteMutation(t, "favorite")
}

func TestUnfavorite(t *testing.T) {
	testFavoriteMutation(t, "unfavorite")
}

func testFavoriteMutation(t *testing.T, method string) {
	t.Helper()
	ctx := userContext("user-1")

	type mockRepoMutation struct {
		isCalled bool
		id       string
		userID   string
		err      error
	}

	tcs := map[string]struct {
		input  string
		mock   mockRepoMutation
		output error
		err    error
	}{
		"success": {
			input: "project-1",
			mock:  mockRepoMutation{isCalled: true, id: "project-1", userID: "user-1"},
		},
		"empty_id": {
			err: project.ErrNotFound,
		},
		"missing_user": {
			input: "project-1",
			err:   project.ErrUpdateFailed,
		},
		"repo_not_found": {
			input: "project-1",
			mock:  mockRepoMutation{isCalled: true, id: "project-1", userID: "user-1", err: repository.ErrNotFound},
			err:   project.ErrNotFound,
		},
		"repo_error": {
			input: "project-1",
			mock:  mockRepoMutation{isCalled: true, id: "project-1", userID: "user-1", err: repository.ErrFailedToUpdate},
			err:   project.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			callCtx := ctx
			if name == "missing_user" {
				callCtx = context.Background()
			}
			if tc.mock.isCalled {
				if method == "favorite" {
					deps.repo.EXPECT().Favorite(callCtx, tc.mock.id, tc.mock.userID).Return(tc.mock.err)
				} else {
					deps.repo.EXPECT().Unfavorite(callCtx, tc.mock.id, tc.mock.userID).Return(tc.mock.err)
				}
			}

			var err error
			if method == "favorite" {
				err = uc.Favorite(callCtx, tc.input)
			} else {
				err = uc.Unfavorite(callCtx, tc.input)
			}

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	type mockRepoDetail struct {
		isCalled bool
		input    string
		output   model.Project
		err      error
	}
	type mockRepoArchive struct {
		isCalled bool
		input    string
		err      error
	}
	type mockData struct {
		repoDetail  mockRepoDetail
		repoArchive mockRepoArchive
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output error
		err    error
	}{
		"success": {
			input: "project-1",
			mock: mockData{
				repoDetail:  mockRepoDetail{isCalled: true, input: "project-1", output: model.Project{ID: "project-1", Status: model.ProjectStatusArchived}},
				repoArchive: mockRepoArchive{isCalled: true, input: "project-1"},
			},
		},
		"empty_id": {
			err: project.ErrNotFound,
		},
		"detail_not_found": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{isCalled: true, input: "project-1", err: repository.ErrNotFound}},
			err:   project.ErrNotFound,
		},
		"detail_error": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{isCalled: true, input: "project-1", err: repository.ErrFailedToGet}},
			err:   project.ErrDeleteFailed,
		},
		"not_archived": {
			input: "project-1",
			mock:  mockData{repoDetail: mockRepoDetail{isCalled: true, input: "project-1", output: model.Project{ID: "project-1", Status: model.ProjectStatusActive}}},
			err:   project.ErrDeleteRequiresArchived,
		},
		"archive_not_found": {
			input: "project-1",
			mock: mockData{
				repoDetail:  mockRepoDetail{isCalled: true, input: "project-1", output: model.Project{ID: "project-1", Status: model.ProjectStatusArchived}},
				repoArchive: mockRepoArchive{isCalled: true, input: "project-1", err: repository.ErrNotFound},
			},
			err: project.ErrNotFound,
		},
		"archive_error": {
			input: "project-1",
			mock: mockData{
				repoDetail:  mockRepoDetail{isCalled: true, input: "project-1", output: model.Project{ID: "project-1", Status: model.ProjectStatusArchived}},
				repoArchive: mockRepoArchive{isCalled: true, input: "project-1", err: repository.ErrFailedToDelete},
			},
			err: project.ErrDeleteFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repoDetail.isCalled {
				deps.repo.EXPECT().Detail(ctx, tc.mock.repoDetail.input).Return(tc.mock.repoDetail.output, tc.mock.repoDetail.err)
			}
			if tc.mock.repoArchive.isCalled {
				deps.repo.EXPECT().Archive(ctx, tc.mock.repoArchive.input).Return(tc.mock.repoArchive.err)
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

func TestListDomains(t *testing.T) {
	ctx := context.Background()

	type mockDomainListActive struct {
		isCalled bool
		output   []domain.Domain
		err      error
	}
	type mockData struct {
		domainListActive mockDomainListActive
	}

	tcs := map[string]struct {
		input  struct{}
		mock   mockData
		output []domain.Domain
		err    error
	}{
		"success": {
			mock:   mockData{domainListActive: mockDomainListActive{isCalled: true, output: []domain.Domain{{DomainCode: "retail", DisplayName: "Retail"}}}},
			output: []domain.Domain{{DomainCode: "retail", DisplayName: "Retail"}},
		},
		"repo_error": {
			mock: mockData{domainListActive: mockDomainListActive{isCalled: true, err: errors.New("redis error")}},
			err:  project.ErrListFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.domainListActive.isCalled {
				deps.domainRepo.EXPECT().ListActive(ctx).Return(tc.mock.domainListActive.output, tc.mock.domainListActive.err)
			}

			output, err := uc.ListDomains(ctx)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestPublishLifecycleEvent(t *testing.T) {
	ctx := context.Background()
	eventProject := model.Project{ID: "project-1", CampaignID: "campaign-1", Status: model.ProjectStatusActive}

	type mockPublisher struct {
		isCalled bool
		err      error
	}
	type mockData struct {
		publisher mockPublisher
	}

	tcs := map[string]struct {
		input  project.LifecycleEventName
		mock   mockData
		output error
		err    error
	}{
		"success": {
			input: project.ProjectLifecycleEventActivated,
			mock:  mockData{publisher: mockPublisher{isCalled: true}},
		},
		"nil_publisher": {
			input: project.ProjectLifecycleEventActivated,
			err:   errors.New("project lifecycle event publisher is nil"),
		},
		"publisher_error": {
			input: project.ProjectLifecycleEventActivated,
			mock:  mockData{publisher: mockPublisher{isCalled: true, err: errors.New("publish error")}},
			err:   errors.New("publish error"),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if !tc.mock.publisher.isCalled {
				uc.publisher = nil
			}
			if tc.mock.publisher.isCalled {
				deps.publisher.EXPECT().PublishLifecycleEvent(ctx, mock.MatchedBy(func(event project.LifecycleEvent) bool {
					return event.EventName == tc.input &&
						event.ProjectID == eventProject.ID &&
						event.CampaignID == eventProject.CampaignID &&
						event.Status == string(eventProject.Status) &&
						event.OccurredAt.IsZero() == false
				})).Return(tc.mock.publisher.err)
			}

			err := uc.publishLifecycleEvent(ctx, tc.input, eventProject)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMapReadinessBlockedError(t *testing.T) {
	tcs := map[string]struct {
		input  project.ActivationReadiness
		mock   struct{}
		output error
		err    error
	}{
		"no_errors": {
			output: project.ErrReadinessFailed,
		},
		"datasource_required": {
			input:  readinessWithCode(project.ActivationReadinessCodeDatasourceRequired),
			output: project.ErrReadinessDatasourceRequired,
		},
		"passive_unconfirmed": {
			input:  readinessWithCode(project.ActivationReadinessCodePassiveUnconfirmed),
			output: project.ErrReadinessPassiveUnconfirmed,
		},
		"target_dryrun_missing": {
			input:  readinessWithCode(project.ActivationReadinessCodeTargetDryrunMissing),
			output: project.ErrReadinessTargetDryrunMissing,
		},
		"target_dryrun_failed": {
			input:  readinessWithCode(project.ActivationReadinessCodeTargetDryrunFailed),
			output: project.ErrReadinessTargetDryrunFailed,
		},
		"active_target_required": {
			input:  readinessWithCode(project.ActivationReadinessCodeActiveTargetRequired),
			output: project.ErrReadinessActiveTargetMissing,
		},
		"datasource_status": {
			input:  readinessWithCode(project.ActivationReadinessCodeDatasourceStatus),
			output: project.ErrReadinessDatasourceStatus,
		},
		"unknown": {
			input:  readinessWithCode(project.ActivationReadinessCode("unknown")),
			output: project.ErrReadinessFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _ := initUseCase(t)

			output := uc.mapReadinessBlockedError(tc.input)

			require.Equal(t, tc.output, output)
			require.NoError(t, tc.err)
		})
	}
}

func TestNormalizeActivationReadinessCommand(t *testing.T) {
	tcs := map[string]struct {
		input  project.ActivationReadinessCommand
		mock   struct{}
		output project.ActivationReadinessCommand
		err    error
	}{
		"resume": {
			input:  project.ActivationReadinessCommandResume,
			output: project.ActivationReadinessCommandResume,
		},
		"default_activate": {
			input:  project.ActivationReadinessCommand(""),
			output: project.ActivationReadinessCommandActivate,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, _ := initUseCase(t)

			output := uc.normalizeActivationReadinessCommand(tc.input)

			require.Equal(t, tc.output, output)
			require.NoError(t, tc.err)
		})
	}
}

func readinessWithCode(code project.ActivationReadinessCode) project.ActivationReadiness {
	return project.ActivationReadiness{Errors: []project.ActivationReadinessError{{Code: code}}}
}

func activeReadiness() project.ActivationReadiness {
	return project.ActivationReadiness{ProjectID: "project-1", CanProceed: true, HasDatasource: true, DataSourceCount: 1}
}

func ingestReadiness() microservice.ActivationReadiness {
	return microservice.ActivationReadiness{
		ProjectID:       "project-1",
		CanProceed:      true,
		HasDatasource:   true,
		DataSourceCount: 1,
		Errors: []contracts.ActivationReadinessError{{
			Code:    contracts.ReadinessCodeDatasourceRequired,
			Message: "missing datasource",
		}},
	}
}
