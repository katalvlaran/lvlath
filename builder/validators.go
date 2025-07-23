// Package builder provides validation helpers to enforce
// parameter contracts in GraphConstructor factories.
//
// Each function returns a formatted error via builderErrorf
// when its precondition is violated.
package builder

// validateMin ensures that the provided integer 'got' is ≥ 'min'.
// Returns an error "<Method>: parameter must be ≥ <min>, got <got>" otherwise.
//
// Parameters:
//   - method: constructor name constant, e.g. MethodCycle.
//   - got:    actual value supplied by user.
//   - min:    minimal acceptable value.
//
// Complexity: O(1) time and space.
func validateMin(method string, got, min int) error {
	if got < min {
		// wrap with builderErrorf to maintain uniform error prefix
		return builderErrorf(method, "parameter must be ≥ %d, got %d", min, got)
	}

	return nil
}

// validatePartition checks that the two integers n1 and n2 are each ≥ 1.
// Used by CompleteBipartite to enforce non-empty partitions.
// Returns "<Method>: partition sizes must be ≥ MaxPartition, got <n1> and <n2>" on failure.
//
// Parameters:
//   - method: canonical constructor name.
//   - n1, n2: sizes of the two partitions.
//
// Complexity: O(1) time and space.
func validatePartition(method string, n1, n2 int) error {
	if n1 < MaxPartition || n2 < MaxPartition {
		return builderErrorf(method, "partition sizes must be ≥ 1, got %d and %d", n1, n2)
	}

	return nil
}

// validateProbability enforces p ∈ [MinProbability, MaxProbability].
// Used by RandomSparse. Returns
// "<Method>: probability must be in [0.0,1.0], got <p>" if out of range.
//
// Parameters:
//   - method: canonical constructor name.
//   - p:      probability value to validate.
//
// Complexity: O(1) time and space.
func validateProbability(method string, p float64) error {
	if p < MinProbability || p > MaxProbability {
		return builderErrorf(method, "probability must be in [%.1f,%.1f], got %f", MinProbability, MaxProbability, p)
	}

	return nil
}
