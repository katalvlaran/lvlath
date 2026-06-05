// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package flow implements Edmonds-Karp over deterministic residual networks.
//
// Edmonds-Karp chooses shortest augmenting paths by BFS. It is often slower than
// Dinic on large graphs, but easier to reason about and excellent for certificates,
// debugging, and regression tests with path witnesses.
package flow

import (
	"context"
	"math"
)

// runEdmondsKarp computes maximum flow using shortest augmenting paths.
// Each iteration performs BFS over positive residual arcs and augments one path.
//
// Implementation:
//   - Stage 1: Check cancellation before each BFS.
//   - Stage 2: Find a shortest augmenting path with findShortestAugmentingPath.
//   - Stage 3: Stop when no path or no positive bottleneck remains.
//   - Stage 4: Update residual capacities along the discovered parent chain.
//   - Stage 5: Notify observer and finalize the result when saturated.
//
// Behavior highlights:
//   - BFS gives a shortest path in number of residual arcs.
//   - Path witnesses are materialized deterministically for observers.
//   - Residual updates are clamped through addResidual.
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
//   - BFS scans rn.adj[from] in sorted order.
//   - The first shortest path under that order is selected.
//
// Complexity:
//   - Time O(V * A^2) worst-case for Edmonds-Karp style augmentation.
//   - Space O(V + A) including residual state and BFS maps.
//
// Notes:
//   - Edmonds-Karp is preferable when deterministic path witnesses matter more than speed.
//
// AI-Hints:
//   - Do not replace BFS with DFS here; that changes the algorithm to Ford-Fulkerson.
//   - Do not iterate over rn.cap maps; use rn.adj for stable traversal.
func runEdmondsKarp(
	source, sink string,
	rn *residualNetwork,
	cfg options,
) (*MaxFlowResult, error) {
	maxFlow := 0.0
	augmentations := 0

	for {
		if err := cfg.ctx.Err(); err != nil {
			return newPartialResult(source, sink, AlgorithmEdmondsKarp, maxFlow, augmentations), err
		}

		parent, delta, found, err := findShortestAugmentingPath(
			cfg.ctx,
			rn,
			source,
			sink,
			cfg.epsilon,
		)
		if err != nil {
			return newPartialResult(source, sink, AlgorithmEdmondsKarp, maxFlow, augmentations), err
		}
		if !found || delta <= cfg.epsilon {
			break
		}

		for vertexID := sink; vertexID != source; vertexID = parent[vertexID] {
			previousID := parent[vertexID]
			addResidual(rn, previousID, vertexID, delta, cfg.epsilon)
		}

		maxFlow += delta
		augmentations++

		if err = notifyAugmentation(cfg.ctx, cfg, AugmentationEvent{
			Algorithm: AlgorithmEdmondsKarp,
			Path:      reconstructPath(parent, source, sink),
			Delta:     delta,
			Total:     maxFlow,
			Index:     augmentations,
		}); err != nil {
			return newPartialResult(source, sink, AlgorithmEdmondsKarp, maxFlow, augmentations), err
		}
	}

	return finalizeResult(
		source,
		sink,
		rn,
		cfg,
		AlgorithmEdmondsKarp,
		maxFlow,
		augmentations,
		false,
	)
}

// findShortestAugmentingPath finds one shortest source-sink residual path by BFS.
// It returns the parent chain and bottleneck capacity for that path.
//
// Implementation:
//   - Stage 1: Initialize parent, bottleneck, visited, and head-index queue.
//   - Stage 2: Expand vertices in FIFO order.
//   - Stage 3: Accept only residual arcs with capacity greater than epsilon.
//   - Stage 4: Stop immediately when sink is first discovered.
//
// Behavior highlights:
//   - The first discovered sink path is shortest by edge count.
//   - Parent links describe a single witness path, not all shortest paths.
//   - Bottleneck is propagated as min(previous bottleneck, arc capacity).
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
//   - FIFO order plus rn.adj lexical order selects a stable witness path.
//
// Complexity:
//   - Time O(V + A), Space O(V), where A is residual adjacency-entry count.
//
// Notes:
//   - The returned parent map is meaningful only when found is true.
//
// AI-Hints:
//   - Do not use bottle[v] == 0 as the visited test; zero bottleneck can collide
//     with epsilon policy. Use the explicit visited map.
func findShortestAugmentingPath(
	ctx context.Context,
	rn *residualNetwork,
	source, sink string,
	epsilon float64,
) (map[string]string, float64, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, false, err
	}

	parent := make(map[string]string, len(rn.vertices))
	bottleneck := make(map[string]float64, len(rn.vertices))
	visited := make(map[string]bool, len(rn.vertices))

	queue := make([]string, 0, len(rn.vertices))
	queue = append(queue, source)
	visited[source] = true
	bottleneck[source] = math.Inf(1)

	for head := 0; head < len(queue); head++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, false, err
		}

		from := queue[head]
		for _, to := range rn.adj[from] {
			capacity := rn.cap[from][to]
			if capacity <= epsilon || visited[to] {
				continue
			}

			parent[to] = from
			visited[to] = true
			bottleneck[to] = math.Min(bottleneck[from], capacity)

			if to == sink {
				return parent, bottleneck[to], true, nil
			}

			queue = append(queue, to)
		}
	}

	return parent, 0, false, nil
}
