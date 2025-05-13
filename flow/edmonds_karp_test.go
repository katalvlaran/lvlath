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

// EdmondsKarpSuite groups tests for Edmonds–Karp.
type EdmondsKarpSuite struct {
	suite.Suite
	ctx context.Context
}

func (s *EdmondsKarpSuite) SetupTest() {
	s.ctx = context.Background()
}

// TestSimplePath: A→B (cap=5) => maxFlow = 5.
func (s *EdmondsKarpSuite) TestSimplePath() {
	g := core.NewGraph(true, true)
	g.AddEdge("A", "B", 5)

	mf, res, err := flow.EdmondsKarp(s.ctx, g, "A", "B", nil)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 5.0, mf, "max flow should match single-edge capacity")
	require.False(s.T(), res.HasEdge("A", "B"), "forward exhausted")
	require.True(s.T(), res.HasEdge("B", "A"), "reverse edge carries flow")
}

// TestMultiPath: two disjoint routes => flow sums them.
func (s *EdmondsKarpSuite) TestMultiPath() {
	g := core.NewGraph(true, true)
	g.AddEdge("A", "B", 3)
	g.AddEdge("A", "C", 4)
	g.AddEdge("C", "B", 2)

	mf, _, err := flow.EdmondsKarp(s.ctx, g, "A", "B", nil)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 5.0, mf, "flow should combine both paths (3 + 2)")
}

// TestNegativeCapacity yields EdgeError.
func (s *EdmondsKarpSuite) TestNegativeCapacity() {
	g := core.NewGraph(true, true)
	g.AddEdge("X", "Y", -1)

	_, _, err := flow.EdmondsKarp(s.ctx, g, "X", "Y", nil)
	var ee flow.EdgeError
	require.Error(s.T(), err)
	require.True(s.T(), errors.As(err, &ee), "error must be EdgeError")
	require.Equal(s.T(), "X", ee.From)
	require.Equal(s.T(), "Y", ee.To)
	require.Equal(s.T(), -1.0, ee.Cap)
}

// TestSourceSinkNotFound covers missing source or sink.
func (s *EdmondsKarpSuite) TestSourceSinkNotFound() {
	g := core.NewGraph(true, true)
	g.AddVertex(&core.Vertex{ID: "A"})

	_, _, err1 := flow.EdmondsKarp(s.ctx, g, "X", "A", nil)
	require.True(s.T(), errors.Is(err1, flow.ErrSourceNotFound))

	_, _, err2 := flow.EdmondsKarp(s.ctx, g, "A", "Z", nil)
	require.True(s.T(), errors.Is(err2, flow.ErrSinkNotFound))
}

func TestEdmondsKarpSuite(t *testing.T) {
	suite.Run(t, new(EdmondsKarpSuite))
}
