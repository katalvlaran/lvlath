// SPDX-License-Identifier: MIT
// Package core_test contains test helpers for lvlath/core.
//
// Purpose:
//   - Provide small, deterministic fixtures and assertion utilities for core.Graph.
//   - Keep tests stdlib-only (no third-party assertion frameworks).
//   - Enforce concurrency-safe testing patterns (no *testing.T usage inside goroutines).

package core_test

import (
	"errors"
	"sort"
	"testing"

	"github.com/katalvlaran/lvlath/core"
)

// Common vertex IDs used across core tests.
const (
	VertexEmpty = ""

	VertexA = "A"
	VertexB = "B"
	VertexC = "C"
	VertexD = "D"

	VertexP = "P"
	VertexQ = "Q"

	VertexU = "U"
	VertexV = "V"

	VertexV1 = "V1"
	VertexV2 = "V2"

	VertexX = "X"
	VertexY = "Y"

	VertexBase = "Base"
)

// Common weights used across core tests (avoid magic numbers in test bodies).
const (
	Weight0 = 0
	Weight1 = 1
	Weight2 = 2
	Weight3 = 3
	Weight5 = 5
	Weight7 = 7
)

// Common concurrency sizes used across core tests (avoid magic numbers in test bodies).
const (
	NAtomicEdgeIDs    = 100
	NConcurrentAdds   = 200
	NConcurrentRounds = 100

	NLoops   = 50
	NReaders = 50
	NCloners = 20
)

// NewGraphFull RETURNS a Graph configured for broad contract coverage.
//
// Implementation:
//   - Stage 1: Call core.NewGraph with WithWeighted/WithMultiEdges/WithLoops.
//   - Stage 2: Return the constructed *core.Graph.
//
// Behavior highlights:
//   - Enables weights to exercise numeric storage.
//   - Enables multi-edges to exercise parallel-edge semantics.
//   - Enables loops to exercise self-loop semantics.
//
// Inputs:
//   - None.
//
// Returns:
//   - *core.Graph: graph with {Weighted=true, MultiEdges=true, Loops=true}.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic configuration (no randomness).
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - This is a TEST-FIXTURE constructor; it intentionally does not belong to production API.
//   - Keeping it in test_helpers_test.go makes the test policy centralized and auditable.
//
// AI-Hints:
//   - Use NewGraphFull when you need maximum feature surface with minimal boilerplate.
//   - For strict-policy tests, prefer building graphs explicitly (NewGraph + options) to isolate constraints.
func NewGraphFull() *core.Graph {
	return core.NewGraph(core.WithWeighted(), core.WithMultiEdges(), core.WithLoops())
}

// MustNoError FAILS the test if err != nil.
//
// Implementation:
//   - Stage 1: If err == nil, return immediately.
//   - Stage 2: Mark helper and abort via t.Fatalf with operation context.
//
// Behavior highlights:
//   - Makes error sites explicit and consistent.
//   - Keeps test code focused on contracts, not boilerplate.
//
// Inputs:
//   - t: *testing.T.
//   - err: error to validate.
//   - op: short operation label (e.g., "AddEdge(A,B,1)").
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if err != nil.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keep op stable and descriptive; avoid long formatted strings.
//
// AI-Hints:
//   - Prefer operation-like labels for op (call signatures) to speed up failure triage.
func MustNoError(t *testing.T, err error, op string) {
	t.Helper()

	if err == nil {
		return
	}

	t.Fatalf("%s: unexpected error: %v", op, err)
}

// MustErrorIs FAILS the test if !errors.Is(err, target).
//
// Implementation:
//   - Stage 1: Evaluate errors.Is(err, target).
//   - Stage 2: t.Fatalf with target and actual error.
//
// Behavior highlights:
//   - Enforces sentinel-error contracts precisely.
//
// Inputs:
//   - t: *testing.T.
//   - err: error to inspect.
//   - target: expected sentinel error.
//   - op: operation label for context.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if the sentinel does not match.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(depth) of wrapped error chain, Space O(1).
//
// Notes:
//   - Use only for sentinel-style contracts (core.Err*).
//
// AI-Hints:
//   - When adding new sentinel errors, update tests to assert errors.Is, not string matching.
func MustErrorIs(t *testing.T, err error, target error, op string) {
	t.Helper()

	if errors.Is(err, target) {
		return
	}

	t.Fatalf("%s: want errors.Is(err,%v)=true; got err=%v", op, target, err)
}

