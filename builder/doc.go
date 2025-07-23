// Package builder provides reusable “functional‐options”‐style building blocks
// for both graph‐ and matrix‐based algorithms. It lives alongside core and matrix
// packages to centralize common configuration, ID schemes, weight distributions,
// and validation logic, keeping implementations DRY, testable, and consistent.
//
// The package offers the following key components:
//
//   - Configuration primitives:
//     – BuilderOption:     a function that mutates builderConfig before use.
//     – builderConfig:     holds RNG, ID‐scheme, weight function, etc.
//   - Vertex‐ID schemes (IDFn implementations):
//     – DefaultIDFn:       decimal strings ("0","1",…).
//     – SymbolIDFn:        single letters ("A","B",…).
//     – ExcelColumnIDFn:   Excel‐style columns ("A","Z","AA",…).
//     – AlphanumericIDFn:  base-36 strings ("0"…"z","10",…).
//     – HexIDFn:           lowercase hexadecimal ("0","a","ff",…).
//   - Edge‐weight distributions (WeightFn implementations):
//     – DefaultWeightFn:   constant weight DefaultEdgeWeight.
//     – ConstantWeightFn:  fixed user-provided value.
//     – UniformWeightFn:   uniform ∼U[min,max].
//     – NormalWeightFn:    Gaussian ∼N(mean,stddev), clipped.
//     – ExponentialWeightFn: exponential ∼Exp(rate).
//   - Validation helpers:
//     – validateMin:       ensure integer ≥ minimum.
//     – validatePartition: ensure bipartition sizes ≥1.
//     – validateProbability: ensure p ∈ [0.0,1.0].
//   - Shared constants:
//     – MinCycleNodes, MinPathNodes, MinStarNodes, MinWheelNodes, MinGridDim.
//     – DefaultEdgeWeight, MinProbability, MaxProbability.
//     – MethodCycle, MethodPath, … tokens for builderErrorf context.
//
// Guarantees:
//
//   - Idempotent configuration: re-running the same builder on g will not duplicate
//     vertices or edges.
//   - Fast‐fail on invalid option parameters via panics in option‐constructors.
//   - Structured runtime errors (builderErrorf) for invalid build parameters,
//     wrapping context tokens for easy filtering.
//   - Documented algorithmic complexity (O(n), O(n²), O(V+E), etc.) per constructor.
//   - Fully testable: all IDFn, WeightFn, BuilderOption, and validation branches
//     are covered by unit tests in builder/builder_test.go.
//
// See individual function documentation for detailed contracts, panic conditions,
// parameter descriptions, and performance notes.
package builder
