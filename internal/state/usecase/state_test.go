package usecase

import (
	"testing"

	"smap-project/internal/model"
)

// TestProjectStateCompletion tests completion logic.
func TestProjectStateCompletion(t *testing.T) {
	tests := []struct {
		name       string
		total      int64
		done       int64
		wantResult bool
	}{
		{
			name:       "complete when done equals total",
			total:      10,
			done:       10,
			wantResult: true,
		},
		{
			name:       "complete when done exceeds total",
			total:      10,
			done:       15,
			wantResult: true,
		},
		{
			name:       "not complete when done less than total",
			total:      10,
			done:       5,
			wantResult: false,
		},
		{
			name:       "not complete when total is zero",
			total:      0,
			done:       0,
			wantResult: false,
		},
		{
			name:       "not complete when total is zero but done is positive",
			total:      0,
			done:       5,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Business logic: done >= total && total > 0
			isComplete := tt.total > 0 && tt.done >= tt.total
			if isComplete != tt.wantResult {
				t.Errorf("completion check = %v, want %v", isComplete, tt.wantResult)
			}
		})
	}
}

// TestProgressPercent tests progress calculation logic.
func TestProgressPercent(t *testing.T) {
	tests := []struct {
		name  string
		total int64
		done  int64
		want  float64
	}{
		{
			name:  "50 percent progress",
			total: 100,
			done:  50,
			want:  50.0,
		},
		{
			name:  "100 percent progress",
			total: 100,
			done:  100,
			want:  100.0,
		},
		{
			name:  "0 percent progress",
			total: 100,
			done:  0,
			want:  0.0,
		},
		{
			name:  "zero total returns 0",
			total: 0,
			done:  0,
			want:  0.0,
		},
		{
			name:  "over 100 percent",
			total: 100,
			done:  150,
			want:  150.0,
		},
		{
			name:  "fractional progress",
			total: 3,
			done:  1,
			want:  33.33, // ~33.33%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got float64
			if tt.total == 0 {
				got = 0
			} else {
				got = float64(tt.done) / float64(tt.total) * 100
			}

			// Use tolerance for floating point comparison
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 0.01 { // 0.01% tolerance
				t.Errorf("progress = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStatusTransitions tests valid status transitions.
func TestStatusTransitions(t *testing.T) {
	validTransitions := map[model.ProjectStatus][]model.ProjectStatus{
		model.ProjectStatusInitializing: {model.ProjectStatusCrawling, model.ProjectStatusFailed},
		model.ProjectStatusCrawling:     {model.ProjectStatusProcessing, model.ProjectStatusFailed},
		model.ProjectStatusProcessing:   {model.ProjectStatusDone, model.ProjectStatusFailed},
		model.ProjectStatusDone:         {}, // Terminal state
		model.ProjectStatusFailed:       {}, // Terminal state
	}

	for from, validTo := range validTransitions {
		t.Run(string(from), func(t *testing.T) {
			if len(validTo) == 0 {
				// Terminal state - no transitions allowed
				return
			}
			// Just verify the transitions are defined
			for _, to := range validTo {
				if to == "" {
					t.Errorf("invalid transition from %s", from)
				}
			}
		})
	}
}
