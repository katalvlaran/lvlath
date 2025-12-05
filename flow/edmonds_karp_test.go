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

// EdmondsKarpSuite covers correctness, edge cases, and invariant checks
// for the Edmonds–Karp maximum flow implementation.
type EdmondsKarpSuite struct {
	suite.Suite
}

// TestSingleEdge verifies that a single edge yields maxFlow == capacity
// and that the residual graph has no forward edge and a reverse edge of equal weight.
func (s *EdmondsKarpSuite) TestSingleEdge() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_, _ = g.AddEdge("S", "T", 5)

	opts := flow.DefaultOptions()
	mf, res, err := flow.EdmondsKarp(g, "S", "T", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 5.0, mf)
	require.False(s.T(), res.HasEdge("S", "T"), "forward edge should be saturated")
	require.True(s.T(), res.HasEdge("T", "S"), "reverse edge should carry the flow")
}

// TestMultiPath sums capacities along disjoint routes.
func (s *EdmondsKarpSuite) TestMultiPath() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	// Path1: S→A (3) → A→T (3)
	_, _ = g.AddEdge("S", "A", 3)
	_, _ = g.AddEdge("A", "T", 3)
	// Path2: S→B (4) → B→T (2)
	_, _ = g.AddEdge("S", "B", 4)
	_, _ = g.AddEdge("B", "T", 2)

	opts := flow.DefaultOptions()
	mf, _, err := flow.EdmondsKarp(g, "S", "T", opts)
	require.NoError(s.T(), err)
	// Expected max flow = 3 + 2 = 5
	require.Equal(s.T(), 5.0, mf)
}

// TestMultiEdgeAggregation verifies that parallel edges are summed before flow.
func (s *EdmondsKarpSuite) TestMultiEdgeAggregation() {
	g := core.NewGraph(
		core.WithDirected(true),
		core.WithWeighted(),
		core.WithMultiEdges(),
	)
	// Parallel edges X→Y: 2 and 7 => total capacity 9
	_, _ = g.AddEdge("X", "Y", 2)
	_, _ = g.AddEdge("X", "Y", 7)

	opts := flow.DefaultOptions()
	mf, _, err := flow.EdmondsKarp(g, "X", "Y", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 9.0, mf)
}

// TestZeroCapacity ensures that zero-capacity edges produce zero flow.
func (s *EdmondsKarpSuite) TestZeroCapacity() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_, _ = g.AddEdge("U", "V", 0)

	opts := flow.DefaultOptions()
	mf, _, err := flow.EdmondsKarp(g, "U", "V", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 0.0, mf)
}

// TestEpsilonEdgeCase verifies that edges with weight ≤ Epsilon are ignored.
func (s *EdmondsKarpSuite) TestEpsilonEdgeCase() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_, _ = g.AddEdge("U", "V", 1)

	opts := flow.DefaultOptions()
	opts.Epsilon = 2 // capacities ≤2 are filtered out => capacity 1 ignored
	mf, _, err := flow.EdmondsKarp(g, "U", "V", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 0.0, mf)
}

// TestContextCancellationDuringBFS ensures that a canceled context aborts BFS promptly.
func (s *EdmondsKarpSuite) TestContextCancellationDuringBFS() {
	// Build a long chain to force a longer BFS
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	prev := "V0"
	_ = g.AddVertex(prev)
	const N = 10000
	for i := 1; i < N; i++ {
		cur := "V" + fmt.Sprint(i)
		_ = g.AddVertex(cur)
		_, _ = g.AddEdge(prev, cur, 1)
		prev = cur
	}
	source, sink := "V0", "V9999"

	// Create a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // ensure deadline exceeded

	opts := flow.DefaultOptions()
	opts.Ctx = ctx

	_, _, err := flow.EdmondsKarp(g, source, sink, opts)
	require.Error(s.T(), err)
	require.True(s.T(), errors.Is(err, context.DeadlineExceeded))
}

// TestResidualIntegrity validates that for each original edge u→v,
// initialCap == forwardResCap + backwardResCap after flow completes.
func (s *EdmondsKarpSuite) TestResidualIntegrity() {
	g := core.NewGraph(
		core.WithDirected(true),
		core.WithWeighted(),
		core.WithMultiEdges(),
	)
	// Construct graph:
	//   A→B (5, then 3) =8, B→C=4, C→D=2, A→D=1
	_, _ = g.AddEdge("A", "B", 5)
	_, _ = g.AddEdge("A", "B", 3)
	_, _ = g.AddEdge("B", "C", 4)
	_, _ = g.AddEdge("C", "D", 2)
	_, _ = g.AddEdge("A", "D", 1)

	opts := flow.DefaultOptions()
	mf, res, err := flow.EdmondsKarp(g, "A", "D", opts)
	require.NoError(s.T(), err)
	// Direct A→D =1 plus A→B→C→D =2 => total flow =3
	require.Equal(s.T(), 3.0, mf)

	// Assert residual integrity for every original edge.
	assertResidualIntegrity(s.T(), g, res)
}

// TestSourceSinkNotFound covers missing source or sink error cases.
func (s *EdmondsKarpSuite) TestSourceSinkNotFound() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	_ = g.AddVertex("A")

	opts := flow.DefaultOptions()
	_, _, err1 := flow.EdmondsKarp(g, "X", "A", opts)
	require.True(s.T(), errors.Is(err1, flow.ErrSourceNotFound))

	_, _, err2 := flow.EdmondsKarp(g, "A", "Z", opts)
	require.True(s.T(), errors.Is(err2, flow.ErrSinkNotFound))
}

// Entry point for running the suite.
func TestEdmondsKarpSuite(t *testing.T) {
	suite.Run(t, new(EdmondsKarpSuite))
}
