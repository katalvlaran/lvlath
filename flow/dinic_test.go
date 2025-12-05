package flow_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

// DinicSuite exercises the Dinic implementation under various scenarios.
type DinicSuite struct {
	suite.Suite
}

// TestSingleEdge verifies that a single edge yields max flow equal to its capacity.
func (s *DinicSuite) TestSingleEdge() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_, _ = g.AddEdge("A", "B", 7)

	opts := flow.DefaultOptions()
	mf, res, err := flow.Dinic(g, "A", "B", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 7.0, mf)
	require.False(s.T(), res.HasEdge("A", "B"), "forward edge should be saturated")
	require.True(s.T(), res.HasEdge("B", "A"), "reverse edge should carry the flow")
}

// TestMultiPath verifies max flow on two disjoint paths.
func (s *DinicSuite) TestMultiPath() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	// Path1: A→B (5)
	_, _ = g.AddEdge("A", "B", 5)
	// Path2: A→C (4) → C→B (3)
	_, _ = g.AddEdge("A", "C", 4)
	_, _ = g.AddEdge("C", "B", 3)

	opts := flow.DefaultOptions()
	mf, _, err := flow.Dinic(g, "A", "B", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 8.0, mf) // 5 + 3
}

// TestMultiEdgeAggregation checks that parallel edges are summed.
func (s *DinicSuite) TestMultiEdgeAggregation() {
	g := core.NewGraph(
		core.WithDirected(true),
		core.WithWeighted(),
		core.WithMultiEdges(),
	)
	_, _ = g.AddEdge("A", "B", 2)
	_, _ = g.AddEdge("A", "B", 5)

	opts := flow.DefaultOptions()
	mf, _, err := flow.Dinic(g, "A", "B", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 7.0, mf) // 2 + 5
}

// TestZeroCapacity ensures that zero-capacity edges yield zero flow.
func (s *DinicSuite) TestZeroCapacity() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_, _ = g.AddEdge("X", "Y", 0)

	opts := flow.DefaultOptions()
	mf, _, err := flow.Dinic(g, "X", "Y", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 0.0, mf)
}

// TestEpsilonEdgeCase verifies that capacities ≤ Epsilon are ignored.
func (s *DinicSuite) TestEpsilonEdgeCase() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_, _ = g.AddEdge("U", "V", 1)

	opts := flow.DefaultOptions()
	opts.Epsilon = 2 // filter out capacity=1
	mf, _, err := flow.Dinic(g, "U", "V", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 0.0, mf)
}

// TestLevelRebuildIntervalMoreThanOne ensures that setting LevelRebuildInterval>1
// does not change the result compared to default (never rebuild).
func (s *DinicSuite) TestLevelRebuildIntervalMoreThanOne() {
	// Graph requiring multiple augmentations:
	// S→A(2), S→B(1), A→C(1), B→C(1), C→T(2)
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_, _ = g.AddEdge("S", "A", 2)
	_, _ = g.AddEdge("S", "B", 1)
	_, _ = g.AddEdge("A", "C", 1)
	_, _ = g.AddEdge("B", "C", 1)
	_, _ = g.AddEdge("C", "T", 2)

	opts1 := flow.DefaultOptions()
	opts1.LevelRebuildInterval = 2
	mf1, _, err1 := flow.Dinic(g, "S", "T", opts1)
	require.NoError(s.T(), err1)

	opts2 := flow.DefaultOptions() // default no rebuild
	mf2, _, err2 := flow.Dinic(g, "S", "T", opts2)
	require.NoError(s.T(), err2)

	require.Equal(s.T(), mf1, mf2)
}

// TestContextCancellationDuringBFS ensures cancellation aborts during BFS.
func (s *DinicSuite) TestContextCancellationDuringBFS() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	prev := "V0"
	_ = g.AddVertex(prev)
	const N = 10000
	for i := 1; i < N; i++ {
		cur := fmt.Sprintf("V%d", i)
		_ = g.AddVertex(cur)
		_, _ = g.AddEdge(prev, cur, 1)
		prev = cur
	}
	source, sink := "V0", fmt.Sprintf("V%d", N-1)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // ensure timeout

	opts := flow.DefaultOptions()
	opts.Ctx = ctx

	_, _, err := flow.Dinic(g, source, sink, opts)
	require.Error(s.T(), err)
	require.True(s.T(), errors.Is(err, context.DeadlineExceeded))
}

// TestContextCancellationDuringDFS ensures cancellation aborts during DFS pushes.
func (s *DinicSuite) TestContextCancellationDuringDFS() {
	// Build a "wide" bipartite graph S→{A1..A1000}→{B1..B1000}→T
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_ = g.AddVertex("S")
	_ = g.AddVertex("T")
	for i := 1; i <= 1000; i++ {
		ai := fmt.Sprintf("A%d", i)
		bi := fmt.Sprintf("B%d", i)
		_ = g.AddVertex(ai)
		_ = g.AddVertex(bi)
		_, _ = g.AddEdge("S", ai, 1)
		_, _ = g.AddEdge(ai, bi, 1)
		_, _ = g.AddEdge(bi, "T", 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)

	opts := flow.DefaultOptions()
	opts.Ctx = ctx

	_, _, err := flow.Dinic(g, "S", "T", opts)
	require.Error(s.T(), err)
	require.True(s.T(), errors.Is(err, context.DeadlineExceeded))
}

// TestResidualIntegrity validates the residual invariant on a small graph.
func (s *DinicSuite) TestResidualIntegrity() {
	g := core.NewGraph(
		core.WithDirected(true),
		core.WithWeighted(),
		core.WithMultiEdges(),
	)
	// A→B (5+3=8), B→C (4), C→D (2), A→D (1)
	_, _ = g.AddEdge("A", "B", 5)
	_, _ = g.AddEdge("A", "B", 3)
	_, _ = g.AddEdge("B", "C", 4)
	_, _ = g.AddEdge("C", "D", 2)
	_, _ = g.AddEdge("A", "D", 1)

	opts := flow.DefaultOptions()
	mf, res, err := flow.Dinic(g, "A", "D", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 3.0, mf) // 1 direct + 2 via A→B→C→D

	assertResidualIntegrity(s.T(), g, res)
}

// TestSourceSinkNotFound covers missing source or sink error cases.
func (s *DinicSuite) TestSourceSinkNotFound() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_ = g.AddVertex("A")

	opts := flow.DefaultOptions()
	_, _, err1 := flow.Dinic(g, "X", "A", opts)
	require.True(s.T(), errors.Is(err1, flow.ErrSourceNotFound))

	_, _, err2 := flow.Dinic(g, "A", "Z", opts)
	require.True(s.T(), errors.Is(err2, flow.ErrSinkNotFound))
}

// Entry point for running the suite.
func TestDinicSuite(t *testing.T) {
	suite.Run(t, new(DinicSuite))
}
