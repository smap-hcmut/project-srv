package http

import (
	"testing"

	"smap-project/internal/model"
)

func TestPatchReq_Validate_ValidStatus(t *testing.T) {
	validStatuses := []string{
		model.ProjectStatusDraft,
		model.ProjectStatusProcess,
		model.ProjectStatusCompleted,
	}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			req := PatchReq{
				ID:     "550e8400-e29b-41d4-a716-446655440000",
				Status: &status,
			}

			err := req.validate()
			if err != nil {
				t.Errorf("Expected no error for valid status %q, got: %v", status, err)
			}
		})
	}
}

func TestPatchReq_Validate_InvalidStatus(t *testing.T) {
	invalidStatuses := []string{
		"active",
		"archived",
		"cancelled",
		"invalid",
		"",
	}

	for _, status := range invalidStatuses {
		t.Run(status, func(t *testing.T) {
			req := PatchReq{
				ID:     "550e8400-e29b-41d4-a716-446655440000",
				Status: &status,
			}

			err := req.validate()
			if err == nil {
				t.Errorf("Expected error for invalid status %q, got nil", status)
			}
			if err != nil && err.Error() != "invalid status" {
				t.Errorf("Expected 'invalid status' error, got: %v", err)
			}
		})
	}
}

func TestGetReq_Validate_ValidStatuses(t *testing.T) {
	req := GetReq{
		Statuses: []string{
			model.ProjectStatusDraft,
			model.ProjectStatusProcess,
			model.ProjectStatusCompleted,
		},
	}

	err := req.validate()
	if err != nil {
		t.Errorf("Expected no error for valid statuses, got: %v", err)
	}
}

func TestGetReq_Validate_InvalidStatuses(t *testing.T) {
	testCases := []struct {
		name     string
		statuses []string
	}{
		{
			name:     "single invalid status",
			statuses: []string{"active"},
		},
		{
			name:     "mixed valid and invalid",
			statuses: []string{model.ProjectStatusDraft, "archived"},
		},
		{
			name:     "multiple invalid",
			statuses: []string{"active", "cancelled"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := GetReq{
				Statuses: tc.statuses,
			}

			err := req.validate()
			if err == nil {
				t.Errorf("Expected error for invalid statuses %v, got nil", tc.statuses)
			}
			if err != nil && err.Error() != "invalid status" {
				t.Errorf("Expected 'invalid status' error, got: %v", err)
			}
		})
	}
}

func TestGetReq_Validate_EmptyStatuses(t *testing.T) {
	req := GetReq{
		Statuses: []string{},
	}

	err := req.validate()
	if err != nil {
		t.Errorf("Expected no error for empty statuses, got: %v", err)
	}
}
