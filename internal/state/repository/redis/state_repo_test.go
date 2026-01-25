package redis

import (
	"testing"

	"smap-project/internal/model"
)

// TestBuildKey tests the key building function.
func TestBuildKey(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		want      string
	}{
		{
			name:      "simple project ID",
			projectID: "123",
			want:      "smap:proj:123",
		},
		{
			name:      "UUID project ID",
			projectID: "550e8400-e29b-41d4-a716-446655440000",
			want:      "smap:proj:550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildKey(tt.projectID)
			if got != tt.want {
				t.Errorf("buildKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatusConstants tests that status constants are defined correctly.
func TestStatusConstants(t *testing.T) {
	tests := []struct {
		status model.ProjectStatus
		want   string
	}{
		{model.ProjectStatusInitializing, "INITIALIZING"},
		{model.ProjectStatusCrawling, "CRAWLING"},
		{model.ProjectStatusProcessing, "PROCESSING"},
		{model.ProjectStatusDone, "DONE"},
		{model.ProjectStatusFailed, "FAILED"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("Status = %v, want %v", tt.status, tt.want)
			}
		})
	}
}

// TestConstants tests repository constants.
func TestConstants(t *testing.T) {
	// Verify key prefix
	if keyPrefix != "smap:proj:" {
		t.Errorf("keyPrefix = %v, want smap:proj:", keyPrefix)
	}

	// Verify field names
	if fieldStatus != "status" {
		t.Errorf("fieldStatus = %v, want status", fieldStatus)
	}
	if fieldTotal != "total" {
		t.Errorf("fieldTotal = %v, want total", fieldTotal)
	}
	if fieldDone != "done" {
		t.Errorf("fieldDone = %v, want done", fieldDone)
	}
	if fieldErrors != "errors" {
		t.Errorf("fieldErrors = %v, want errors", fieldErrors)
	}

	// Verify TTL is 7 days (604800 seconds)
	expectedTTL := 7 * 24 * 60 * 60 // 604800 seconds
	if stateTTLSeconds != expectedTTL {
		t.Errorf("stateTTLSeconds = %v, want %v", stateTTLSeconds, expectedTTL)
	}
}