// MustTrue FAILS the test if cond is false.
//
// Implementation:
//   - Stage 1: If cond is true, return.
//   - Stage 2: t.Fatalf with op.
//
// Behavior highlights:
//   - Minimizes repetitive "if !x { t.Fatalf }" patterns.
//
// Inputs:
//   - t: *testing.T.
//   - cond: predicate.
//   - op: operation label.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if cond==false.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Use op to describe the invariant, not the mechanism.
//
// AI-Hints:
//   - Prefer naming the invariant: "Vertices() must be sorted", "HasEdge must be safe", etc.
func MustTrue(t *testing.T, cond bool, op string) {
	t.Helper()

	if cond {
		return
	}

	t.Fatalf("%s: predicate is false", op)
}

// MustFalse FAILS the test if cond is true.
//
// Implementation:
//   - Stage 1: If cond is false, return.
//   - Stage 2: t.Fatalf with op.
//
// Behavior highlights:
//   - Symmetric to MustTrue.
//
// Inputs:
//   - t: *testing.T.
//   - cond: predicate.
//   - op: operation label.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if cond==true.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Keep op describing the expected falsehood.
//
// AI-Hints:
//   - Use for negative contracts: "duplicate AddVertex must not change count", "HasEdge on unknown vertices must be false".
func MustFalse(t *testing.T, cond bool, op string) {
	t.Helper()

	if !cond {
		return
	}

	t.Fatalf("%s: predicate is true", op)
}

// MustEqualInt FAILS if got != want.
//
// Implementation:
//   - Stage 1: Compare ints.
//   - Stage 2: t.Fatalf with got/want.
//
// Behavior highlights:
//   - Avoids generic helpers to keep test style close to stdlib and explicit.
//
// Inputs:
//   - t: *testing.T.
//   - got, want: int values.
//   - op: operation label.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Prefer for counts (Edges/Vertices/Neighbors).
//
// AI-Hints:
//   - Use MustEqualInt(len(x), N, "...") to keep failures actionable.
func MustEqualInt(t *testing.T, got, want int, op string) {
	t.Helper()

	if got == want {
		return
	}

	t.Fatalf("%s: got=%d want=%d", op, got, want)
}

// MustEqualString FAILS if got != want.
//
// Implementation:
//   - Stage 1: Compare strings.
//   - Stage 2: t.Fatalf with got/want.
//
// Behavior highlights:
//   - Explicit, readable comparisons.
//
// Inputs:
//   - t: *testing.T.
//   - got, want: strings.
//   - op: operation label.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on mismatch.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(n) compare, Space O(1).
//
// Notes:
//   - Use for vertex IDs, edge IDs, endpoints.
//
// AI-Hints:
//   - Prefer comparing endpoints via GetEdge(id) rather than scanning Edges().
func MustEqualString(t *testing.T, got, want string, op string) {
	t.Helper()

	if got == want {
		return
	}

	t.Fatalf("%s: got=%q want=%q", op, got, want)
}

// MustNotEqualString FAILS if got == notWant.
//
// Implementation:
//   - Stage 1: Compare strings.
//   - Stage 2: t.Fatalf when equal.
//
// Behavior highlights:
//   - Used to assert non-colliding edge IDs.
//
// Inputs:
//   - t: *testing.T.
//   - got, notWant: strings.
//   - op: operation label.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure when equal.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(n) compare, Space O(1).
//
// Notes:
//   - Keep op describing the collision you are preventing.
//
// AI-Hints:
//   - For view/subgraph ID-carry tests: assert newID != copiedID.
func MustNotEqualString(t *testing.T, got, notWant string, op string) {
	t.Helper()

	if got != notWant {
		return
	}

	t.Fatalf("%s: got=%q must_not_equal=%q", op, got, notWant)
}

// MustNonEmptyString FAILS if s == "".
//
// Implementation:
//   - Stage 1: Check non-empty.
//   - Stage 2: t.Fatalf on empty.
//
// Behavior highlights:
//   - Used for ID generation contracts.
//
// Inputs:
//   - t: *testing.T.
//   - s: string to validate.
//   - op: operation label.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if s is empty.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// Notes:
//   - Useful when ID format is not part of the contract but non-emptiness is.
//
// AI-Hints:
//   - Prefer this over checking prefixes unless prefix is an explicit API contract.
func MustNonEmptyString(t *testing.T, s string, op string) {
	t.Helper()

	if s != "" {
		return
	}

	t.Fatalf("%s: expected non-empty string", op)
}

//// MustLenInt validates the expected length for slices/maps/strings where len(x) is meaningful.
//func MustLenInt(t *testing.T, gotLen, wantLen int) {
//	t.Helper()
//	msg := fmt.Sprintf(format, args...)
//	if gotLen == wantLen {
//		return
//	}
//	t.Fatalf(format, args...)
//
//	Must(t, gotLen == wantLen, "%s: got_len=%d want_len=%d", msg, gotLen, wantLen)
//}

