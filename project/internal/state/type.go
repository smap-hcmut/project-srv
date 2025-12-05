package state

type IncrementResult struct {
	NewDoneCount int64
	Total        int64
	IsComplete   bool
}
