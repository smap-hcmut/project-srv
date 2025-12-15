package model

import (
	"encoding/json"
	"testing"
)

func TestCrawlProgressPercent(t *testing.T) {
	tests := []struct {
		name     string
		state    ProjectState
		expected float64
	}{
		{
			name: "normal progress",
			state: ProjectState{
				CrawlTotal:  100,
				CrawlDone:   80,
				CrawlErrors: 5,
			},
			expected: 85.0,
		},
		{
			name: "zero total",
			state: ProjectState{
				CrawlTotal:  0,
				CrawlDone:   0,
				CrawlErrors: 0,
			},
			expected: 0.0,
		},
		{
			name: "all errors",
			state: ProjectState{
				CrawlTotal:  100,
				CrawlDone:   0,
				CrawlErrors: 100,
			},
			expected: 100.0,
		},
		{
			name: "complete with no errors",
			state: ProjectState{
				CrawlTotal:  50,
				CrawlDone:   50,
				CrawlErrors: 0,
			},
			expected: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.CrawlProgressPercent()
			if result != tt.expected {
				t.Errorf("CrawlProgressPercent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAnalyzeProgressPercent(t *testing.T) {
	tests := []struct {
		name     string
		state    ProjectState
		expected float64
	}{
		{
			name: "normal progress",
			state: ProjectState{
				AnalyzeTotal:  100,
				AnalyzeDone:   45,
				AnalyzeErrors: 5,
			},
			expected: 50.0,
		},
		{
			name: "zero total",
			state: ProjectState{
				AnalyzeTotal:  0,
				AnalyzeDone:   0,
				AnalyzeErrors: 0,
			},
			expected: 0.0,
		},
		{
			name: "complete",
			state: ProjectState{
				AnalyzeTotal:  80,
				AnalyzeDone:   75,
				AnalyzeErrors: 5,
			},
			expected: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.AnalyzeProgressPercent()
			if result != tt.expected {
				t.Errorf("AnalyzeProgressPercent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOverallProgressPercent(t *testing.T) {
	tests := []struct {
		name     string
		state    ProjectState
		expected float64
	}{
		{
			name: "both phases at 50%",
			state: ProjectState{
				CrawlTotal:   100,
				CrawlDone:    50,
				AnalyzeTotal: 100,
				AnalyzeDone:  50,
			},
			expected: 50.0,
		},
		{
			name: "crawl complete, analyze at 50%",
			state: ProjectState{
				CrawlTotal:   100,
				CrawlDone:    100,
				AnalyzeTotal: 100,
				AnalyzeDone:  50,
			},
			expected: 75.0,
		},
		{
			name: "crawl only (analyze not started)",
			state: ProjectState{
				CrawlTotal:   100,
				CrawlDone:    60,
				AnalyzeTotal: 0,
				AnalyzeDone:  0,
			},
			expected: 30.0,
		},
		{
			name: "both complete",
			state: ProjectState{
				CrawlTotal:   100,
				CrawlDone:    100,
				AnalyzeTotal: 100,
				AnalyzeDone:  100,
			},
			expected: 100.0,
		},
		{
			name: "both zero",
			state: ProjectState{
				CrawlTotal:   0,
				AnalyzeTotal: 0,
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.OverallProgressPercent()
			if result != tt.expected {
				t.Errorf("OverallProgressPercent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestProjectStateJSONSerialization(t *testing.T) {
	state := ProjectState{
		Status:        ProjectStatusProcessing,
		CrawlTotal:    100,
		CrawlDone:     80,
		CrawlErrors:   2,
		AnalyzeTotal:  80,
		AnalyzeDone:   40,
		AnalyzeErrors: 1,
	}

	// Serialize
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Deserialize
	var decoded ProjectState
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if decoded.Status != state.Status {
		t.Errorf("Status mismatch: got %v, want %v", decoded.Status, state.Status)
	}
	if decoded.CrawlTotal != state.CrawlTotal {
		t.Errorf("CrawlTotal mismatch: got %v, want %v", decoded.CrawlTotal, state.CrawlTotal)
	}
	if decoded.CrawlDone != state.CrawlDone {
		t.Errorf("CrawlDone mismatch: got %v, want %v", decoded.CrawlDone, state.CrawlDone)
	}
	if decoded.AnalyzeTotal != state.AnalyzeTotal {
		t.Errorf("AnalyzeTotal mismatch: got %v, want %v", decoded.AnalyzeTotal, state.AnalyzeTotal)
	}
}
