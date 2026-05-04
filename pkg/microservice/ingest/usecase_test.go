package ingest

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"project-srv/pkg/microservice"

	"github.com/smap-hcmut/shared-libs/go/contracts"
	"github.com/stretchr/testify/require"
)

type fakeHTTPClient struct {
	getBody    []byte
	getStatus  int
	getErr     error
	postBody   []byte
	postStatus int
	postErr    error
	lastMethod string
	lastURL    string
	headers    map[string]string
}

func (f *fakeHTTPClient) Get(_ context.Context, url string, headers map[string]string) ([]byte, int, error) {
	f.lastMethod = http.MethodGet
	f.lastURL = url
	f.headers = headers
	return f.getBody, f.getStatus, f.getErr
}

func (f *fakeHTTPClient) Post(_ context.Context, url string, _ interface{}, headers map[string]string) ([]byte, int, error) {
	f.lastMethod = http.MethodPost
	f.lastURL = url
	f.headers = headers
	return f.postBody, f.postStatus, f.postErr
}

func newTestUseCase(client *fakeHTTPClient) *implUseCase {
	return &implUseCase{
		baseURL:     "http://ingest.local/api/v1",
		internalKey: "internal-key",
		client:      client,
	}
}

func TestNew(t *testing.T) {
	tcs := map[string]struct {
		input struct {
			baseURL   string
			timeoutMS int
		}
		mock   struct{}
		output bool
		err    error
	}{
		"default_timeout": {
			input: struct {
				baseURL   string
				timeoutMS int
			}{baseURL: "http://ingest.local/", timeoutMS: 0},
			output: true,
		},
		"custom_timeout": {
			input: struct {
				baseURL   string
				timeoutMS int
			}{baseURL: "http://ingest.local/", timeoutMS: 1000},
			output: true,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := New(nil, tc.input.baseURL, tc.input.timeoutMS, "key")

			require.Equal(t, tc.output, output != nil)
		})
	}
}

func TestBuildEndpoint(t *testing.T) {
	tcs := map[string]struct {
		input struct {
			baseURL string
			path    string
		}
		mock   struct{}
		output string
		err    error
	}{
		"api_v1": {
			input:  struct{ baseURL, path string }{baseURL: "http://ingest.local/api/v1", path: "/projects/1/activate"},
			output: "http://ingest.local/api/v1/internal/projects/1/activate",
		},
		"plain_base": {
			input:  struct{ baseURL, path string }{baseURL: "http://ingest.local", path: "/projects/1/activate"},
			output: "http://ingest.local/api/v1/internal/projects/1/activate",
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc := &implUseCase{baseURL: tc.input.baseURL}

			output := uc.buildEndpoint(tc.input.path)

			require.Equal(t, tc.output, output)
		})
	}
}

func TestGetActivationReadiness(t *testing.T) {
	ctx := context.Background()
	successBody := []byte(`{"data":{"project_id":"project-1","datasource_count":1,"has_datasource":true,"passive_unconfirmed_count":2,"missing_target_dryrun_count":3,"failed_target_dryrun_count":4,"can_proceed":false,"errors":[{"code":"DATASOURCE_REQUIRED","message":"missing","datasource_id":"ds-1","target_id":"target-1"}]}}`)

	tcs := map[string]struct {
		input microservice.ActivationReadinessInput
		mock  struct {
			client *fakeHTTPClient
		}
		output microservice.ActivationReadiness
		err    error
	}{
		"success": {
			input: microservice.ActivationReadinessInput{ProjectID: " project-1 ", Command: microservice.ActivationReadinessCommandResume},
			mock:  struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{getBody: successBody, getStatus: http.StatusOK}},
			output: microservice.ActivationReadiness{
				ProjectID:                "project-1",
				DataSourceCount:          1,
				HasDatasource:            true,
				PassiveUnconfirmedCount:  2,
				MissingTargetDryrunCount: 3,
				FailedTargetDryrunCount:  4,
				CanProceed:               false,
				Errors: []microservice.ActivationReadinessError{{
					Code:         contracts.ReadinessCodeDatasourceRequired,
					Message:      "missing",
					DataSourceID: "ds-1",
					TargetID:     "target-1",
				}},
			},
		},
		"request_error": {
			input: microservice.ActivationReadinessInput{ProjectID: "project-1"},
			mock:  struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{getErr: errors.New("request error")}},
			err:   microservice.ErrRequestFailed,
		},
		"status_error": {
			input: microservice.ActivationReadinessInput{ProjectID: "project-1"},
			mock:  struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{getBody: []byte("bad"), getStatus: http.StatusBadRequest}},
			err:   microservice.ErrBadRequest,
		},
		"bad_envelope": {
			input: microservice.ActivationReadinessInput{ProjectID: "project-1"},
			mock:  struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{getBody: []byte(`{`), getStatus: http.StatusOK}},
			err:   microservice.ErrRequestFailed,
		},
		"bad_data": {
			input: microservice.ActivationReadinessInput{ProjectID: "project-1"},
			mock:  struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{getBody: []byte(`{"data":{`), getStatus: http.StatusOK}},
			err:   microservice.ErrRequestFailed,
		},
		"fallback_project_id": {
			input:  microservice.ActivationReadinessInput{ProjectID: "project-1"},
			mock:   struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{getBody: []byte(`{"data":{"can_proceed":true}}`), getStatus: http.StatusOK}},
			output: microservice.ActivationReadiness{ProjectID: "project-1", CanProceed: true, Errors: []microservice.ActivationReadinessError{}},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc := newTestUseCase(tc.mock.client)

			output, err := uc.GetActivationReadiness(ctx, tc.input)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.output, output)
			require.Equal(t, "internal-key", tc.mock.client.headers[internalAuthHeader])
		})
	}
}

