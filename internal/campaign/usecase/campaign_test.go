package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"project-srv/internal/campaign"
	"project-srv/internal/campaign/repository"
	"project-srv/internal/model"

	"github.com/golang-jwt/jwt"
	"github.com/smap-hcmut/shared-libs/go/auth"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/require"
)

type mockDeps struct {
	repo *repository.MockRepository
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
	uc := New(l, repo).(*implUseCase)

	return uc, mockDeps{repo: repo}
}

func userContext(userID string) context.Context {
	return auth.SetPayloadToContext(context.Background(), auth.Payload{
		UserID:         userID,
		StandardClaims: jwt.StandardClaims{Subject: userID},
	})
}

func TestFavoriteCampaignForUser(t *testing.T) {
	tcs := map[string]struct {
		input struct {
			item   model.Campaign
			userID string
		}
		mock   struct{}
		output model.Campaign
		err    error
	}{
		"empty_user": {
			input: struct {
				item   model.Campaign
				userID string
			}{item: model.Campaign{ID: "campaign-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}},
			output: model.Campaign{ID: "campaign-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: false},
		},
		"favorite": {
			input: struct {
				item   model.Campaign
				userID string
			}{item: model.Campaign{ID: "campaign-1", FavoriteUserIDs: []string{"user-1"}}, userID: "user-1"},
			output: model.Campaign{ID: "campaign-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true},
		},
		"not_favorite": {
			input: struct {
				item   model.Campaign
				userID string
			}{item: model.Campaign{ID: "campaign-1", FavoriteUserIDs: []string{"user-2"}}, userID: "user-1"},
			output: model.Campaign{ID: "campaign-1", FavoriteUserIDs: []string{"user-2"}, IsFavorite: false},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := favoriteCampaignForUser(tc.input.item, tc.input.userID)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestCreate(t *testing.T) {
	ctx := userContext("user-1")
	start := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	repoErr := errors.New("repo error")

	type mockRepoCreate struct {
		isCalled bool
		input    repository.CreateOptions
		output   model.Campaign
		err      error
	}
	type mock struct {
		repo mockRepoCreate
	}

	tcs := map[string]struct {
		input  campaign.CreateInput
		mock   mock
		output campaign.CreateOutput
		err    error
	}{
		"success": {
			input: campaign.CreateInput{
				Name:        "Campaign A",
				Description: "Description",
				StartDate:   start.Format(time.RFC3339),
				EndDate:     end.Format(time.RFC3339),
			},
			mock: mock{repo: mockRepoCreate{
				isCalled: true,
				input: repository.CreateOptions{
					Name:        "Campaign A",
					Description: "Description",
					StartDate:   &start,
					EndDate:     &end,
					CreatedBy:   "user-1",
				},
				output: model.Campaign{ID: "campaign-1", Name: "Campaign A", FavoriteUserIDs: []string{"user-1"}},
			}},
			output: campaign.CreateOutput{Campaign: model.Campaign{ID: "campaign-1", Name: "Campaign A", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}},
		},
		"name_required": {
			input: campaign.CreateInput{},
			err:   campaign.ErrNameRequired,
		},
		"invalid_start_date": {
			input: campaign.CreateInput{Name: "Campaign A", StartDate: "bad-date"},
			err:   campaign.ErrInvalidDateRange,
		},
		"invalid_end_date": {
			input: campaign.CreateInput{Name: "Campaign A", EndDate: "bad-date"},
			err:   campaign.ErrInvalidDateRange,
		},
		"start_after_end": {
			input: campaign.CreateInput{Name: "Campaign A", StartDate: end.Format(time.RFC3339), EndDate: start.Format(time.RFC3339)},
			err:   campaign.ErrInvalidDateRange,
		},
		"repo_error": {
			input: campaign.CreateInput{Name: "Campaign A"},
			mock: mock{repo: mockRepoCreate{
				isCalled: true,
				input:    repository.CreateOptions{Name: "Campaign A", CreatedBy: "user-1"},
				err:      repoErr,
			}},
			err: campaign.ErrCreateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repo.isCalled {
				deps.repo.EXPECT().Create(ctx, tc.mock.repo.input).Return(tc.mock.repo.output, tc.mock.repo.err)
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
		output   model.Campaign
		err      error
	}
	type mock struct {
		repo mockRepoDetail
	}

	tcs := map[string]struct {
		input  string
		mock   mock
		output campaign.DetailOutput
		err    error
	}{
		"success": {
			input:  "campaign-1",
			mock:   mock{repo: mockRepoDetail{isCalled: true, input: "campaign-1", output: model.Campaign{ID: "campaign-1", FavoriteUserIDs: []string{"user-1"}}}},
			output: campaign.DetailOutput{Campaign: model.Campaign{ID: "campaign-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}},
		},
		"empty_id": {
			err: campaign.ErrNotFound,
		},
		"repo_error": {
			input: "campaign-1",
			mock:  mock{repo: mockRepoDetail{isCalled: true, input: "campaign-1", err: repository.ErrFailedToGet}},
			err:   campaign.ErrNotFound,
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

func TestList(t *testing.T) {
	ctx := userContext("user-1")
	pagQuery := paginator.PaginateQuery{Page: 1, Limit: 10}
	pag := paginator.Paginator{Total: 1, Count: 1, PerPage: 10, CurrentPage: 1}
	repoErr := errors.New("repo error")

	type mockRepoGet struct {
		isCalled  bool
		input     repository.GetOptions
		output    []model.Campaign
		paginator paginator.Paginator
		err       error
	}
	type mock struct {
		repo mockRepoGet
	}

	tcs := map[string]struct {
		input  campaign.ListInput
		mock   mock
		output campaign.ListOutput
		err    error
	}{
		"success": {
			input: campaign.ListInput{Status: string(model.CampaignStatusActive), Name: "Campaign", FavoriteOnly: true, Sort: campaignSortCreatedAtDesc, Paginator: pagQuery},
			mock: mock{repo: mockRepoGet{
				isCalled:  true,
				input:     repository.GetOptions{Status: string(model.CampaignStatusActive), Name: "Campaign", FavoriteOnly: true, Sort: campaignSortCreatedAtDesc, CurrentUserID: "user-1", Paginator: pagQuery},
				output:    []model.Campaign{{ID: "campaign-1", FavoriteUserIDs: []string{"user-1"}}},
				paginator: pag,
			}},
			output: campaign.ListOutput{Campaigns: []model.Campaign{{ID: "campaign-1", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}}, Paginator: pag},
		},
		"invalid_status": {
			input: campaign.ListInput{Status: "DONE"},
			err:   campaign.ErrInvalidStatus,
		},
		"invalid_sort": {
			input: campaign.ListInput{Sort: "bad"},
			err:   campaign.ErrInvalidSort,
		},
		"repo_error": {
			input: campaign.ListInput{},
			mock:  mock{repo: mockRepoGet{isCalled: true, input: repository.GetOptions{CurrentUserID: "user-1"}, err: repoErr}},
			err:   campaign.ErrListFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repo.isCalled {
				deps.repo.EXPECT().Get(ctx, tc.mock.repo.input).Return(tc.mock.repo.output, tc.mock.repo.paginator, tc.mock.repo.err)
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
	start := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	type mockRepoUpdate struct {
		isCalled bool
		input    repository.UpdateOptions
		output   model.Campaign
		err      error
	}
	type mock struct {
		repo mockRepoUpdate
	}

	tcs := map[string]struct {
		input  campaign.UpdateInput
		mock   mock
		output campaign.UpdateOutput
		err    error
	}{
		"success": {
			input: campaign.UpdateInput{ID: "campaign-1", Name: "New Name", Status: string(model.CampaignStatusActive), StartDate: start.Format(time.RFC3339), EndDate: end.Format(time.RFC3339)},
			mock: mock{repo: mockRepoUpdate{
				isCalled: true,
				input:    repository.UpdateOptions{ID: "campaign-1", Name: "New Name", Status: string(model.CampaignStatusActive), StartDate: &start, EndDate: &end},
				output:   model.Campaign{ID: "campaign-1", Name: "New Name", FavoriteUserIDs: []string{"user-1"}},
			}},
			output: campaign.UpdateOutput{Campaign: model.Campaign{ID: "campaign-1", Name: "New Name", FavoriteUserIDs: []string{"user-1"}, IsFavorite: true}},
		},
		"empty_id": {
			input: campaign.UpdateInput{},
			err:   campaign.ErrNotFound,
		},
		"invalid_status": {
			input: campaign.UpdateInput{ID: "campaign-1", Status: "DONE"},
			err:   campaign.ErrInvalidStatus,
		},
		"invalid_start_date": {
			input: campaign.UpdateInput{ID: "campaign-1", StartDate: "bad-date"},
			err:   campaign.ErrInvalidDateRange,
		},
		"invalid_end_date": {
			input: campaign.UpdateInput{ID: "campaign-1", EndDate: "bad-date"},
			err:   campaign.ErrInvalidDateRange,
		},
		"start_after_end": {
			input: campaign.UpdateInput{ID: "campaign-1", StartDate: end.Format(time.RFC3339), EndDate: start.Format(time.RFC3339)},
			err:   campaign.ErrInvalidDateRange,
		},
		"repo_not_found": {
			input: campaign.UpdateInput{ID: "campaign-1"},
			mock:  mock{repo: mockRepoUpdate{isCalled: true, input: repository.UpdateOptions{ID: "campaign-1"}, err: repository.ErrFailedToGet}},
			err:   campaign.ErrNotFound,
		},
		"repo_error": {
			input: campaign.UpdateInput{ID: "campaign-1"},
			mock:  mock{repo: mockRepoUpdate{isCalled: true, input: repository.UpdateOptions{ID: "campaign-1"}, err: repository.ErrFailedToUpdate}},
			err:   campaign.ErrUpdateFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repo.isCalled {
				deps.repo.EXPECT().Update(ctx, tc.mock.repo.input).Return(tc.mock.repo.output, tc.mock.repo.err)
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
			input: "campaign-1",
			mock:  mockRepoMutation{isCalled: true, id: "campaign-1", userID: "user-1"},
		},
		"empty_id": {
			err: campaign.ErrNotFound,
		},
		"missing_user": {
			input: "campaign-1",
			err:   campaign.ErrUpdateFailed,
		},
		"repo_not_found": {
			input: "campaign-1",
			mock:  mockRepoMutation{isCalled: true, id: "campaign-1", userID: "user-1", err: repository.ErrFailedToGet},
			err:   campaign.ErrNotFound,
		},
		"repo_error": {
			input: "campaign-1",
			mock:  mockRepoMutation{isCalled: true, id: "campaign-1", userID: "user-1", err: repository.ErrFailedToUpdate},
			err:   campaign.ErrUpdateFailed,
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

func TestArchive(t *testing.T) {
	ctx := context.Background()

	type mockRepoArchive struct {
		isCalled bool
		input    string
		err      error
	}
	type mock struct {
		repo mockRepoArchive
	}

	tcs := map[string]struct {
		input  string
		mock   mock
		output error
		err    error
	}{
		"success": {
			input: "campaign-1",
			mock:  mock{repo: mockRepoArchive{isCalled: true, input: "campaign-1"}},
		},
		"empty_id": {
			err: campaign.ErrNotFound,
		},
		"repo_not_found": {
			input: "campaign-1",
			mock:  mock{repo: mockRepoArchive{isCalled: true, input: "campaign-1", err: repository.ErrFailedToGet}},
			err:   campaign.ErrNotFound,
		},
		"repo_error": {
			input: "campaign-1",
			mock:  mock{repo: mockRepoArchive{isCalled: true, input: "campaign-1", err: repository.ErrFailedToDelete}},
			err:   campaign.ErrDeleteFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc, deps := initUseCase(t)
			if tc.mock.repo.isCalled {
				deps.repo.EXPECT().Archive(ctx, tc.mock.repo.input).Return(tc.mock.repo.err)
			}

			err := uc.Archive(ctx, tc.input)

			if tc.err != nil {
				require.EqualError(t, err, tc.err.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}
