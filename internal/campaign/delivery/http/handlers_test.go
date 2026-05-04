package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-srv/internal/campaign"
	"project-srv/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(t *testing.T) (*handler, *campaign.MockUseCase) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{
		Level:        log.LevelFatal,
		Mode:         log.ModeProduction,
		Encoding:     log.EncodingJSON,
		ColorEnabled: false,
	})
	uc := campaign.NewMockUseCase(t)
	return New(l, uc, nil).(*handler), uc
}

func newTestContext(method, target, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Params = params
	return c, w
}

func TestCreate(t *testing.T) {
	type mockCreate struct {
		isCalled bool
		input    campaign.CreateInput
		output   campaign.CreateOutput
		err      error
	}
	type mockData struct {
		create mockCreate
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: `{"name":"Campaign A","description":"Desc","start_date":"2026-01-01T00:00:00Z","end_date":"2026-02-01T00:00:00Z"}`,
			mock: mockData{create: mockCreate{
				isCalled: true,
				input:    campaign.CreateInput{Name: "Campaign A", Description: "Desc", StartDate: "2026-01-01T00:00:00Z", EndDate: "2026-02-01T00:00:00Z"},
				output:   campaign.CreateOutput{Campaign: model.Campaign{ID: "campaign-1", Name: "Campaign A", Status: model.CampaignStatusPending}},
			}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			output: http.StatusBadRequest,
		},
		"wrong_body_validate": {
			input:  `{}`,
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: `{"name":"Campaign A"}`,
			mock: mockData{create: mockCreate{
				isCalled: true,
				input:    campaign.CreateInput{Name: "Campaign A"},
				err:      campaign.ErrCreateFailed,
			}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.create.isCalled {
				uc.EXPECT().Create(context.Background(), tc.mock.create.input).Return(tc.mock.create.output, tc.mock.create.err)
			}
			c, w := newTestContext(http.MethodPost, "/campaigns", tc.input, nil)

			h.Create(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestDetail(t *testing.T) {
	type mockDetail struct {
		isCalled bool
		input    string
		output   campaign.DetailOutput
		err      error
	}
	type mockData struct {
		detail mockDetail
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  "campaign-1",
			mock:   mockData{detail: mockDetail{isCalled: true, input: "campaign-1", output: campaign.DetailOutput{Campaign: model.Campaign{ID: "campaign-1"}}}},
			output: http.StatusOK,
		},
		"uc_error": {
			input:  "campaign-1",
			mock:   mockData{detail: mockDetail{isCalled: true, input: "campaign-1", err: campaign.ErrNotFound}},
			output: http.StatusNotFound,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.detail.isCalled {
				uc.EXPECT().Detail(context.Background(), tc.mock.detail.input).Return(tc.mock.detail.output, tc.mock.detail.err)
			}
			c, w := newTestContext(http.MethodGet, "/campaigns/"+tc.input, "", gin.Params{{Key: "id", Value: tc.input}})

			h.Detail(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestList(t *testing.T) {
	type mockList struct {
		isCalled bool
		input    campaign.ListInput
		output   campaign.ListOutput
		err      error
	}
	type mockData struct {
		list mockList
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: "?status=PENDING&name=A&sort=created_at_desc&page=1&limit=10",
			mock: mockData{list: mockList{
				isCalled: true,
				input:    campaign.ListInput{Status: "PENDING", Name: "A", Sort: "created_at_desc", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				output:   campaign.ListOutput{Campaigns: []model.Campaign{{ID: "campaign-1"}}, Paginator: paginator.Paginator{Total: 1}},
			}},
			output: http.StatusOK,
		},
		"wrong_query": {
			input:  "?sort=bad",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: "?page=1&limit=10",
			mock: mockData{list: mockList{
				isCalled: true,
				input:    campaign.ListInput{Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				err:      campaign.ErrListFailed,
			}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.list.isCalled {
				uc.EXPECT().List(context.Background(), tc.mock.list.input).Return(tc.mock.list.output, tc.mock.list.err)
			}
			c, w := newTestContext(http.MethodGet, "/campaigns"+tc.input, "", nil)

			h.List(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestListFavorites(t *testing.T) {
	type mockList struct {
		isCalled bool
		input    campaign.ListInput
		output   campaign.ListOutput
		err      error
	}
	type mockData struct {
		list mockList
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: "?sort=favorite_desc&page=1&limit=10",
			mock: mockData{list: mockList{
				isCalled: true,
				input:    campaign.ListInput{FavoriteOnly: true, Sort: "favorite_desc", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				output:   campaign.ListOutput{Campaigns: []model.Campaign{{ID: "campaign-1"}}, Paginator: paginator.Paginator{Total: 1}},
			}},
			output: http.StatusOK,
		},
		"wrong_query": {
			input:  "?sort=bad",
			output: http.StatusBadRequest,
		},
		"wrong_query_bind": {
			input:  "?page=bad",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: "?page=1&limit=10",
			mock: mockData{list: mockList{
				isCalled: true,
				input:    campaign.ListInput{FavoriteOnly: true, Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				err:      campaign.ErrListFailed,
			}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.list.isCalled {
				uc.EXPECT().List(context.Background(), tc.mock.list.input).Return(tc.mock.list.output, tc.mock.list.err)
			}
			c, w := newTestContext(http.MethodGet, "/campaigns/favorites"+tc.input, "", nil)

			h.ListFavorites(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestUpdate(t *testing.T) {
	type mockUpdate struct {
		isCalled bool
		input    campaign.UpdateInput
		output   campaign.UpdateOutput
		err      error
	}
	type mockData struct {
		update mockUpdate
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: `{"name":"Campaign B","status":"ACTIVE"}`,
			mock: mockData{update: mockUpdate{
				isCalled: true,
				input:    campaign.UpdateInput{ID: "campaign-1", Name: "Campaign B", Status: "ACTIVE"},
				output:   campaign.UpdateOutput{Campaign: model.Campaign{ID: "campaign-1", Name: "Campaign B", Status: model.CampaignStatusActive}},
			}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{"status":"BAD"}`,
			output: http.StatusBadRequest,
		},
		"wrong_body_bind": {
			input:  `{`,
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: `{"name":"Campaign B"}`,
			mock: mockData{update: mockUpdate{
				isCalled: true,
				input:    campaign.UpdateInput{ID: "campaign-1", Name: "Campaign B"},
				err:      campaign.ErrUpdateFailed,
			}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.update.isCalled {
				uc.EXPECT().Update(context.Background(), tc.mock.update.input).Return(tc.mock.update.output, tc.mock.update.err)
			}
			c, w := newTestContext(http.MethodPut, "/campaigns/campaign-1", tc.input, gin.Params{{Key: "id", Value: "campaign-1"}})

			h.Update(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestFavorite(t *testing.T) {
	testIDCommand(t, "favorite", func(h *handler, c *gin.Context) { h.Favorite(c) }, func(uc *campaign.MockUseCase, ctx interface{}, id string, err error) {
		uc.EXPECT().Favorite(ctx, id).Return(err)
	})
}

func TestUnfavorite(t *testing.T) {
	testIDCommand(t, "unfavorite", func(h *handler, c *gin.Context) { h.Unfavorite(c) }, func(uc *campaign.MockUseCase, ctx interface{}, id string, err error) {
		uc.EXPECT().Unfavorite(ctx, id).Return(err)
	})
}

func TestArchive(t *testing.T) {
	testIDCommand(t, "archive", func(h *handler, c *gin.Context) { h.Archive(c) }, func(uc *campaign.MockUseCase, ctx interface{}, id string, err error) {
		uc.EXPECT().Archive(ctx, id).Return(err)
	})
}

func testIDCommand(t *testing.T, name string, call func(*handler, *gin.Context), expect func(*campaign.MockUseCase, interface{}, string, error)) {
	t.Helper()

	tcs := map[string]struct {
		input  string
		mock   struct{ err error }
		output int
		err    error
	}{
		"success": {
			input:  "campaign-1",
			output: http.StatusOK,
		},
		"uc_error": {
			input:  "campaign-1",
			mock:   struct{ err error }{err: campaign.ErrNotFound},
			output: http.StatusNotFound,
		},
	}

	for caseName, tc := range tcs {
		t.Run(name+"_"+caseName, func(t *testing.T) {
			h, uc := newTestHandler(t)
			c, w := newTestContext(http.MethodPost, "/campaigns/"+tc.input+"/"+name, "", gin.Params{{Key: "id", Value: tc.input}})
			expect(uc, context.Background(), tc.input, tc.mock.err)

			call(h, c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}
