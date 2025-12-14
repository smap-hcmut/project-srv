package usecase

import (
	"math/rand"
	"time"
)

// keywordSelector defines the private interface for keyword selection algorithms
type keywordSelector interface {
	Select(keywords []string, count int) ([]string, string)
}

// randomSelector implements random keyword selection
type randomSelector struct {
	rand *rand.Rand
}

// newRandomSelector creates a new random selector
func newRandomSelector() keywordSelector {
	source := rand.NewSource(time.Now().UnixNano())
	return &randomSelector{rand: rand.New(source)}
}

// Select randomly selects the specified number of keywords from the input slice
func (s *randomSelector) Select(keywords []string, count int) ([]string, string) {
	if count <= 0 {
		return []string{}, "empty"
	}

	if count >= len(keywords) {
		return keywords, "all"
	}

	// Create a copy to avoid modifying the original slice
	shuffled := make([]string, len(keywords))
	copy(shuffled, keywords)

	// Shuffle using Fisher-Yates algorithm
	s.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count], "random"
}
