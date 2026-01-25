package keyword

// ValidateInput contains keywords to validate
type ValidateInput struct {
	Keywords []string `json:"keywords" validate:"required,min=1"`
}

// TestInput contains keywords to test (dry run)
type TestInput struct {
	Keywords []string `json:"keywords" validate:"required,min=1"`
}

// SuggestOutput contains suggested niche and negative keywords
type SuggestOutput struct {
	Niche    []string `json:"niche"`
	Negative []string `json:"negative"`
}

// ValidateOutput contains validated keywords
type ValidateOutput struct {
	ValidKeywords []string `json:"valid_keywords"`
}

// TestOutput contains test results from keyword dry run
type TestOutput struct {
	Results []interface{} `json:"results"`
}
