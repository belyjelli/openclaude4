package core

import "fmt"

// IterationLimitError is returned when the model↔tool loop exceeds [Agent.MaxIterations]
// or [DefaultMaxIterations] without finishing the turn.
type IterationLimitError struct {
	MaxIterations int
}

func (e *IterationLimitError) Error() string {
	if e == nil {
		return "agent: iteration limit exceeded"
	}
	return fmt.Sprintf("agent: exceeded %d tool iterations", e.MaxIterations)
}
