package flow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

// FordFulkersonSuite exercises the Ford–Fulkerson implementation under various scenarios.
type FordFulkersonSuite struct {
	suite.Suite
}

// TestSimplePath verifies that a single-edge graph yields max flow == that capacity,
// and that the residual graph has no forward edge and a reverse edge of equal weight.
func (s *FordFulkersonSuite) TestSimplePath() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	g.AddEdge("A", "B", 10)

	opts := flow.DefaultOptions()
	mf, res, err := flow.FordFulkerson(g, "A", "B", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(10), mf)

	// after saturation, no forward capacity
	require.False(s.T(), res.HasEdge("A", "B"))
	// reverse capacity carries the entire flow
	require.True(s.T(), res.HasEdge("B", "A"))
}

// TestMultiPathGraph verifies that two disjoint paths combine their capacities.
func (s *FordFulkersonSuite) TestMultiPathGraph() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	// Path1: A→B cap=5
	g.AddEdge("A", "B", 5)
	// Path2: A→C cap=7 → C→B cap=4
	g.AddEdge("A", "C", 7)
	g.AddEdge("C", "B", 4)

	opts := flow.DefaultOptions()
	mf, _, err := flow.FordFulkerson(g, "A", "B", opts)
	require.NoError(s.T(), err)
	// Maximum should be 5 + 4 = 9
	require.Equal(s.T(), int64(9), mf)
}

// TestZeroCapacity ensures that edges with zero capacity produce zero flow.
func (s *FordFulkersonSuite) TestZeroCapacity() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	g.AddEdge("X", "Y", 0)

	opts := flow.DefaultOptions()
	mf, _, err := flow.FordFulkerson(g, "X", "Y", opts)
	require.NoError(s.T(), err)
	require.Equal(s.T(), int64(0), mf)
}

// TestEpsilonEdgeCase verifies that Epsilon filtering treats small capacities as zero.
func (s *FordFulkersonSuite) TestEpsilonEdgeCase() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	g.AddEdge("U", "V", 1)

	opts := flow.DefaultOptions()
	// set Epsilon > 1 so that capacity 1 is ignored
	opts.Epsilon = 2
	mf, _, err := flow.FordFulkerson(g, "U", "V", opts)
	require.NoError(s.T(), err)
	// no path should be found
	require.Equal(s.T(), int64(0), mf)
}

// TestMultiEdgeLoop checks multi-edge aggregation and loop-ignoring behavior.
func (s *FordFulkersonSuite) TestMultiEdgeLoop() {
	// enable parallel edges and loops
	g := core.NewGraph(
		core.WithDirected(true),
		core.WithWeighted(),
		core.WithMultiEdges(),
		core.WithLoops(),
	)
	// parallel edges U→V of 3 and 2 → total 5
	g.AddEdge("U", "V", 3)
	g.AddEdge("U", "V", 2)
	// self-loop on W→W should be ignored entirely
	g.AddEdge("W", "W", 5)

	opts := flow.DefaultOptions()
	mf, _, err := flow.FordFulkerson(g, "U", "V", opts)
	require.NoError(s.T(), err)
	// all parallel capacity summed
	require.Equal(s.T(), int64(5), mf)
}

// TestResidualIntegrity constructs a small graph with multiple edges and
// verifies the residual integrity invariant after running Ford–Fulkerson.
func (s *FordFulkersonSuite) TestResidualIntegrity() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted(), core.WithMultiEdges())
	// Build a graph:
	//   A→B (5, then 3) → total 8
	//   B→C (4)
	//   C→D (2)
	//   A→D (1)
	g.AddEdge("A", "B", 5)
	g.AddEdge("A", "B", 3)
	g.AddEdge("B", "C", 4)
	g.AddEdge("C", "D", 2)
	g.AddEdge("A", "D", 1)

	opts := flow.DefaultOptions()
	mf, res, err := flow.FordFulkerson(g, "A", "D", opts)
	require.NoError(s.T(), err)
	// one direct unit A→D and two via A→B→C→D
	require.Equal(s.T(), int64(3), mf)

	// verify residual integrity across all edges
	assertResidualIntegrity(s.T(), g, res)
}

// TestSourceOrSinkNotFound covers missing source or sink error cases.
func (s *FordFulkersonSuite) TestSourceOrSinkNotFound() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	g.AddVertex("A")

	opts := flow.DefaultOptions()
	// missing source
	_, _, err1 := flow.FordFulkerson(g, "X", "A", opts)
	require.True(s.T(), errors.Is(err1, flow.ErrSourceNotFound))
	// missing sink
	_, _, err2 := flow.FordFulkerson(g, "A", "Z", opts)
	require.True(s.T(), errors.Is(err2, flow.ErrSinkNotFound))
}

// TestContextCancellation verifies that a canceled context aborts quickly.
func (s *FordFulkersonSuite) TestContextCancellation() {
	g := core.NewGraph(core.WithDirected(true), core.WithWeighted())
	// chain graph A→B→C→D
	g.AddEdge("A", "B", 1)
	g.AddEdge("B", "C", 1)
	g.AddEdge("C", "D", 1)

	// timeout almost immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // ensure deadline exceeded

	opts := flow.DefaultOptions()
	opts.Ctx = ctx

	_, _, err := flow.FordFulkerson(g, "A", "D", opts)
	require.Error(s.T(), err)
	require.Equal(s.T(), context.DeadlineExceeded, err)
}

// Entry point for running the suite
func TestFordFulkersonSuite(t *testing.T) {
	suite.Run(t, new(FordFulkersonSuite))
}

//
// Helpers methods
// // // // // // // // // //

// assertResidualIntegrity verifies that for every edge u→v in the original graph,
// the following invariant holds on the residual graph:
//
//	  initialCapacity(u,v) == forwardResidual(u,v) + backwardResidual(v,u)
//
//	- initialCapacity sums up all parallel edges in the original.
//	- forwardResidual is the sum of weights on all result edges u→v.
//	- backwardResidual is the sum of weights on all result edges v→u.
//
// t:       the testing context (from *testing.T).
// original: the input graph before max-flow was run.
// result:   the residual graph returned by the max-flow algorithm.
func assertResidualIntegrity(
	t *testing.T,
	original *core.Graph,
	result *core.Graph,
) {
	// Build a map of initial capacities for each ordered pair (u, v).
	initial := make(map[[2]string]int64)
	for _, e := range original.Edges() {
		// Sum together parallel edges
		initial[[2]string{e.From, e.To}] += e.Weight
	}

	// For each original (u→v), compute forward+backward residuals and assert.
	for uv, initCap := range initial {
		u, v := uv[0], uv[1]

		// Sum forward residual capacity on edges u→v in `result`.
		var forwardRes int64
		if result.HasEdge(u, v) {
			neighbors, err := result.Neighbors(u)
			require.NoError(t, err, "failed to list neighbors of %s", u)
			for _, e := range neighbors {
				if e.To == v {
					forwardRes += e.Weight
				}
			}
		}

		// Sum backward residual capacity on edges v→u in `result`.
		var backwardRes int64
		if result.HasEdge(v, u) {
			neighbors, err := result.Neighbors(v)
			require.NoError(t, err, "failed to list neighbors of %s", v)
			for _, e := range neighbors {
				if e.To == u {
					backwardRes += e.Weight
				}
			}
		}

		// Assert the invariant for this edge.
		require.Equal(t, initCap, forwardRes+backwardRes, "residual invariant failed for edge %s→%s", u, v)
	}
}
