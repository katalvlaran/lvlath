package flow

import "fmt"

// ErrSourceNotFound is returned when the specified source vertex is missing.
var ErrSourceNotFound = fmt.Errorf("flow: %w", errSourceNotFound)
var errSourceNotFound = fmt.Errorf("source vertex not found")

// ErrSinkNotFound is returned when the specified sink vertex is missing.
var ErrSinkNotFound = fmt.Errorf("flow: %w", errSinkNotFound)
var errSinkNotFound = fmt.Errorf("sink vertex not found")

// EdgeError is returned when an edge has a negative capacity.
type EdgeError struct {
	From, To string
	Cap      float64
}

func (e EdgeError) Error() string {
	return fmt.Sprintf("flow: negative capacity on edge %q→%q: %g", e.From, e.To, e.Cap)
}

// FlowOptions configures all max-flow algorithms.
//   - Epsilon: treat capacities ≤ Epsilon as zero (default 1e-9).
//   - Verbose: if true, logs each augmentation when possible.
//   - LevelRebuildInterval: for Dinic, rebuild level graph every N augmentations.
type FlowOptions struct {
	Epsilon              float64
	Verbose              bool
	LevelRebuildInterval int
}
