package http

import (
	"errors"
	"testing"
	"time"

	"project-srv/internal/campaign"
	"project-srv/internal/model"

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
		"not_found":          {input: campaign.ErrNotFound, output: errNotFound},
		"name_required":      {input: campaign.ErrNameRequired, output: errNameRequired},
		"invalid_status":     {input: campaign.ErrInvalidStatus, output: errInvalidStatus},
		"invalid_sort":       {input: campaign.ErrInvalidSort, output: errInvalidSort},
		"invalid_date_range": {input: campaign.ErrInvalidDateRange, output: errInvalidDateRange},
		"create_failed":      {input: campaign.ErrCreateFailed, output: errCreateFailed},
		"update_failed":      {input: campaign.ErrUpdateFailed, output: errUpdateFailed},
		"delete_failed":      {input: campaign.ErrDeleteFailed, output: errDeleteFailed},
		"list_failed":        {input: campaign.ErrListFailed, output: errListFailed},
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
		"success":             {input: createReq{Name: "Campaign A", StartDate: "2026-01-01T00:00:00Z", EndDate: "2026-02-01T00:00:00Z"}},
		"name_required":       {input: createReq{}, err: errNameRequired},
		"invalid_start":       {input: createReq{Name: "Campaign A", StartDate: "bad"}, err: errInvalidDateFormat},
		"invalid_end":         {input: createReq{Name: "Campaign A", EndDate: "bad"}, err: errInvalidDateFormat},
		"invalid_both_format": {input: createReq{Name: "Campaign A", StartDate: "bad", EndDate: "2026-02-01T00:00:00Z"}, err: errInvalidDateFormat},
		"invalid_range":       {input: createReq{Name: "Campaign A", StartDate: "2026-02-01T00:00:00Z", EndDate: "2026-01-01T00:00:00Z"}, err: errInvalidDateRange},
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
		"success":        {input: updateReq{Status: "ACTIVE", StartDate: "2026-01-01T00:00:00Z", EndDate: "2026-02-01T00:00:00Z"}},
		"success_start":  {input: updateReq{StartDate: "2026-01-01T00:00:00Z"}},
		"success_end":    {input: updateReq{EndDate: "2026-02-01T00:00:00Z"}},
		"invalid_status": {input: updateReq{Status: "BAD"}, err: errInvalidStatus},
		"invalid_start":  {input: updateReq{StartDate: "bad"}, err: errInvalidDateFormat},
		"invalid_end":    {input: updateReq{EndDate: "bad"}, err: errInvalidDateFormat},
		"invalid_end_with_valid_start": {input: updateReq{
			StartDate: "2026-01-01T00:00:00Z",
			EndDate:   "bad",
		}, err: errInvalidDateFormat},
		"invalid_range": {input: updateReq{StartDate: "2026-02-01T00:00:00Z", EndDate: "2026-01-01T00:00:00Z"}, err: errInvalidDateRange},
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

func TestToCampaignResp(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		input  model.Campaign
		mock   struct{}
		output campaignResp
		err    error
	}{
		"with_optional_fields": {
			input: model.Campaign{ID: "campaign-1", Name: "Campaign A", Description: "Desc", Status: model.CampaignStatusActive, IsFavorite: true, StartDate: &now, EndDate: &now, CreatedBy: "user-1", CreatedAt: now, UpdatedAt: now},
			output: campaignResp{
				ID:          "campaign-1",
				Name:        "Campaign A",
				Description: "Desc",
				Status:      "ACTIVE",
				IsFavorite:  true,
				StartDate:   ptr("2026-01-01T00:00:00Z"),
				EndDate:     ptr("2026-01-01T00:00:00Z"),
				CreatedBy:   "user-1",
				CreatedAt:   "2026-01-01T00:00:00Z",
				UpdatedAt:   "2026-01-01T00:00:00Z",
			},
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			output := toCampaignResp(tc.input)

			require.Equal(t, tc.output, output)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
