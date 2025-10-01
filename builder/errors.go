// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// errors.go — sentinel errors for the builder package.
//
// Error policy (explicit and strict):
//   • Only sentinel variables (package-level) are exposed.
//   • Callers MUST use errors.Is(err, ErrX) to branch on semantics.
//   • Sentinels are NEVER wrapped with formatted strings at definition site.
//   • Implementations SHOULD attach context using `%w` (see AI-Hints below).
//   • Algorithms MUST NOT panic at runtime; validation panics are confined to
//     option constructor functions (WithX...), per lvlath 99-rules.
//
// AI-Hints (practical guidance for implementers and LLMs):
//   • Wrap lower-level errors with method context: wrapf(MethodCycle, "AddEdge(u,v)", err).
//   • Return ONLY these sentinels for validation classes (size/probability/rng/mode).
//   • Do NOT stringify parameters into sentinel definitions; use %w wrapping instead.
//   • Check with errors.Is in tests and production code; avoid string comparisons.

package builder

import (
	"errors"
	"fmt"
)

// ErrTooFewVertices indicates that a numeric parameter (e.g., n, rows, cols, degree)
// is smaller than the allowed minimum for the requested constructor.
// Classification: Validation error (parameters).
// Typical origins: Cycle/Path/Grid/RandomRegular (n,d constraints), etc.
// Usage: if errors.Is(err, ErrTooFewVertices) { /* report invalid size */ }.
var ErrTooFewVertices = errors.New("builder: parameter too small")

// ErrInvalidProbability indicates that a probability value is outside the
// closed interval [0,1]. This covers RandomSparse(p) and any probability-based
// utilities if introduced in the future.
// Usage: if errors.Is(err, ErrInvalidProbability) { /* clamp or reject p */ }.
var ErrInvalidProbability = errors.New("builder: probability out of range")

// ErrNeedRandSource indicates that a stochastic constructor requires a non-nil
// *rand.Rand in the resolved builderConfig (e.g., WithSeed/WithRand must be set).
// Typical origins: RandomSparse/RandomRegular without RNG.
// Usage: if errors.Is(err, ErrNeedRandSource) { /* supply seeded RNG */ }.
var ErrNeedRandSource = errors.New("builder: rng is required")

// ErrUnsupportedGraphMode indicates the invoked constructor is incompatible with
// the current core.Graph mode (e.g., RandomRegular on a directed graph, or a
// simple-graph requirement violated by mode flags).
// Usage: if errors.Is(err, ErrUnsupportedGraphMode) { /* switch graph mode */ }.
var ErrUnsupportedGraphMode = errors.New("builder: unsupported graph mode")

// ErrConstructFailed indicates that the builder exhausted permitted strategies
// or attempts (e.g., stub-matching retries for RandomRegular) and could not
// construct a topology without breaking invariants (no loops / no multiedges,
// connectivity/degree constraints, etc.).
// Usage: if errors.Is(err, ErrConstructFailed) { /* retry with different seed */ }.
var ErrConstructFailed = errors.New("builder: construction failed")

// ErrUnknownLetter indicates an unsupported rune/symbol was requested in Letters/Word
// constructors according to the canonical alphabet spec (letters_spec.go).
// Usage: if errors.Is(err, ErrUnknownLetter) { /* validate input word/alphabet */ }.
var ErrUnknownLetter = errors.New("builder: unknown letter")

// ErrBadSize indicates invalid sizes/lengths for sequence datasets (Pulse/Chirp/OHLC)
// or other non-topology sizes (e.g., days < 1, n < 1 for sequences).
// Usage: if errors.Is(err, ErrBadSize) { /* fix n/days */ }.
var ErrBadSize = errors.New("builder: invalid size/length")

// ErrOptionViolation indicates that a WithX(...) option constructor received a
// meaningless or unsafe value (e.g., WithAmplitude(A<=0), WithNoise(sigma<0),
// WithIDScheme(nil), WithRand(nil)). NOTE: such violations SHOULD panic in the
// option constructor by design; this sentinel is reserved for validations that
// must surface as errors rather than panics (e.g., runtime option resolution).
// Usage: if errors.Is(err, ErrOptionViolation) { /* correct option values */ }.
var ErrOptionViolation = errors.New("builder: invalid option value")

// builderErrorf wraps an inner error message with the given method context.
// It returns an error of the form "<Method>: <formatted message>".
//
// Parameters:
//   - method: canonical constructor name, e.g. MethodCycle.
//   - format: format string for the inner message.
//   - args:   values for the format placeholders.
//
// Complexity: O(len(format) + Σlen(args)), negligible for our use.
func builderErrorf(method, format string, args ...interface{}) error {
	// Build the inner message using fmt.Sprintf
	inner := fmt.Sprintf(format, args...)
	// Prefix with the method name and return a new error
	return fmt.Errorf("%s: %s", method, inner)
}

// --- Implementation Notes ----------------------------------------------------
//
// 1) Wrapping style (required):
//      return wrapf(MethodRandomSparse, "rng is required", ErrNeedRandSource)
//    This preserves the sentinel (ErrNeedRandSource) for errors.Is while adding
//    a deterministic context prefix "RandomSparse: rng is required".
//
// 2) Priority (tie-break guidance when multiple validations fail):
//    • ErrTooFewVertices       — size/domain checks first (n, rows, cols, degree).
//    • ErrInvalidProbability   — then probability ranges.
//    • ErrNeedRandSource       — then RNG presence for stochastic builders.
//    • ErrUnsupportedGraphMode — then mode compatibility (directed/loops/multi).
//    • ErrConstructFailed      — only after all retries/strategies are exhausted.
//    • Letters/Datasets (ErrUnknownLetter / ErrBadSize / ErrOptionViolation) —
//      apply in their respective subsystems (letters_spec / sequences).
//
// 3) Testing guidance:
//    Use table tests asserting errors.Is(err, ErrX). Avoid matching error strings.
//    Provide edge cases: n=0, p=-0.1, rng=nil, directed-mode for RandomRegular,
//    unknown rune in Letters, n<1 days for OHLC, etc.
//
// 4) Complexity impact:
//    Sentinels add O(1) overhead. Wrapping via %w is also O(1). No allocations
//    beyond the error value itself at definition time.
//
// 5) Compatibility:
//    These names and messages are stable and form part of the public contract.
//    Do not rename or change messages; add NEW sentinels only under a versioned
//    migration note in doc.go if absolutely necessary.
