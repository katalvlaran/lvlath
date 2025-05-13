package flow_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/flow"
)

// FordFulkersonSuite runs unit tests for the Ford–Fulkerson implementation.
type FordFulkersonSuite struct {
	suite.Suite
	ctx context.Context
}

// SetupTest prepares a background context before each test.
func (s *FordFulkersonSuite) SetupTest() {
	s.ctx = context.Background()
}

// TestSimplePath verifies max flow on a trivial single-edge graph:
//
//	A → B (capacity 10)
//
// Expected max flow = 10.
func (s *FordFulkersonSuite) TestSimplePath() {
	g := core.NewGraph(true, true)
	g.AddEdge("A", "B", 10)

	flowValue, residual, err := flow.FordFulkerson(s.ctx, g, "A", "B", nil)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 10.0, flowValue, "max flow on single edge should equal its capacity")

	// Residual graph should have zero forward capacity and 10 in reverse
	require.False(s.T(), residual.HasEdge("A", "B"), "forward capacity should be exhausted")
	require.True(s.T(), residual.HasEdge("B", "A"), "reverse edge should carry the flow")
}

// TestMultiPathGraph verifies max flow on a graph with two disjoint paths:
//
//	A → B (cap 5)
//	A → C (cap 7), C → B (cap 4)
//
// Expected max flow = 5 + 4 = 9.
func (s *FordFulkersonSuite) TestMultiPathGraph() {
	g := core.NewGraph(true, true)
	g.AddEdge("A", "B", 5)
	g.AddEdge("A", "C", 7)
	g.AddEdge("C", "B", 4)

	flowValue, _, err := flow.FordFulkerson(s.ctx, g, "A", "B", nil)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 9.0, flowValue, "max flow should combine both disjoint paths")
}

// TestNegativeCapacityError ensures that an edge with negative capacity
// yields an EdgeError describing the offending edge.
func (s *FordFulkersonSuite) TestNegativeCapacityError() {
	g := core.NewGraph(true, true)
	g.AddEdge("X", "Y", -5) // invalid negative capacity

	_, _, err := flow.FordFulkerson(s.ctx, g, "X", "Y", nil)
	var eErr flow.EdgeError
	require.Error(s.T(), err, "negative capacity should produce an error")
	require.True(s.T(), errors.As(err, &eErr), "error should be of type EdgeError")
	require.Equal(s.T(), "X", eErr.From, "EdgeError.From should match")
	require.Equal(s.T(), "Y", eErr.To, "EdgeError.To should match")
	require.Equal(s.T(), -5.0, eErr.Cap, "EdgeError.Cap should carry the negative value")
}

// TestSourceOrSinkNotFound covers missing source or sink vertices.
func (s *FordFulkersonSuite) TestSourceOrSinkNotFound() {
	g := core.NewGraph(true, true)
	g.AddVertex(&core.Vertex{ID: "A"})

	_, _, err1 := flow.FordFulkerson(s.ctx, g, "X", "A", nil)
	require.True(s.T(), errors.Is(err1, flow.ErrSourceNotFound), "missing source should yield ErrSourceNotFound")

	_, _, err2 := flow.FordFulkerson(s.ctx, g, "A", "Y", nil)
	require.True(s.T(), errors.Is(err2, flow.ErrSinkNotFound), "missing sink should yield ErrSinkNotFound")
}

func TestFordFulkersonSuite(t *testing.T) {
	suite.Run(t, new(FordFulkersonSuite))
}