// MustSortedStrings FAILS if ids are not sorted ascending.
//
// Implementation:
//   - Stage 1: Use sort.StringsAreSorted.
//   - Stage 2: t.Fatalf with the slice.
//
// Behavior highlights:
//   - Enforces deterministic ordering contracts (Vertices/Edges/Neighbors).
//
// Inputs:
//   - t: *testing.T.
//   - ids: slice to validate.
//   - op: operation label.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if not sorted.
//
// Determinism:
//   - Deterministic.
//
// Complexity:
//   - Time O(n), Space O(1).
//
// Notes:
//   - Only checks ordering, not uniqueness.
//
// AI-Hints:
//   - Use for determinism guarantees: stable outputs simplify downstream algorithms.
func MustSortedStrings(t *testing.T, ids []string, op string) {
	t.Helper()

	if sort.StringsAreSorted(ids) {
		return
	}

	t.Fatalf("%s: not sorted asc: %v", op, ids)
}

// MustSameStringSet FAILS if a and b are not equal as sets (order-independent).
//
// Implementation:
//   - Stage 1: Copy and sort both slices.
//   - Stage 2: Compare element-wise.
//
// Behavior highlights:
//   - Replaces third-party ElementsMatch with deterministic stdlib logic.
//
// Inputs:
//   - t: *testing.T.
//   - a,b: slices to compare as sets.
//   - op: operation label.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure on mismatch.
//
// Determinism:
//   - Deterministic (sort-based).
//
// Complexity:
//   - Time O(n log n), Space O(n).
//
// Notes:
//   - Requires equal lengths; duplicates are treated as multiplicities.
//
// AI-Hints:
//   - Use when vertex ordering is allowed to vary but membership must be identical.
func MustSameStringSet(t *testing.T, a, b []string, op string) {
	t.Helper()

	if len(a) != len(b) {
		t.Fatalf("%s: len(a)=%d len(b)=%d; a=%v b=%v", op, len(a), len(b), a, b)
	}

	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)

	var i int
	for i = 0; i < len(aa); i++ {
		if aa[i] != bb[i] {
			t.Fatalf("%s: set mismatch at i=%d; a=%v b=%v", op, i, aa, bb)
		}
	}
}

// ExtractEdgeIDs RETURNS edge IDs preserving the incoming slice order.
//
// Implementation:
//   - Stage 1: Allocate output slice sized to edges.
//   - Stage 2: Copy Edge.ID into output.
//
// Behavior highlights:
//   - Small utility for edge-inventory comparisons.
//
// Inputs:
//   - edges: []*core.Edge.
//
// Returns:
//   - []string: IDs in the same order.
//
// Errors:
//   - None.
//
// Determinism:
//   - Deterministic for a fixed input slice.
//
// Complexity:
//   - Time O(n), Space O(n).
//
// Notes:
//   - Prefer comparing sets via MustSameStringSet if order is not part of the contract.
//
// AI-Hints:
//   - Combine with MustSortedStrings if Edges() ordering is contractual.
func ExtractEdgeIDs(edges []*core.Edge) []string {
	out := make([]string, len(edges))

	var i int
	for i = 0; i < len(edges); i++ {
		out[i] = edges[i].ID
	}

	return out
}

// MustNoErrorsFromChan FAILS the test if any non-nil error is received.
//
// Implementation:
//   - Stage 1: Range over errCh until it is closed.
//   - Stage 2: On first non-nil error, fail via t.Fatalf.
//
// Behavior highlights:
//   - Enforces the rule "no *testing.T usage inside goroutines":
//     goroutines send errors to a channel; the parent goroutine validates.
//
// Inputs:
//   - t: *testing.T.
//   - errCh: receive-only channel of errors (must be closed by caller).
//   - op: operation label describing the concurrent scenario.
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal test failure if any error is non-nil.
//
// Determinism:
//   - Deterministic for a fixed schedule of produced errors.
//
// Complexity:
//   - Time O(k) where k is number of channel items, Space O(1).
//
// Notes:
//   - If your concurrent scenario permits certain sentinel outcomes (e.g., ErrEdgeNotFound),
//     filter them before sending to errCh.
//
// AI-Hints:
//   - In concurrent tests, send only unexpected errors to errCh to keep failures signal-rich.
func MustNoErrorsFromChan(t *testing.T, errCh <-chan error, op string) {
	t.Helper()

	for err := range errCh {
		if err == nil {
			continue
		}
		t.Fatalf("%s: unexpected concurrent error: %v", op, err)
	}
}
