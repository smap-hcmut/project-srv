package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-srv/internal/domain"
	"project-srv/internal/model"
	"project-srv/internal/project"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/paginator"
	"github.com/stretchr/testify/require"
)

const (
	testCampaignID = "550e8400-e29b-41d4-a716-446655440000"
	testProjectID  = "550e8400-e29b-41d4-a716-446655440002"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(t *testing.T) (*handler, *project.MockUseCase) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{
		Level:        log.LevelFatal,
		Mode:         log.ModeProduction,
		Encoding:     log.EncodingJSON,
		ColorEnabled: false,
	})
	uc := project.NewMockUseCase(t)
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
		input    project.CreateInput
		output   project.CreateOutput
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
			input: `{"name":"Project A","description":"Desc","brand":"Brand","entity_type":"product","entity_name":"VF8","domain_type_code":"ev"}`,
			mock: mockData{create: mockCreate{
				isCalled: true,
				input:    project.CreateInput{CampaignID: testCampaignID, Name: "Project A", Description: "Desc", Brand: "Brand", EntityType: "product", EntityName: "VF8", DomainTypeCode: "ev"},
				output:   project.CreateOutput{Project: model.Project{ID: testProjectID, CampaignID: testCampaignID, Name: "Project A"}},
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
			input: `{"name":"Project A","entity_type":"product","entity_name":"VF8","domain_type_code":"ev"}`,
			mock: mockData{create: mockCreate{
				isCalled: true,
				input:    project.CreateInput{CampaignID: testCampaignID, Name: "Project A", EntityType: "product", EntityName: "VF8", DomainTypeCode: "ev"},
				err:      project.ErrCreateFailed,
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
			c, w := newTestContext(http.MethodPost, "/campaigns/"+testCampaignID+"/projects", tc.input, gin.Params{{Key: "id", Value: testCampaignID}})

			h.Create(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestProcessCreateReq(t *testing.T) {
	tcs := map[string]struct {
		input struct {
			body       string
			campaignID string
		}
		mock   struct{}
		output createReq
		err    error
	}{
		"validate_error": {
			input: struct {
				body       string
				campaignID string
			}{
				body:       `{"name":"Project A","entity_type":"product","entity_name":"VF8","domain_type_code":"ev"}`,
				campaignID: "bad-id",
			},
			err: errWrongBody,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, _ := newTestHandler(t)
			c, _ := newTestContext(http.MethodPost, "/campaigns/"+tc.input.campaignID+"/projects", tc.input.body, gin.Params{{Key: "id", Value: tc.input.campaignID}})

			output, err := h.processCreateReq(c)

			if tc.err != nil {
				require.Equal(t, tc.err, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestDetail(t *testing.T) {
	testDetailCommand(t, "detail", func(h *handler, c *gin.Context) { h.Detail(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Detail(context.Background(), id).Return(project.DetailOutput{Project: model.Project{ID: id}}, err)
	})
}

func TestInternalDetail(t *testing.T) {
	testDetailCommand(t, "internal_detail", func(h *handler, c *gin.Context) { h.InternalDetail(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Detail(context.Background(), id).Return(project.DetailOutput{Project: model.Project{ID: id}}, err)
	})
}

func testDetailCommand(t *testing.T, name string, call func(*handler, *gin.Context), expect func(*project.MockUseCase, string, error)) {
	t.Helper()

	tcs := map[string]struct {
		input  string
		mock   struct{ err error }
		output int
		err    error
	}{
		"success": {
			input:  testProjectID,
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  "bad-id",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input:  testProjectID,
			mock:   struct{ err error }{err: project.ErrNotFound},
			output: http.StatusNotFound,
		},
	}

	for caseName, tc := range tcs {
		t.Run(name+"_"+caseName, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.input == testProjectID {
				expect(uc, tc.input, tc.mock.err)
			}
			c, w := newTestContext(http.MethodGet, "/projects/"+tc.input, "", gin.Params{{Key: "project_id", Value: tc.input}})

			call(h, c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestList(t *testing.T) {
	type mockList struct {
		isCalled bool
		input    project.ListInput
		output   project.ListOutput
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
			input: "?status=PENDING&name=A&brand=B&entity_type=product&sort=created_at_desc&page=1&limit=10",
			mock: mockData{list: mockList{
				isCalled: true,
				input:    project.ListInput{CampaignID: testCampaignID, Status: "PENDING", Name: "A", Brand: "B", EntityType: "product", Sort: "created_at_desc", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				output:   project.ListOutput{Projects: []model.Project{{ID: testProjectID}}, Paginator: paginator.Paginator{Total: 1}},
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
				input:    project.ListInput{CampaignID: testCampaignID, Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				err:      project.ErrListFailed,
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
			c, w := newTestContext(http.MethodGet, "/campaigns/"+testCampaignID+"/projects"+tc.input, "", gin.Params{{Key: "id", Value: testCampaignID}})

			h.List(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestListFavorites(t *testing.T) {
	type mockList struct {
		isCalled bool
		input    project.ListInput
		output   project.ListOutput
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
			input: "?campaign_id=" + testCampaignID + "&sort=favorite_desc&page=1&limit=10",
			mock: mockData{list: mockList{
				isCalled: true,
				input:    project.ListInput{CampaignID: testCampaignID, FavoriteOnly: true, Sort: "favorite_desc", Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				output:   project.ListOutput{Projects: []model.Project{{ID: testProjectID}}, Paginator: paginator.Paginator{Total: 1}},
			}},
			output: http.StatusOK,
		},
		"wrong_query": {
			input:  "?campaign_id=bad-id",
			output: http.StatusBadRequest,
		},
		"wrong_query_bind": {
			input:  "?page=bad",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: "?campaign_id=" + testCampaignID + "&page=1&limit=10",
			mock: mockData{list: mockList{
				isCalled: true,
				input:    project.ListInput{CampaignID: testCampaignID, FavoriteOnly: true, Paginator: paginator.PaginateQuery{Page: 1, Limit: 10}},
				err:      project.ErrListFailed,
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
			c, w := newTestContext(http.MethodGet, "/projects/favorites"+tc.input, "", nil)

			h.ListFavorites(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestUpdate(t *testing.T) {
	type mockUpdate struct {
		isCalled bool
		input    project.UpdateInput
		output   project.UpdateOutput
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
			input: `{"name":"Project B","entity_type":"product","entity_name":"VF9","domain_type_code":"ev"}`,
			mock: mockData{update: mockUpdate{
				isCalled: true,
				input:    project.UpdateInput{ID: testProjectID, Name: "Project B", EntityType: "product", EntityName: "VF9", DomainTypeCode: "ev"},
				output:   project.UpdateOutput{Project: model.Project{ID: testProjectID, Name: "Project B"}},
			}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{"entity_type":"bad"}`,
			output: http.StatusBadRequest,
		},
		"wrong_body_bind": {
			input:  `{`,
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: `{"name":"Project B"}`,
			mock: mockData{update: mockUpdate{
				isCalled: true,
				input:    project.UpdateInput{ID: testProjectID, Name: "Project B"},
				err:      project.ErrUpdateFailed,
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
			c, w := newTestContext(http.MethodPut, "/projects/"+testProjectID, tc.input, gin.Params{{Key: "project_id", Value: testProjectID}})

			h.Update(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestFavorite(t *testing.T) {
	testIDCommand(t, "favorite", func(h *handler, c *gin.Context) { h.Favorite(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Favorite(context.Background(), id).Return(err)
	})
}

func TestUnfavorite(t *testing.T) {
	testIDCommand(t, "unfavorite", func(h *handler, c *gin.Context) { h.Unfavorite(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Unfavorite(context.Background(), id).Return(err)
	})
}

func TestDelete(t *testing.T) {
	testIDCommand(t, "delete", func(h *handler, c *gin.Context) { h.Delete(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Delete(context.Background(), id).Return(err)
	})
}

func testIDCommand(t *testing.T, name string, call func(*handler, *gin.Context), expect func(*project.MockUseCase, string, error)) {
	t.Helper()

	tcs := map[string]struct {
		input  string
		mock   struct{ err error }
		output int
		err    error
	}{
		"success": {
			input:  testProjectID,
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  "bad-id",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input:  testProjectID,
			mock:   struct{ err error }{err: project.ErrNotFound},
			output: http.StatusNotFound,
		},
	}

	for caseName, tc := range tcs {
		t.Run(name+"_"+caseName, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.input == testProjectID {
				expect(uc, tc.input, tc.mock.err)
			}
			c, w := newTestContext(http.MethodPost, "/projects/"+tc.input+"/"+name, "", gin.Params{{Key: "project_id", Value: tc.input}})

			call(h, c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestArchive(t *testing.T) {
	testLifecycleCommand(t, "archive", func(h *handler, c *gin.Context) { h.Archive(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Archive(context.Background(), id).Return(project.ArchiveOutput{Project: model.Project{ID: id, Status: model.ProjectStatusArchived}}, err)
	})
}

func TestActivate(t *testing.T) {
	testLifecycleCommand(t, "activate", func(h *handler, c *gin.Context) { h.Activate(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Activate(context.Background(), id).Return(project.ActivateOutput{Project: model.Project{ID: id, Status: model.ProjectStatusActive}}, err)
	})
}

func TestPause(t *testing.T) {
	testLifecycleCommand(t, "pause", func(h *handler, c *gin.Context) { h.Pause(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Pause(context.Background(), id).Return(project.PauseOutput{Project: model.Project{ID: id, Status: model.ProjectStatusPaused}}, err)
	})
}

func TestResume(t *testing.T) {
	testLifecycleCommand(t, "resume", func(h *handler, c *gin.Context) { h.Resume(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Resume(context.Background(), id).Return(project.ResumeOutput{Project: model.Project{ID: id, Status: model.ProjectStatusActive}}, err)
	})
}

func TestUnarchive(t *testing.T) {
	testLifecycleCommand(t, "unarchive", func(h *handler, c *gin.Context) { h.Unarchive(c) }, func(uc *project.MockUseCase, id string, err error) {
		uc.EXPECT().Unarchive(context.Background(), id).Return(project.UnarchiveOutput{Project: model.Project{ID: id, Status: model.ProjectStatusPaused}}, err)
	})
}

func testLifecycleCommand(t *testing.T, name string, call func(*handler, *gin.Context), expect func(*project.MockUseCase, string, error)) {
	t.Helper()

	tcs := map[string]struct {
		input  string
		mock   struct{ err error }
		output int
		err    error
	}{
		"success": {
			input:  testProjectID,
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  "bad-id",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input:  testProjectID,
			mock:   struct{ err error }{err: project.ErrInvalidTransition},
			output: http.StatusBadRequest,
		},
	}

	for caseName, tc := range tcs {
		t.Run(name+"_"+caseName, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.input == testProjectID {
				expect(uc, tc.input, tc.mock.err)
			}
			c, w := newTestContext(http.MethodPost, "/projects/"+tc.input+"/"+name, "", gin.Params{{Key: "project_id", Value: tc.input}})

			call(h, c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestActivationReadiness(t *testing.T) {
	type mockReadiness struct {
		isCalled bool
		input    project.ActivationReadinessInput
		output   project.ActivationReadiness
		err      error
	}
	type mockData struct {
		readiness mockReadiness
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: "?command=resume",
			mock: mockData{readiness: mockReadiness{
				isCalled: true,
				input:    project.ActivationReadinessInput{ProjectID: testProjectID, Command: project.ActivationReadinessCommandResume},
				output:   project.ActivationReadiness{ProjectID: testProjectID, CanProceed: true},
			}},
			output: http.StatusOK,
		},
		"wrong_query": {
			input:  "?command=bad",
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: "",
			mock: mockData{readiness: mockReadiness{
				isCalled: true,
				input:    project.ActivationReadinessInput{ProjectID: testProjectID},
				err:      project.ErrLifecycleManagerForbidden,
			}},
			output: http.StatusForbidden,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.readiness.isCalled {
				uc.EXPECT().GetActivationReadiness(context.Background(), tc.mock.readiness.input).Return(tc.mock.readiness.output, tc.mock.readiness.err)
			}
			c, w := newTestContext(http.MethodGet, "/projects/"+testProjectID+"/activation-readiness"+tc.input, "", gin.Params{{Key: "project_id", Value: testProjectID}})

			h.ActivationReadiness(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestListDomains(t *testing.T) {
	type mockListDomains struct {
		isCalled bool
		output   []domain.Domain
		err      error
	}
	type mockData struct {
		listDomains mockListDomains
	}

	tcs := map[string]struct {
		input  struct{}
		mock   mockData
		output int
		err    error
	}{
		"success": {
			mock:   mockData{listDomains: mockListDomains{isCalled: true, output: []domain.Domain{{DomainCode: "ev", DisplayName: "EV"}}}},
			output: http.StatusOK,
		},
		"uc_error": {
			mock:   mockData{listDomains: mockListDomains{isCalled: true, err: project.ErrListFailed}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.listDomains.isCalled {
				uc.EXPECT().ListDomains(context.Background()).Return(tc.mock.listDomains.output, tc.mock.listDomains.err)
			}
			c, w := newTestContext(http.MethodGet, "/domains", "", nil)

			h.ListDomains(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}
