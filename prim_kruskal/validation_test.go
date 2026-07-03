// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package prim_kruskal_test

import (
	"errors"
	"math"
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/prim_kruskal"
)

func TestValidation_EmptyGraphReturnsDisconnectedAndEmptyGraph(t *testing.T) {
	graph := mustWeightedGraph(t)

	_, err := prim_kruskal.Kruskal(graph)

	mustErrorIs(t, err, prim_kruskal.ErrDisconnected, "Kruskal empty graph disconnected")
	mustErrorIs(t, err, prim_kruskal.ErrEmptyGraph, "Kruskal empty graph precise")
}

func TestValidation_NilGraphReturnsPreciseSentinels(t *testing.T) {
	_, err := prim_kruskal.MinimumSpanningTree(nil)

	mustErrorIs(t, err, prim_kruskal.ErrInvalidGraph, "nil graph umbrella")
	mustErrorIs(t, err, prim_kruskal.ErrNilGraph, "nil graph precise")
}

func TestValidation_UnweightedGraphRejected(t *testing.T) {
	graph, err := core.NewGraph()
	mustNoError(t, err, "core.NewGraph")

	_, err = prim_kruskal.Kruskal(graph)

	mustErrorIs(t, err, prim_kruskal.ErrInvalidGraph, "unweighted graph umbrella")
	mustErrorIs(t, err, prim_kruskal.ErrUnweightedGraph, "unweighted graph precise")
}

func TestValidation_DirectedGraphRejected(t *testing.T) {
	graph, err := core.NewGraph(core.WithWeighted(), core.WithDirected(true))
	mustNoError(t, err, "core.NewGraph directed weighted")

	_, err = prim_kruskal.Kruskal(graph)

	mustErrorIs(t, err, prim_kruskal.ErrInvalidGraph, "directed graph umbrella")
	mustErrorIs(t, err, prim_kruskal.ErrDirectedGraph, "directed graph precise")
}

func TestValidation_DirectedEdgeRejected(t *testing.T) {
	graph, err := core.NewGraph(core.WithWeighted(), core.WithMixedEdges())
	mustNoError(t, err, "core.NewGraph mixed weighted")

	_, err = graph.AddEdge("A", "B", 1, core.WithEdgeDirected(true))
	mustNoError(t, err, "Graph.AddEdge directed override")

	_, err = prim_kruskal.Prim(graph, "A")

	mustErrorIs(t, err, prim_kruskal.ErrInvalidGraph, "directed edge umbrella")
	mustErrorIs(t, err, prim_kruskal.ErrDirectedEdge, "directed edge precise")
}

func TestValidation_EmptyPrimRootRejectedBeforeTraversal(t *testing.T) {
	graph := mustWeightedGraph(t)
	_ = graph.AddVertex("X")

	_, err := prim_kruskal.Prim(graph, "")

	mustErrorIs(t, err, prim_kruskal.ErrEmptyRoot, "empty Prim root")
}

func TestValidation_NonFiniteWeightRejectedAfterAliasMutation(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
	}{
		{name: "NaN", weight: math.NaN()},
		{name: "PositiveInf", weight: math.Inf(1)},
		{name: "NegativeInf", weight: math.Inf(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := mustWeightedGraph(t)
			edgeID, _ := graph.AddEdge("A", "B", 1)
			mustCorruptEdgeWeight(t, graph, edgeID, tt.weight)

			_, err := prim_kruskal.Kruskal(graph)
			mustErrorIs(t, err, prim_kruskal.ErrNaNInfWeight, "Kruskal non-finite weight")

			_, err = prim_kruskal.Prim(graph, "A")
			mustErrorIs(t, err, prim_kruskal.ErrNaNInfWeight, "Prim non-finite weight")
		})
	}
}

func TestValidation_NilOptionRejected(t *testing.T) {
	graph := mustWeightedGraph(t)
	_, _ = graph.AddEdge("A", "B", 1)

	_, err := prim_kruskal.MinimumSpanningTree(graph, nil)

	mustErrorIs(t, err, prim_kruskal.ErrNilOption, "nil option")
}

func TestValidation_UnsupportedAlgorithmRejected(t *testing.T) {
	graph := mustWeightedGraph(t)
	_, _ = graph.AddEdge("A", "B", 1)

	_, err := prim_kruskal.MinimumSpanningTree(
		graph,
		prim_kruskal.WithAlgorithm(prim_kruskal.Algorithm("unknown")),
	)

	mustErrorIs(t, err, prim_kruskal.ErrUnsupportedAlgorithm, "unsupported algorithm")
}

func TestValidation_StrictModeRejectsDisconnectedGraph(t *testing.T) {
	graph := mustWeightedGraph(t)
	_, _ = graph.AddEdge("A", "B", 1)
	_, _ = graph.AddEdge("C", "D", 2)

	_, err := prim_kruskal.MinimumSpanningTree(graph)

	mustErrorIs(t, err, prim_kruskal.ErrDisconnected, "strict disconnected graph")
}

func TestValidation_NoErrorStringMatching(t *testing.T) {
	err := errors.Join(prim_kruskal.ErrInvalidGraph, prim_kruskal.ErrNilGraph)

	mustErrorIs(t, err, prim_kruskal.ErrInvalidGraph, "joined invalid graph")
	mustErrorIs(t, err, prim_kruskal.ErrNilGraph, "joined nil graph")
}
