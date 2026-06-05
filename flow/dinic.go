// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

// Package flow implements Dinic's maximum-flow kernel over residualNetwork.
//
// Dinic uses BFS level graphs and DFS blocking-flow pushes. The public Dinic
// function is a legacy wrapper; canonical callers should use MaxFlow with
// WithAlgorithm(AlgorithmDinic).
package flow

import (
	"context"
	"math"
)

// runDinic computes maximum flow with Dinic's level-graph/blocking-flow method.
// It operates only on residualNetwork so graph adaptation stays centralized.
//
// Implementation:
//   - Stage 1: Repeatedly build a BFS level graph from the current residual network.
//   - Stage 2: Stop when sink is not reachable in the level graph.
//   - Stage 3: Push blocking flow by DFS over admissible level-increasing arcs.
//   - Stage 4: Clamp residual updates through addResidual to avoid epsilon noise.
//   - Stage 5: Finalize MaxFlowResult with residual graph and min-cut certificate.
//
// Behavior highlights:
//   - Uses sorted rn.adj lists for deterministic BFS and DFS traversal.
//   - Uses context checks before BFS and before each blocking-flow push.
//   - LevelRebuildInterval can force a fresh level graph after N augmentations.
//
// Inputs:
//   - source: source vertex ID already validated by MaxFlow.
//   - sink: sink vertex ID already validated by MaxFlow.
//   - rn: deterministic residual network built from core.Edges().
//   - cfg: finalized runtime options.
//
// Returns:
//   - *MaxFlowResult: canonical result with flow value, residual graph, and cut.
//   - error: nil on success or interruption/error.
//
// Errors:
//   - context cancellation errors from cfg.ctx.
//   - ErrObserverFailure when observer rejects an augmentation event.
//   - core graph construction errors from finalizeResult/buildResidualGraph.
//
// Determinism:
//   - Level BFS scans rn.vertices/rn.adj order.
//   - DFS scans rn.adj[from] order and advances iter[from] monotonically.
//   - Equal admissible choices are resolved by lexical residual adjacency order.
//
// Complexity:
//   - General Dinic worst-case O(V^2 * A) with A residual adjacency entries.
//   - Common unit-capacity bounds can be stronger, but this implementation does not
//     specialize unit networks.
//   - Space O(V + A) for level, iterator maps, and residual state.
//
// Notes:
//   - Dinic observer events report Delta and Total. Path may be nil because this
//     blocking-flow DFS does not materialize a complete path witness.
//   - Ford-Fulkerson and Edmonds-Karp should be used when per-augmentation path
//     witnesses are required for diagnostics.
//
// AI-Hints:
//   - Do not iterate rn.cap maps directly; rn.adj is the deterministic edge order.
//   - Do not treat pushed == 0 as cancellation; dfsDinicPush returns errors explicitly.
func runDinic(
	source, sink string,
	rn *residualNetwork,
	cfg options,
) (*MaxFlowResult, error) {
	maxFlow := 0.0
	augmentations := 0
	var err error
	var pushed float64

	for {
		if err = cfg.ctx.Err(); err != nil {
			return newPartialResult(source, sink, AlgorithmDinic, maxFlow, augmentations), err
		}

		level, reachable, err := buildDinicLevels(cfg.ctx, rn, source, sink, cfg.epsilon)
		if err != nil {
			return newPartialResult(source, sink, AlgorithmDinic, maxFlow, augmentations), err
		}
		if !reachable {
			break
		}

		iter := make(map[string]int, len(rn.vertices))

		for {
			if err = cfg.ctx.Err(); err != nil {
				return newPartialResult(source, sink, AlgorithmDinic, maxFlow, augmentations), err
			}

			pushed, err = dfsDinicPush(
				cfg.ctx,
				rn,
				level,
				iter,
				source,
				sink,
				math.Inf(1),
				cfg.epsilon,
			)
			if err != nil {
				return newPartialResult(source, sink, AlgorithmDinic, maxFlow, augmentations), err
			}
			if pushed <= cfg.epsilon {
				break
			}

			maxFlow += pushed
			augmentations++

			if err = notifyAugmentation(cfg.ctx, cfg, AugmentationEvent{
				Algorithm: AlgorithmDinic,
				Path:      nil,
				Delta:     pushed,
				Total:     maxFlow,
				Index:     augmentations,
			}); err != nil {
				return newPartialResult(source, sink, AlgorithmDinic, maxFlow, augmentations), err
			}

			if cfg.levelRebuildInterval > 0 &&
				augmentations%cfg.levelRebuildInterval == 0 {
				break
			}
		}
	}

	return finalizeResult(
		source,
		sink,
		rn,
		cfg,
		AlgorithmDinic,
		maxFlow,
		augmentations,
		false,
	)
}

