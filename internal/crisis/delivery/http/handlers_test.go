package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-srv/internal/crisis"
	"project-srv/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler(t *testing.T) (*handler, *crisis.MockUseCase) {
	t.Helper()
	l := log.NewZapLogger(log.ZapConfig{
		Level:        log.LevelFatal,
		Mode:         log.ModeProduction,
		Encoding:     log.EncodingJSON,
		ColorEnabled: false,
	})
	uc := crisis.NewMockUseCase(t)
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

func TestUpsert(t *testing.T) {
	type mockUpsert struct {
		isCalled bool
		input    crisis.UpsertInput
		output   crisis.UpsertOutput
		err      error
	}
	type mockData struct {
		upsert mockUpsert
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input: `{"keywords_trigger":{"enabled":true,"logic":"AND","groups":[{"name":"Pin","keywords":["pin"],"weight":10}]}}`,
			mock: mockData{upsert: mockUpsert{
				isCalled: true,
				input: crisis.UpsertInput{
					ProjectID: "project-1",
					KeywordsTrigger: &model.KeywordsTrigger{
						Enabled: true,
						Logic:   "AND",
						Groups:  []model.KeywordGroup{{Name: "Pin", Keywords: []string{"pin"}, Weight: 10}},
					},
				},
				output: crisis.UpsertOutput{CrisisConfig: model.CrisisConfig{ProjectID: "project-1"}},
			}},
			output: http.StatusOK,
		},
		"wrong_body": {
			input:  `{`,
			output: http.StatusBadRequest,
		},
		"invalid_trigger": {
			input:  `{"keywords_trigger":{"enabled":true,"logic":"AND","groups":[]}}`,
			output: http.StatusBadRequest,
		},
		"uc_error": {
			input: `{"volume_trigger":{"enabled":true,"metric":"MENTIONS","rules":[{"level":"CRITICAL","threshold_percent_growth":150,"comparison_window_hours":1,"baseline":"PREVIOUS_PERIOD"}]}}`,
			mock: mockData{upsert: mockUpsert{
				isCalled: true,
				input: crisis.UpsertInput{
					ProjectID: "project-1",
					VolumeTrigger: &model.VolumeTrigger{
						Enabled: true,
						Metric:  "MENTIONS",
						Rules:   []model.VolumeRule{{Level: "CRITICAL", ThresholdPercentGrowth: 150, ComparisonWindowHours: 1, Baseline: "PREVIOUS_PERIOD"}},
					},
				},
				err: crisis.ErrUpsertFailed,
			}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.upsert.isCalled {
				uc.EXPECT().Upsert(context.Background(), tc.mock.upsert.input).Return(tc.mock.upsert.output, tc.mock.upsert.err)
			}
			c, w := newTestContext(http.MethodPut, "/projects/project-1/crisis-config", tc.input, gin.Params{{Key: "project_id", Value: "project-1"}})

			h.Upsert(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestDetail(t *testing.T) {
	type mockDetail struct {
		isCalled bool
		input    string
		output   crisis.DetailOutput
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
			input:  "project-1",
			mock:   mockData{detail: mockDetail{isCalled: true, input: "project-1", output: crisis.DetailOutput{CrisisConfig: model.CrisisConfig{ProjectID: "project-1"}}}},
			output: http.StatusOK,
		},
		"uc_error": {
			input:  "project-1",
			mock:   mockData{detail: mockDetail{isCalled: true, input: "project-1", err: crisis.ErrNotFound}},
			output: http.StatusNotFound,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.detail.isCalled {
				uc.EXPECT().Detail(context.Background(), tc.mock.detail.input).Return(tc.mock.detail.output, tc.mock.detail.err)
			}
			c, w := newTestContext(http.MethodGet, "/projects/"+tc.input+"/crisis-config", "", gin.Params{{Key: "project_id", Value: tc.input}})

			h.Detail(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}

func TestDelete(t *testing.T) {
	type mockDelete struct {
		isCalled bool
		input    string
		err      error
	}
	type mockData struct {
		delete mockDelete
	}

	tcs := map[string]struct {
		input  string
		mock   mockData
		output int
		err    error
	}{
		"success": {
			input:  "project-1",
			mock:   mockData{delete: mockDelete{isCalled: true, input: "project-1"}},
			output: http.StatusOK,
		},
		"uc_error": {
			input:  "project-1",
			mock:   mockData{delete: mockDelete{isCalled: true, input: "project-1", err: crisis.ErrDeleteFailed}},
			output: http.StatusInternalServerError,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			h, uc := newTestHandler(t)
			if tc.mock.delete.isCalled {
				uc.EXPECT().Delete(context.Background(), tc.mock.delete.input).Return(tc.mock.delete.err)
			}
			c, w := newTestContext(http.MethodDelete, "/projects/"+tc.input+"/crisis-config", "", gin.Params{{Key: "project_id", Value: tc.input}})

			h.Delete(c)

			require.Equal(t, tc.output, w.Code)
		})
	}
}
