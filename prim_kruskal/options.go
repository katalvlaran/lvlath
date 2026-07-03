// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package prim_kruskal defines option assembly for the canonical MST facade.
// Options are validated before graph adaptation and before any algorithm allocation.
package prim_kruskal

// Option configures MinimumSpanningTree through a safe, error-returning option model.
//
// Implementation:
//   - Stage 1: applyOptions starts from default options.
//   - Stage 2: Options mutate exactly one policy dimension.
//   - Stage 3: applyOptions validates derived invariants.
//
// Behavior highlights:
//   - Nil options return ErrNilOption.
//   - Last writer wins for repeated algorithm/mode/root setters.
//   - Prim strict tree mode requires an explicit root.
//   - Prim forest mode may omit root and then uses lexicographic component roots.
//
// AI-Hints:
//   - Do not reintroduce panic-based option validation; use sentinel errors.
type Option func(*Options) error

// Options configures the canonical MST facade.
// It fixes algorithm selection, connectivity mode, and the optional Prim root before graph adaptation.
//
// Implementation:
//   - Stage 1: DefaultOptions creates Kruskal + strict tree policy.
//   - Stage 2: Option setters update one policy field at a time.
//   - Stage 3: ApplyOptions validates algorithm, mode, nil options, and Prim root invariants.
//
// Behavior highlights:
//   - AlgorithmKruskal is the default because it does not require a root.
//   - ModeStrictTree is the default because forest mode is a contract-changing opt-in.
//   - Root is required only for AlgorithmPrim in ModeStrictTree.
//   - AlgorithmPrim with ModeForest may omit Root and then starts from deterministic vertex order.
//
// Inputs:
//   - Algorithm: AlgorithmKruskal or AlgorithmPrim.
//   - Mode: ModeStrictTree or ModeForest.
//   - Root: optional vertex ID consumed by Prim.
//
// Notes:
//   - Options is a value-type policy snapshot.
//   - Graph topology is not inspected during option assembly.
//
// AI-Hints:
//   - Do not infer forest mode from graph disconnection; use WithForest explicitly.
//   - Do not validate vertex existence in ApplyOptions; that belongs to snapshot/kernel validation.
type Options struct {
	// Algorithm selects the MST algorithm used by MinimumSpanningTree.
	Algorithm Algorithm

	// Mode selects strict spanning tree or explicit spanning forest semantics.
	Mode Mode

	// Root is the explicit starting vertex for Prim; Kruskal ignores it.
	Root string
}

// DefaultOptions returns the canonical MST policy.
//
// Implementation:
//   - Stage 1: Set algorithm to AlgorithmKruskal.
//   - Stage 2: Set mode to ModeStrictTree.
//   - Stage 3: Leave Root empty because Kruskal does not consume it.
//
// Returns:
//   - options: a value configured for Kruskal.
//
// Determinism:
//   - O(1) pure value construction; no graph state is read.
//
// Complexity:
//   - Time O(1), Space O(1).
func DefaultOptions() Options {
	return Options{
		Algorithm: AlgorithmKruskal,
		Mode:      ModeStrictTree,
		Root:      "",
	}
}

// ApplyOptions assembles and validates canonical MST options.
//
// Implementation:
//   - Stage 1: Start from defaultOptions.
//   - Stage 2: Apply every user option in order.
//   - Stage 3: Validate algorithm/mode constants.
//   - Stage 4: Enforce Prim strict-root invariant.
//
// Behavior highlights:
//   - Last option wins for repeated setters.
//   - Forest mode is explicit and never inferred from graph shape.
//
// Inputs:
//   - user: variadic Option list.
//
// Returns:
//   - options: finalized internal policy.
//   - error: sentinel option failure.
//
// Errors:
//   - ErrNilOption for nil option values.
//   - ErrUnsupportedAlgorithm for unknown algorithm values.
//   - ErrInvalidOption for unknown mode values.
//   - ErrEmptyRoot when Prim strict tree mode has no root.
//
// Determinism:
//   - Options are applied in caller-provided order.
//
// Complexity:
//   - Time O(k), Space O(1), where k is the number of options.
//
// AI-Hints:
//   - Do not inspect graph topology here; option assembly must stay graph-independent.
func ApplyOptions(user ...Option) (Options, error) {
	cfg := DefaultOptions()

	for _, opt := range user {
		if opt == nil {
			return Options{}, ErrNilOption
		}
		if err := opt(&cfg); err != nil {
			return Options{}, err
		}
	}

	switch cfg.Algorithm {
	case AlgorithmKruskal, AlgorithmPrim:
	default:
		return Options{}, ErrUnsupportedAlgorithm
	}

	switch cfg.Mode {
	case ModeStrictTree, ModeForest:
	default:
		return Options{}, ErrInvalidOption
	}

	if cfg.Algorithm == AlgorithmPrim && cfg.Mode == ModeStrictTree && cfg.Root == "" {
		return Options{}, ErrEmptyRoot
	}

	return cfg, nil
}