// buildDinicLevels constructs one BFS level graph for Dinic's algorithm.
// Vertices unreachable from source keep level -1.
//
// Implementation:
//   - Stage 1: Initialize every known vertex level to -1.
//   - Stage 2: Seed source with level 0.
//   - Stage 3: BFS through residual arcs with capacity greater than epsilon.
//   - Stage 4: Report whether sink became reachable.
//
// Behavior highlights:
//   - Uses a head-index queue to avoid frontier retention leaks.
//   - Scans rn.adj[from] in deterministic order.
//   - Does not mutate residual capacities.
//
// Inputs:
//   - ctx: cancellation context.
//   - rn: deterministic residual network.
//   - source: source vertex ID.
//   - sink: sink vertex ID.
//   - epsilon: residual threshold for arc eligibility.
//
// Returns:
//   - map[string]int: level per vertex, with -1 for unreachable vertices.
//   - bool: true if sink is reachable in the level graph.
//   - error: context error when cancellation is observed.
//
// Errors:
//   - ctx.Err() when canceled before or during BFS.
//
// Determinism:
//   - Queue expansion order follows rn.adj order and head-index FIFO discipline.
//
// Complexity:
//   - Time O(V + A), Space O(V), where A is residual adjacency-entry count.
//
// Notes:
//   - This helper builds a fresh level graph per Dinic phase.
//   - It treats capacities <= epsilon as absent arcs.
//
// AI-Hints:
//   - Do not use queue = queue[1:] here.
//   - Do not scan rn.cap maps; that reintroduces nondeterministic traversal.
func buildDinicLevels(
	ctx context.Context,
	rn *residualNetwork,
	source, sink string,
	epsilon float64,
) (map[string]int, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	level := make(map[string]int, len(rn.vertices))
	for _, vertexID := range rn.vertices {
		level[vertexID] = -1
	}

	queue := make([]string, 0, len(rn.vertices))
	queue = append(queue, source)
	level[source] = 0

	for head := 0; head < len(queue); head++ {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}

		from := queue[head]
		for _, to := range rn.adj[from] {
			if rn.cap[from][to] <= epsilon || level[to] >= 0 {
				continue
			}

			level[to] = level[from] + 1
			queue = append(queue, to)
		}
	}

	return level, level[sink] >= 0, nil
}

// dfsDinicPush pushes one admissible blocking-flow fragment through the level graph.
// It mutates residual capacities only after a downstream push succeeds.
//
// Implementation:
//   - Stage 1: Respect cancellation at function entry.
//   - Stage 2: Return available flow when sink is reached.
//   - Stage 3: Continue scanning from iter[from] to avoid rescanning exhausted arcs.
//   - Stage 4: Use only admissible arcs where level[to] == level[from]+1.
//   - Stage 5: Update residual capacities with epsilon clamping after success.
//
// Behavior highlights:
//   - Recursive depth is at most the length of an admissible source-sink path.
//   - iter[from] is monotonic within one level graph phase.
//   - Residual mutation is centralized through addResidual.
//
// Inputs:
//   - ctx: cancellation context.
//   - rn: mutable residual network.
//   - level: BFS level graph.
//   - iter: per-vertex next-adjacency cursor.
//   - from: current vertex.
//   - sink: sink vertex.
//   - available: bottleneck available before entering from.
//   - epsilon: residual threshold.
//
// Returns:
//   - float64: pushed flow amount, or 0 when no admissible push remains.
//   - error: context cancellation error.
//
// Errors:
//   - ctx.Err() when cancellation is observed.
//
// Determinism:
//   - Candidate arcs are tested in rn.adj[from] order.
//   - Equal-capacity choices follow stable adjacency order.
//
// Complexity:
//   - Across one blocking-flow phase, total scanning is O(A) due to iter cursors.
//   - Recursion stack space O(V) in the worst case.
//
// Notes:
//   - This helper does not construct path witnesses for observer events.
//   - For witness-heavy diagnostics, Edmonds-Karp is easier to inspect.
//
// AI-Hints:
//   - Do not reset iter[from] inside recursion.
//   - Do not subtract residual capacity before the recursive push succeeds.
func dfsDinicPush(
	ctx context.Context,
	rn *residualNetwork,
	level map[string]int,
	iter map[string]int,
	from, sink string,
	available float64,
	epsilon float64,
) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if from == sink {
		return available, nil
	}

	for iter[from] < len(rn.adj[from]) {
		to := rn.adj[from][iter[from]]
		iter[from]++

		capacity := rn.cap[from][to]
		if capacity <= epsilon || level[to] != level[from]+1 {
			continue
		}

		pushed, err := dfsDinicPush(
			ctx,
			rn,
			level,
			iter,
			to,
			sink,
			math.Min(available, capacity),
			epsilon,
		)
		if err != nil {
			return 0, err
		}
		if pushed <= epsilon {
			continue
		}

		addResidual(rn, from, to, pushed, epsilon)
		return pushed, nil
	}

	return 0, nil
}
