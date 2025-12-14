package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"smap-project/config"
	"smap-project/internal/sampling"
	"smap-project/pkg/log"
)

func TestUsecase_Sample(t *testing.T) {
	cfg := config.DryRunSamplingConfig{
		Percentage:          10,
		MinKeywords:         3,
		MaxKeywords:         5,
		KeywordTimeEstimate: "16s",
		DefaultStrategy:     "percentage",
		EmergencyThreshold:  "200s", // High enough to not trigger
		EmergencyKeywords:   3,
	}
	logger := log.NewNopLogger()
	uc := NewStrategy(cfg, logger)

	tests := []struct {
		name           string
		keywords       []string
		expectedCount  int
		expectedRatio  float64
		expectedMethod string
	}{
		{
			name:           "empty keywords",
			keywords:       []string{},
			expectedCount:  0,
			expectedRatio:  0,
			expectedMethod: "empty",
		},
		{
			name:           "50 keywords with 10% sampling",
			keywords:       generateKeywords(50),
			expectedCount:  5, // 10% of 50 = 5, capped at max
			expectedRatio:  0.1,
			expectedMethod: "random",
		},
		{
			name:           "10 keywords with 10% sampling",
			keywords:       generateKeywords(10),
			expectedCount:  3, // 10% of 10 = 1, but minimum is 3
			expectedRatio:  0.3,
			expectedMethod: "random",
		},
		{
			name:           "2 keywords with 10% sampling",
			keywords:       generateKeywords(2),
			expectedCount:  2, // Cannot exceed available keywords
			expectedRatio:  1.0,
			expectedMethod: "all",
		},
		{
			name:           "single keyword",
			keywords:       []string{"keyword1"},
			expectedCount:  1,
			expectedRatio:  1.0,
			expectedMethod: "all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := sampling.SampleInput{
				Keywords: tt.keywords,
				UserID:   "test-user",
			}

			result, err := uc.Sample(context.Background(), input)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if len(result.SelectedKeywords) != tt.expectedCount {
				t.Errorf("Expected %d selected keywords, got %d", tt.expectedCount, len(result.SelectedKeywords))
			}

			if result.TotalKeywords != len(tt.keywords) {
				t.Errorf("Expected total keywords %d, got %d", len(tt.keywords), result.TotalKeywords)
			}

			if abs(result.SamplingRatio-tt.expectedRatio) > 0.01 {
				t.Errorf("Expected sampling ratio %.2f, got %.2f", tt.expectedRatio, result.SamplingRatio)
			}

			if result.SelectionMethod != tt.expectedMethod {
				t.Errorf("Expected selection method %s, got %s", tt.expectedMethod, result.SelectionMethod)
			}

			// Verify all selected keywords are from the original set
			for _, selected := range result.SelectedKeywords {
				found := false
				for _, original := range tt.keywords {
					if selected == original {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Selected keyword %s not found in original keywords", selected)
				}
			}
		})
	}
}

func TestUsecase_Sample_EmergencyFallback(t *testing.T) {
	cfg := config.DryRunSamplingConfig{
		Percentage:          10,
		MinKeywords:         3,
		MaxKeywords:         10, // High max to trigger emergency fallback
		KeywordTimeEstimate: "16s",
		DefaultStrategy:     "percentage",
		EmergencyThreshold:  "70s", // 70s / 16s = ~4.4 keywords max
		EmergencyKeywords:   3,
	}
	logger := log.NewNopLogger()
	uc := NewStrategy(cfg, logger)

	// Create enough keywords to trigger emergency fallback
	keywords := generateKeywords(100) // 10% = 10 keywords, but 10*16s = 160s > 70s

	input := sampling.SampleInput{
		Keywords: keywords,
		UserID:   "test-user",
	}

	result, err := uc.Sample(context.Background(), input)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should use emergency keywords count instead of calculated count
	if len(result.SelectedKeywords) != cfg.EmergencyKeywords {
		t.Errorf("Expected emergency fallback to %d keywords, got %d", cfg.EmergencyKeywords, len(result.SelectedKeywords))
	}

	// Estimated time should be within emergency threshold
	emergencyThreshold, _ := time.ParseDuration(cfg.EmergencyThreshold)
	if result.EstimatedTime > emergencyThreshold {
		t.Errorf("Emergency fallback should result in time <= %v, got %v", emergencyThreshold, result.EstimatedTime)
	}
}

func TestUsecase_FallbackToDefaults(t *testing.T) {
	// Test with invalid config - should fallback to defaults
	invalidCfg := config.DryRunSamplingConfig{
		Percentage:          0, // Invalid - should trigger fallback
		MinKeywords:         3,
		MaxKeywords:         5,
		KeywordTimeEstimate: "16s",
		DefaultStrategy:     "percentage",
		EmergencyThreshold:  "70s",
		EmergencyKeywords:   3,
	}
	logger := log.NewNopLogger()
	uc := NewStrategy(invalidCfg, logger)

	// Should still work with default config
	input := sampling.SampleInput{
		Keywords: generateKeywords(50),
		UserID:   "test-user",
	}

	result, err := uc.Sample(context.Background(), input)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should use default config (10%, min 3, max 5)
	// 50 keywords * 10% = 5, capped at max 5
	// But default emergency threshold is 70s, and 5 * 16s = 80s > 70s
	// So emergency fallback kicks in and returns 3 keywords
	if len(result.SelectedKeywords) != 3 {
		t.Errorf("Expected 3 keywords with default config (emergency fallback), got %d", len(result.SelectedKeywords))
	}
}