// WithAlgorithm selects the MST algorithm used by MinimumSpanningTree.
//
// Implementation:
//   - Stage 1: Check the requested algorithm against supported constants.
//   - Stage 2: Store the algorithm into Options.
//
// Behavior highlights:
//   - Last writer wins when multiple WithAlgorithm options are supplied.
//   - This option does not validate graph topology.
//
// Inputs:
//   - algorithm: AlgorithmKruskal or AlgorithmPrim.
//
// Returns:
//   - Option: safe error-returning option setter.
//
// Errors:
//   - ErrUnsupportedAlgorithm when algorithm is not AlgorithmKruskal or AlgorithmPrim.
//
// Determinism:
//   - Pure constant validation; no graph state is read.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not accept arbitrary strings silently; unsupported algorithms must remain sentinel-classified.
func WithAlgorithm(algorithm Algorithm) Option {
	return func(cfg *Options) error {
		switch algorithm {
		case AlgorithmKruskal, AlgorithmPrim:
			cfg.Algorithm = algorithm
			return nil
		default:
			return ErrUnsupportedAlgorithm
		}
	}
}

// WithRoot sets the explicit Prim root.
// It validates only the empty-string contract; vertex existence is checked after graph snapshot construction.
//
// Implementation:
//   - Stage 1: Reject empty root with ErrEmptyRoot.
//   - Stage 2: Store root in Options.
//
// Behavior highlights:
//   - Last writer wins when multiple WithRoot options are supplied.
//   - Kruskal ignores Root.
//   - Prim strict tree mode requires Root.
//
// Inputs:
//   - root: non-empty vertex ID.
//
// Returns:
//   - Option: safe error-returning option setter.
//
// Errors:
//   - ErrEmptyRoot when root is empty.
//
// Determinism:
//   - Pure value validation; no graph state is read.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not check graph.HasVertex here; option assembly must remain independent of graph topology.
func WithRoot(root string) Option {
	return func(cfg *Options) error {
		if root == "" {
			return ErrEmptyRoot
		}
		cfg.Root = root
		return nil
	}
}

// WithForest enables explicit minimum spanning forest mode.
// Disconnected graphs return an MSTResult containing one tree per connected component.
//
// Implementation:
//   - Stage 1: Set Options.Mode to ModeForest.
//   - Stage 2: Leave Algorithm and Root unchanged.
//
// Behavior highlights:
//   - Forest mode is opt-in and never inferred from graph shape.
//   - Kruskal forest mode uses DSU component roots.
//   - Prim forest mode grows components in snapshot vertex order.
//
// Returns:
//   - Option: safe error-returning option setter.
//
// Determinism:
//   - Pure mode assignment; graph traversal order is fixed later by the selected kernel.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Do not silently downgrade ErrDisconnected into a forest result without this option.
func WithForest() Option {
	return func(cfg *Options) error {
		cfg.Mode = ModeForest
		return nil
	}
}

// WithStrictTree enables strict spanning tree mode.
// Disconnected graphs return ErrDisconnected instead of a forest result.
//
// Implementation:
//   - Stage 1: Set Options.Mode to ModeStrictTree.
//   - Stage 2: Leave Algorithm and Root unchanged.
//
// Behavior highlights:
//   - This is the default mode.
//   - Use it to override an earlier WithForest option in the same option list.
//
// Returns:
//   - Option: safe error-returning option setter.
//
// Determinism:
//   - Last writer wins for repeated mode setters.
//
// Complexity:
//   - Time O(1), Space O(1).
//
// AI-Hints:
//   - Keep strict mode as the default; forest mode changes the mathematical contract.
func WithStrictTree() Option {
	return func(cfg *Options) error {
		cfg.Mode = ModeStrictTree
		return nil
	}
}
