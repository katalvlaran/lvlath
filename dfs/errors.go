// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package dfs

import "errors"

// Package-level sentinel errors classify stable DFS failure categories.
//
// AI-Hints:
//   - Use errors.Is to classify package errors.
//   - Preserve underlying causes with %w; never flatten them into strings.
//   - Package sentinels describe failure categories, while wrapped causes preserve source detail.
var (
	// ErrGraphNil reports that a nil graph pointer was passed to a package algorithm.
	ErrGraphNil = errors.New("dfs: graph is nil")

	// ErrStartVertexNotFound reports that the requested DFS start vertex does not exist.
	ErrStartVertexNotFound = errors.New("dfs: start vertex not found")

	// ErrNeighborFetch reports that neighbor enumeration failed during traversal.
	ErrNeighborFetch = errors.New("dfs: neighbor iteration error")

	// ErrCycleDetected reports that a cycle was detected where acyclicity is required.
	ErrCycleDetected = errors.New("dfs: cycle detected")

	// ErrGraphNotDirected reports that a directed-only algorithm was applied to a non-directed graph.
	ErrGraphNotDirected = errors.New("dfs: operation requires directed graph")

	// ErrOptionViolation reports invalid explicit option input.
	ErrOptionViolation = errors.New("dfs: invalid option")
)