func TestActivate(t *testing.T) {
	testLifecycleCommand(t, "activate", func(uc *implUseCase, ctx context.Context, projectID string) error {
		return uc.Activate(ctx, projectID)
	})
}

func TestPause(t *testing.T) {
	testLifecycleCommand(t, "pause", func(uc *implUseCase, ctx context.Context, projectID string) error {
		return uc.Pause(ctx, projectID)
	})
}

func TestResume(t *testing.T) {
	testLifecycleCommand(t, "resume", func(uc *implUseCase, ctx context.Context, projectID string) error {
		return uc.Resume(ctx, projectID)
	})
}

func testLifecycleCommand(t *testing.T, name string, call func(*implUseCase, context.Context, string) error) {
	t.Helper()
	ctx := context.Background()

	tcs := map[string]struct {
		input string
		mock  struct {
			client *fakeHTTPClient
		}
		output struct{}
		err    error
	}{
		"success": {
			input: " project-1 ",
			mock:  struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{postStatus: http.StatusOK}},
		},
		"request_error": {
			input: "project-1",
			mock:  struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{postErr: errors.New("request error")}},
			err:   microservice.ErrRequestFailed,
		},
		"status_error": {
			input: "project-1",
			mock:  struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{postBody: []byte("forbidden"), postStatus: http.StatusForbidden}},
			err:   microservice.ErrForbidden,
		},
	}

	for caseName, tc := range tcs {
		t.Run(name+"_"+caseName, func(t *testing.T) {
			uc := newTestUseCase(tc.mock.client)

			err := call(uc, ctx, tc.input)

			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, http.MethodPost, tc.mock.client.lastMethod)
		})
	}
}

func TestDoRequest(t *testing.T) {
	ctx := context.Background()

	tcs := map[string]struct {
		input string
		mock  struct {
			client *fakeHTTPClient
		}
		output int
		err    error
	}{
		"unsupported_method": {
			input:  http.MethodPut,
			mock:   struct{ client *fakeHTTPClient }{client: &fakeHTTPClient{}},
			output: 0,
			err:    microservice.ErrRequestFailed,
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			uc := newTestUseCase(tc.mock.client)

			_, output, err := uc.doRequest(ctx, tc.input, "http://ingest.local")

			require.ErrorIs(t, err, tc.err)
			require.Equal(t, tc.output, output)
		})
	}
}

func TestMapStatusError(t *testing.T) {
	tcs := map[string]struct {
		input  int
		mock   struct{}
		output error
		err    error
	}{
		"bad_request":   {input: http.StatusBadRequest, output: microservice.ErrBadRequest},
		"unauthorized":  {input: http.StatusUnauthorized, output: microservice.ErrUnauthorized},
		"forbidden":     {input: http.StatusForbidden, output: microservice.ErrForbidden},
		"request_error": {input: http.StatusInternalServerError, output: microservice.ErrRequestFailed},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			err := mapStatusError(tc.input, []byte("body"))

			require.ErrorIs(t, err, tc.output)
		})
	}
}
