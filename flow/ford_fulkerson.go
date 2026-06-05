// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package flow implements deterministic DFS Ford-Fulkerson compatibility mode.
//
// This kernel is useful for small or educational networks. Dinic remains the
// default canonical algorithm because it has stronger practical behavior.
package flow

import (
	"context"
	"math"
)

// runFordFulkerson computes maximum flow through deterministic DFS augmentations.
// It repeatedly finds any admissible source-sink path and pushes its bottleneck.
//
// Implementation:
//   - Stage 1: Check cancellation before each DFS search.
//   - Stage 2: Find one depth-first augmenting path.
//   - Stage 3: Stop when no positive residual path exists.
//   - Stage 4: Apply residual updates through addResidual.
//   - Stage 5: Notify observer and publish final result.
//
// Behavior highlights:
//   - DFS path choice is deterministic because rn.adj is sorted.
//   - This kernel is simple but may require many augmentations.
//   - Observer events include concrete path witnesses.
//
// Inputs:
//   - source: source vertex ID already validated by MaxFlow.
//   - sink: sink vertex ID already validated by MaxFlow.
//   - rn: deterministic residual network.
//   - cfg: finalized runtime options.
//
// Returns:
//   - *MaxFlowResult: canonical result with residual graph and min-cut certificate.
//   - error: nil on success or sentinel-classified interruption.
//
// Errors:
//   - context cancellation errors.
//   - ErrObserverFailure from notifyAugmentation.
//   - core graph construction errors from finalizeResult.
//
// Determinism:
//   - Stack expansion scans rn.adj[from] in sorted order.
//   - Equal choices follow stable lexical residual adjacency order.
//
// Complexity:
//   - Time O(A * F) for integral-like capacity regimes, where F is the number
//     of augmentation units/path pushes under the chosen capacities.
//   - For arbitrary real capacities, Ford-Fulkerson has weaker termination
//     guarantees than Edmonds-Karp or Dinic.
//   - Space O(V + A) including residual state and DFS maps.
//
// Notes:
//   - Prefer Dinic for production-sized networks.
//   - Prefer Edmonds-Karp when shortest augmenting path witnesses are needed.
//
// AI-Hints:
//   - Do not claim Edmonds-Karp complexity for this DFS kernel.
//   - Do not scan rn.cap maps; that reintroduces nondeterministic DFS.
func runFordFulkerson(
	source, sink string,
	rn *residualNetwork,
	cfg options,
) (*MaxFlowResult, error) {
	maxFlow := 0.0
	augmentations := 0

	for {
		if err := cfg.ctx.Err(); err != nil {
			return newPartialResult(source, sink, AlgorithmFordFulkerson, maxFlow, augmentations), err
		}

		parent, delta, found, err := findDepthFirstAugmentingPath(
			cfg.ctx,
			rn,
			source,
			sink,
			cfg.epsilon,
		)
		if err != nil {
			return newPartialResult(source, sink, AlgorithmFordFulkerson, maxFlow, augmentations), err
		}
		if !found || delta <= cfg.epsilon {
			break
		}

		// The limit is checked only after proving that one more augmenting path exists.
		// This prevents false ErrAugmentationLimit when the optimum is reached exactly
		// on the previous augmentation.
		if err = checkAugmentationLimit(augmentations, cfg.maxAugmentations); err != nil {
			return newPartialResult(source, sink, AlgorithmFordFulkerson, maxFlow, augmentations), err
		}

		for vertexID := sink; vertexID != source; vertexID = parent[vertexID] {
			previousID := parent[vertexID]
			addResidual(rn, previousID, vertexID, delta, cfg.epsilon)
		}

		maxFlow += delta
		augmentations++

		if err = notifyAugmentation(cfg.ctx, cfg, AugmentationEvent{
			Algorithm: AlgorithmFordFulkerson,
			Path:      reconstructPath(parent, source, sink),
			Delta:     delta,
			Total:     maxFlow,
			Index:     augmentations,
		}); err != nil {
			return newPartialResult(source, sink, AlgorithmFordFulkerson, maxFlow, augmentations), err
		}
	}

	return finalizeResult(
		source,
		sink,
		rn,
		cfg,
		AlgorithmFordFulkerson,
		maxFlow,
		augmentations,
		false,
	)
}

// findDepthFirstAugmentingPath finds one deterministic DFS residual path.
// It returns parent links and bottleneck capacity for the discovered path.
//
// Implementation:
//   - Stage 1: Initialize explicit stack, parent map, and visited set.
//   - Stage 2: Pop vertices in LIFO order.
//   - Stage 3: Scan deterministic residual neighbors with capacity > epsilon.
//   - Stage 4: Stop when sink is discovered.
//
// Behavior highlights:
//   - Uses an explicit stack instead of recursive DFS.
//   - Avoids call-stack growth on long residual paths.
//   - Marks vertices when pushed to prevent duplicate stack entries.
//
// Inputs:
//   - ctx: cancellation context.
//   - rn: residual network.
//   - source: source vertex ID.
//   - sink: sink vertex ID.
//   - epsilon: residual threshold.
//
// Returns:
//   - map[string]string: parent links for the discovered path.
//   - float64: path bottleneck capacity.
//   - bool: true when sink was reached.
//   - error: context cancellation error.
//
// Errors:
//   - ctx.Err() when cancellation is observed.
//
// Determinism:
//   - Candidate arcs are considered in rn.adj[from] order.
//   - LIFO stack makes the exact path deterministic for fixed rn.adj.
//
// Complexity:
//   - Time O(V + A) for one search, Space O(V).
//
// Notes:
//   - The parent map is meaningful only when found is true.
//   - This helper returns one witness path, not all possible paths.
//
// AI-Hints:
//   - Do not replace visited with bottleneck == 0 checks.
//   - Do not use recursion unless depth limits are explicitly handled.
func findDepthFirstAugmentingPath(
	ctx context.Context,
	rn *residualNetwork,
	source, sink string,
	epsilon float64,
) (map[string]string, float64, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, false, err
	}

	type stackEntry struct {
		vertexID   string
		bottleneck float64
	}

	parent := make(map[string]string, len(rn.vertices))
	visited := make(map[string]bool, len(rn.vertices))

	stack := make([]stackEntry, 0, len(rn.vertices))
	stack = append(stack, stackEntry{
		vertexID:   source,
		bottleneck: math.Inf(1),
	})
	visited[source] = true

	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, 0, false, err
		}

		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for _, to := range rn.adj[entry.vertexID] {
			capacity := rn.cap[entry.vertexID][to]
			if capacity <= epsilon || visited[to] {
				continue
			}

			parent[to] = entry.vertexID
			visited[to] = true

			nextBottleneck := math.Min(entry.bottleneck, capacity)
			if to == sink {
				return parent, nextBottleneck, true, nil
			}

			stack = append(stack, stackEntry{
				vertexID:   to,
				bottleneck: nextBottleneck,
			})
		}
	}

	return parent, 0, false, nil
}
