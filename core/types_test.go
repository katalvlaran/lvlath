package core_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/katalvlaran/lvlath/core"
)

// TypesSuite exercises basic Graph configuration, vertex management,
// edge ID generation, adjacency, and cloning in the new nested-map design.
type TypesSuite struct {
	suite.Suite
	g *core.Graph
}

// SetupTest creates a weighted, multi-edge, loop-enabled graph before each test.
func (s *TypesSuite) SetupTest() {
	s.g = core.NewGraph(
		core.WithWeighted(),   // allow non-zero weight
		core.WithMultiEdges(), // allow parallel edges
		core.WithLoops(),      // allow self-loops
	)
}

// TestOptions ensures each GraphOption toggles the correct flag.
func (s *TypesSuite) TestOptions() {
	req := require.New(s.T())

	// Default directed=false (undirected)
	req.False(s.g.Directed(), "Graph should default to undirected")
	// Weighted was enabled in SetupTest
	req.True(s.g.Weighted(), "WithWeighted should enable weights")
	// Empty ID should be considered absent, because we reject empty IDs
	req.False(s.g.HasVertex(""), "HasVertex(\"\") should return false for empty ID")

	// Directed override verifies WithDirected works
	dg := core.NewGraph(core.WithDirected(true))
	req.True(dg.Directed(), "WithDirected(true) sets directed flag")

	// Without multi-edges, adding identical endpoints twice must error
	sg := core.NewGraph() // no special options
	_, err := sg.AddEdge("X", "Y", 0)
	req.NoError(err, "first AddEdge should succeed")
	_, err = sg.AddEdge("X", "Y", 0)
	req.ErrorIs(err, core.ErrMultiEdgeNotAllowed, "second AddEdge must ERR_MULTI_EDGE")
}

// TestVertexLifecycle covers AddVertex, HasVertex, RemoveVertex invariants.
func (s *TypesSuite) TestVertexLifecycle() {
	req := require.New(s.T())

	// empty ID
	err := s.g.AddVertex("")
	req.ErrorIs(err, core.ErrEmptyVertexID)

	// valid add
	req.NoError(s.g.AddVertex("V1"))
	req.True(s.g.HasVertex("V1"), "V1 should exist after AddVertex")

	// idempotent add
	count := len(s.g.Vertices())
	req.NoError(s.g.AddVertex("V1"))
	req.Equal(count, len(s.g.Vertices()), "adding existing vertex is no-op")

	// remove nonexistent
	err = s.g.RemoveVertex("Z")
	req.ErrorIs(err, core.ErrVertexNotFound)

	// remove empty
	err = s.g.RemoveVertex("")
	req.ErrorIs(err, core.ErrEmptyVertexID)

	// remove existing
	req.NoError(s.g.RemoveVertex("V1"))
	req.False(s.g.HasVertex("V1"), "V1 should be removed")
}

// TestAtomicEdgeIDs verifies that each AddEdge yields a unique, sequential ID.
func (s *TypesSuite) TestAtomicEdgeIDs() {
	req := require.New(s.T())

	const N = 100
	idCh := make(chan string, N) // buffered so goroutines never block
	var wg sync.WaitGroup
	wg.Add(N)

	// Fire off N concurrent AddEdge calls
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			eid, err := s.g.AddEdge("A", "B", int64(i))
			req.NoError(err, "AddEdge must not error")
			idCh <- eid // send ID into channel
		}(i)
	}

	// wait for all to finish, then close channel
	go func() {
		wg.Wait()
		close(idCh)
	}()

	// Collect into a set to test uniqueness and count
	ids := make(map[string]struct{}, N)
	for eid := range idCh {
		ids[eid] = struct{}{}
	}

	req.Len(ids, N, "should have N unique edge IDs")
}

// TestAdjacencyMap ensures HasEdge is O(1) nested-map lookup.
func (s *TypesSuite) TestAdjacencyMap() {
	req := require.New(s.T())

	// no edge initially
	req.False(s.g.HasEdge("P", "Q"))

	// add one
	eid, err := s.g.AddEdge("P", "Q", 0)
	req.NoError(err)
	req.True(s.g.HasEdge("P", "Q"), "HasEdge should find newly added edge")
	// remove and re-check
	req.NoError(s.g.RemoveEdge(eid))
	req.False(s.g.HasEdge("P", "Q"))
}

// TestCloneMethods covers CloneEmpty and Clone deep-vs-shallow behaviors.
func (s *TypesSuite) TestCloneMethods() {
	req := require.New(s.T())

	// prepare a small graph
	_, _ = s.g.AddEdge("X", "Y", 1)
	_, _ = s.g.AddEdge("Y", "Y", 2) // self-loop

	// CloneEmpty: vertices copied, edges none
	ce := s.g.CloneEmpty()
	req.ElementsMatch(s.g.Vertices(), ce.Vertices(), "CloneEmpty preserves vertices")
	req.Empty(ce.Edges(), "CloneEmpty should have no edges")

	// Clone: full copy
	c := s.g.Clone()
	req.ElementsMatch(s.g.Vertices(), c.Vertices(), "Clone copies vertices")
	req.ElementsMatch(extractIDs(s.g.Edges()), extractIDs(c.Edges()), "Clone copies edges")

	// independence: mutating original edge weight does not affect clone
	orig := s.g.Edges()[0]
	cl := c.Edges()[0]
	orig.Weight = 999
	req.NotEqual(orig.Weight, cl.Weight, "edge weights should not alias")
}

// TestVerticesMapReadOnly ensures VerticesMap returns a safe copy.
func (s *TypesSuite) TestVerticesMapReadOnly() {
	req := require.New(s.T())
	req.NoError(s.g.AddVertex("Z"))
	vm := s.g.VerticesMap()
	vm["NEW"] = &core.Vertex{ID: "NEW"}
	req.False(s.g.HasVertex("NEW"), "mutating VerticesMap must not affect Graph")
}

// TestHasVertexConcurrency ensures that concurrent HasVertex and AddVertex are safe.
func (s *TypesSuite) TestHasVertexConcurrency() {
	var wg sync.WaitGroup
	const M = 50
	wg.Add(2 * M)
	for i := 0; i < M; i++ {
		go func(i int) {
			defer wg.Done()
			_ = s.g.AddVertex(fmt.Sprintf("V%d", i))
		}(i)
		go func(i int) {
			defer wg.Done()
			_ = s.g.HasVertex(fmt.Sprintf("V%d", i))
		}(i)
	}
	wg.Wait()
}

// helper to extract IDs from []*core.Edge
func extractIDs(edges []*core.Edge) []string {
	out := make([]string, len(edges))
	for i, e := range edges {
		out[i] = e.ID
	}

	return out
}

// Run the suite
func TestTypesSuite(t *testing.T) {
	suite.Run(t, new(TypesSuite))
}