func TestUsecase_Sample_ValidationErrors(t *testing.T) {
	cfg := config.DryRunSamplingConfig{
		Percentage:          10,
		MinKeywords:         3,
		MaxKeywords:         5,
		KeywordTimeEstimate: "16s",
		DefaultStrategy:     "percentage",
		EmergencyThreshold:  "70s",
		EmergencyKeywords:   3,
	}
	logger := log.NewNopLogger()
	uc := NewStrategy(cfg, logger)

	tests := []struct {
		name        string
		input       sampling.SampleInput
		expectError error
	}{
		{
			name: "missing user ID",
			input: sampling.SampleInput{
				Keywords: []string{"kw1", "kw2"},
				UserID:   "",
			},
			expectError: sampling.ErrMissingUserID,
		},
		{
			name: "nil keywords",
			input: sampling.SampleInput{
				Keywords: nil,
				UserID:   "user123",
			},
			expectError: sampling.ErrNilKeywords,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Sample(context.Background(), tt.input)
			if err != tt.expectError {
				t.Errorf("Expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

// Helper functions
func generateKeywords(count int) []string {
	keywords := make([]string, count)
	for i := 0; i < count; i++ {
		keywords[i] = fmt.Sprintf("keyword%d", i+1)
	}
	return keywords
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Benchmark tests
func BenchmarkUsecase_Sample(b *testing.B) {
	cfg := config.DryRunSamplingConfig{
		Percentage:          10,
		MinKeywords:         3,
		MaxKeywords:         5,
		KeywordTimeEstimate: "16s",
		DefaultStrategy:     "percentage",
		EmergencyThreshold:  "70s",
		EmergencyKeywords:   3,
	}
	logger := log.NewNopLogger()
	uc := NewStrategy(cfg, logger)

	keywords := generateKeywords(100)
	input := sampling.SampleInput{
		Keywords: keywords,
		UserID:   "bench-user",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.Sample(context.Background(), input)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}

// Test Fixed Strategy
func TestUsecase_Sample_FixedStrategy(t *testing.T) {
	cfg := config.DryRunSamplingConfig{
		Percentage:          10,
		MinKeywords:         3,
		MaxKeywords:         5,
		KeywordTimeEstimate: "16s",
		DefaultStrategy:     "fixed",
		EmergencyThreshold:  "200s", // High enough to not trigger
		EmergencyKeywords:   3,
	}
	logger := log.NewNopLogger()
	uc := NewStrategy(cfg, logger)

	tests := []struct {
		name          string
		keywords      []string
		expectedCount int
	}{
		{
			name:          "fixed strategy with 50 keywords",
			keywords:      generateKeywords(50),
			expectedCount: 5, // MaxKeywords
		},
		{
			name:          "fixed strategy with 3 keywords",
			keywords:      generateKeywords(3),
			expectedCount: 3, // Cannot exceed available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := sampling.SampleInput{
				Keywords: tt.keywords,
				UserID:   "test-user",
			}

			result, err := uc.Sample(context.Background(), input)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if len(result.SelectedKeywords) != tt.expectedCount {
				t.Errorf("Expected %d selected keywords, got %d", tt.expectedCount, len(result.SelectedKeywords))
			}
		})
	}
}

// Test Tiered Strategy
func TestUsecase_Sample_TieredStrategy(t *testing.T) {
	cfg := config.DryRunSamplingConfig{
		Percentage:          10,
		MinKeywords:         3,
		MaxKeywords:         8,
		KeywordTimeEstimate: "16s",
		DefaultStrategy:     "tiered",
		EmergencyThreshold:  "200s", // High enough to not trigger
		EmergencyKeywords:   3,
	}
	logger := log.NewNopLogger()
	uc := NewStrategy(cfg, logger)

	tests := []struct {
		name          string
		keywords      []string
		expectedCount int
	}{
		{
			name:          "tiered strategy with 5 keywords (small tier)",
			keywords:      generateKeywords(5),
			expectedCount: 3, // MinKeywords for small tier
		},
		{
			name:          "tiered strategy with 30 keywords (medium tier)",
			keywords:      generateKeywords(30),
			expectedCount: 5, // (MinKeywords + MaxKeywords) / 2 = (3+8)/2 = 5
		},
		{
			name:          "tiered strategy with 100 keywords (large tier)",
			keywords:      generateKeywords(100),
			expectedCount: 8, // MaxKeywords for large tier
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := sampling.SampleInput{
				Keywords: tt.keywords,
				UserID:   "test-user",
			}

			result, err := uc.Sample(context.Background(), input)
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if len(result.SelectedKeywords) != tt.expectedCount {
				t.Errorf("Expected %d selected keywords, got %d", tt.expectedCount, len(result.SelectedKeywords))
			}
		})
	}
}

// Test config parsing errors
func TestUsecase_ConfigParsingErrors(t *testing.T) {
	logger := log.NewNopLogger()

	tests := []struct {
		name string
		cfg  config.DryRunSamplingConfig
	}{
		{
			name: "invalid keyword time estimate",
			cfg: config.DryRunSamplingConfig{
				Percentage:          10,
				MinKeywords:         3,
				MaxKeywords:         5,
				KeywordTimeEstimate: "invalid",
				DefaultStrategy:     "percentage",
				EmergencyThreshold:  "70s",
				EmergencyKeywords:   3,
			},
		},
		{
			name: "invalid emergency threshold",
			cfg: config.DryRunSamplingConfig{
				Percentage:          10,
				MinKeywords:         3,
				MaxKeywords:         5,
				KeywordTimeEstimate: "16s",
				DefaultStrategy:     "percentage",
				EmergencyThreshold:  "invalid",
				EmergencyKeywords:   3,
			},
		},
		{
			name: "invalid strategy",
			cfg: config.DryRunSamplingConfig{
				Percentage:          10,
				MinKeywords:         3,
				MaxKeywords:         5,
				KeywordTimeEstimate: "16s",
				DefaultStrategy:     "invalid_strategy",
				EmergencyThreshold:  "70s",
				EmergencyKeywords:   3,
			},
		},
		{
			name: "invalid min keywords",
			cfg: config.DryRunSamplingConfig{
				Percentage:          10,
				MinKeywords:         0, // Invalid
				MaxKeywords:         5,
				KeywordTimeEstimate: "16s",
				DefaultStrategy:     "percentage",
				EmergencyThreshold:  "70s",
				EmergencyKeywords:   3,
			},
		},
		{
			name: "invalid max keywords",
			cfg: config.DryRunSamplingConfig{
				Percentage:          10,
				MinKeywords:         5,
				MaxKeywords:         3, // Less than min
				KeywordTimeEstimate: "16s",
				DefaultStrategy:     "percentage",
				EmergencyThreshold:  "70s",
				EmergencyKeywords:   3,
			},
		},
		{
			name: "invalid emergency keywords",
			cfg: config.DryRunSamplingConfig{
				Percentage:          10,
				MinKeywords:         3,
				MaxKeywords:         5,
				KeywordTimeEstimate: "16s",
				DefaultStrategy:     "percentage",
				EmergencyThreshold:  "70s",
				EmergencyKeywords:   0, // Invalid
			},
		},
		{
			name: "percentage over 100",
			cfg: config.DryRunSamplingConfig{
				Percentage:          150, // Invalid
				MinKeywords:         3,
				MaxKeywords:         5,
				KeywordTimeEstimate: "16s",
				DefaultStrategy:     "percentage",
				EmergencyThreshold:  "70s",
				EmergencyKeywords:   3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should fallback to defaults and still work
			uc := NewStrategy(tt.cfg, logger)

			input := sampling.SampleInput{
				Keywords: generateKeywords(10),
				UserID:   "test-user",
			}

			result, err := uc.Sample(context.Background(), input)
			if err != nil {
				t.Errorf("Expected no error (fallback to defaults), got %v", err)
			}

			// Should have selected some keywords using default config
			if len(result.SelectedKeywords) == 0 {
				t.Error("Expected some keywords to be selected with default config")
			}
		})
	}
}

// Test selector edge cases
func TestSelector_EdgeCases(t *testing.T) {
	selector := newRandomSelector()

	tests := []struct {
		name           string
		keywords       []string
		count          int
		expectedLen    int
		expectedMethod string
	}{
		{
			name:           "count is zero",
			keywords:       []string{"a", "b", "c"},
			count:          0,
			expectedLen:    0,
			expectedMethod: "empty",
		},
		{
			name:           "count is negative",
			keywords:       []string{"a", "b", "c"},
			count:          -1,
			expectedLen:    0,
			expectedMethod: "empty",
		},
		{
			name:           "count equals keywords length",
			keywords:       []string{"a", "b", "c"},
			count:          3,
			expectedLen:    3,
			expectedMethod: "all",
		},
		{
			name:           "count exceeds keywords length",
			keywords:       []string{"a", "b"},
			count:          5,
			expectedLen:    2,
			expectedMethod: "all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, method := selector.Select(tt.keywords, tt.count)

			if len(result) != tt.expectedLen {
				t.Errorf("Expected %d keywords, got %d", tt.expectedLen, len(result))
			}

			if method != tt.expectedMethod {
				t.Errorf("Expected method %s, got %s", tt.expectedMethod, method)
			}
		})
	}
}
