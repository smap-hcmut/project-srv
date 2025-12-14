package sampling

import "errors"

var (
	ErrInvalidPercentage   = errors.New("percentage must be between 0 and 100")
	ErrInvalidMinKeywords  = errors.New("minimum keywords must be at least 1")
	ErrInvalidMaxKeywords  = errors.New("maximum keywords cannot be less than minimum")
	ErrInvalidTimeEstimate = errors.New("keyword time estimate must be positive")
	ErrInvalidStrategy     = errors.New("invalid sampling strategy")
	ErrMissingUserID       = errors.New("user ID is required")
	ErrNilKeywords         = errors.New("keywords array cannot be nil")
)
