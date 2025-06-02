package dfs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dfs"
)

// position returns index of v in slice or -1 if not found
func position(order []string, v string) int {
	for i, x := range order {
		if x == v {
			return i
		}
	}

	return -1
}

// TestTopo_NilGraph verifies that passing a nil graph returns ErrGraphNil.
func TestTopo_NilGraph(t *testing.T) {
	order, err := dfs.TopologicalSort(nil)
	assert.Nil(t, order)
	assert.ErrorIs(t, err, dfs.ErrGraphNil)
}

// TestTopo_UndirectedGraph ensures TopologicalSort rejects undirected graphs.
func TestTopo_UndirectedGraph(t *testing.T) {
	g := core.NewGraph() // undirected by default
	_, err := dfs.TopologicalSort(g)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires directed graph")
}

// TestTopo_EmptyGraph covers a directed graph with no vertices.
func TestTopo_EmptyGraph(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// no vertices added
	order, err := dfs.TopologicalSort(g)
	assert.NoError(t, err)
	assert.Empty(t, order)
}

// TestTopo_NoEdges checks that a directed graph with vertices but no edges
// can be sorted in any order.
func TestTopo_NoEdges(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	_ = g.AddVertex("A")
	_ = g.AddVertex("B")
	_ = g.AddVertex("C")

	order, err := dfs.TopologicalSort(g)
	assert.NoError(t, err)
	// any permutation of A,B,C is valid
	assert.ElementsMatch(t, []string{"A", "B", "C"}, order)
}

// TestTopo_SimpleChain verifies linear chain A→B→C yields [A,B,C].
func TestTopo_SimpleChain(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("B", "C", 0)

	order, err := dfs.TopologicalSort(g)
	assert.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, order)
}

// TestTopo_BranchingDAG checks a DAG with A→B and A→C: A must come first,
// B and C in any order afterward.
func TestTopo_BranchingDAG(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("A", "C", 0)

	order, err := dfs.TopologicalSort(g)
	assert.NoError(t, err)
	assert.Equal(t, "A", order[0])
	assert.ElementsMatch(t, []string{"B", "C"}, order[1:])
}

// TestTopo_Disconnected verifies that disconnected components are included:
// each component appears in a valid topological segment.
func TestTopo_Disconnected(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// component 1: X→Y
	_, _ = g.AddEdge("X", "Y", 0)
	// component 2: A→B
	_, _ = g.AddEdge("A", "B", 0)

	order, err := dfs.TopologicalSort(g)
	assert.NoError(t, err)
	// X must precede Y, A must precede B; the two pairs can interleave
	pos := func(v string) int {
		for i, id := range order {
			if id == v {
				return i
			}
		}

		return -1
	}
	assert.Less(t, pos("X"), pos("Y"))
	assert.Less(t, pos("A"), pos("B"))
	assert.Len(t, order, 4)
	assert.ElementsMatch(t, []string{"X", "Y", "A", "B"}, order)
}

// TestTopo_Cycle ensures that a cycle detection returns ErrCycleDetected.
func TestTopo_Cycle(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("B", "C", 0)
	_, _ = g.AddEdge("C", "A", 0)

	order, err := dfs.TopologicalSort(g)
	assert.Nil(t, order)
	assert.ErrorIs(t, err, dfs.ErrCycleDetected)
}

// TestTopo_LargeLinearChain verifies a linear chain of 10 vertices A→B→C→...→J.
func TestTopo_LargeLinearChain(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// Build chain A→B→C→...→J
	vertices := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	for i := 0; i < len(vertices)-1; i++ {
		from, to := vertices[i], vertices[i+1]
		_, err := g.AddEdge(from, to, 0)
		assert.NoError(t, err, "AddEdge(%s→%s)", from, to)
	}
	// Sort
	order, err := dfs.TopologicalSort(g)
	assert.NoError(t, err)
	// Verify all 10 present
	assert.Len(t, order, 10)
	// Each predecessor appears before successor
	for i := 0; i < len(vertices)-1; i++ {
		u, v := vertices[i], vertices[i+1]
		assert.Lessf(t,
			position(order, u), position(order, v),
			"node %s should come before %s", u, v,
		)
	}
}

// TestTopo_DisconnectedLarge ensures two disjoint chains are interleaved correctly.
func TestTopo_DisconnectedLarge(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// Chain1: 1→2→3→4
	chain1 := []string{"1", "2", "3", "4"}
	for i := 0; i < len(chain1)-1; i++ {
		_, err := g.AddEdge(chain1[i], chain1[i+1], 0)
		assert.NoError(t, err)
	}
	// Chain2: A→B→C→D→E
	chain2 := []string{"A", "B", "C", "D", "E"}
	for i := 0; i < len(chain2)-1; i++ {
		_, err := g.AddEdge(chain2[i], chain2[i+1], 0)
		assert.NoError(t, err)
	}
	order, err := dfs.TopologicalSort(g)
	assert.NoError(t, err)
	assert.Len(t, order, len(chain1)+len(chain2))
	// Verify ordering constraints in each chain
	for i := 0; i < len(chain1)-1; i++ {
		u, v := chain1[i], chain1[i+1]
		assert.Less(t,
			position(order, u), position(order, v),
			"%s should precede %s", u, v,
		)
	}
	for i := 0; i < len(chain2)-1; i++ {
		u, v := chain2[i], chain2[i+1]
		assert.Less(t,
			position(order, u), position(order, v),
			"%s should precede %s", u, v,
		)
	}
}

// TestTopo_ComplexDAG builds a DAG of 10 vertices with cross-links and ensures validity.
func TestTopo_ComplexDAG(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// vertices V1...V10
	vs := []string{"V1", "V2", "V3", "V4", "V5", "V6", "V7", "V8", "V9", "V10"}
	// add vertices explicitly
	for _, v := range vs {
		_ = g.AddVertex(v)
	}
	// add edges: V1→V3, V1→V2, V2→V5, V3→V5, V2→V4, V4→V6,
	// V5→V7, V6→V8, V7→V9, V8→V10
	edges := [][2]string{
		{"V1", "V3"}, {"V1", "V2"}, {"V2", "V5"}, {"V3", "V5"},
		{"V2", "V4"}, {"V4", "V6"}, {"V5", "V7"}, {"V6", "V8"},
		{"V7", "V9"}, {"V8", "V10"},
	}
	for _, e := range edges {
		_, err := g.AddEdge(e[0], e[1], 0)
		assert.NoError(t, err)
	}
	order, err := dfs.TopologicalSort(g)
	assert.NoError(t, err)
	assert.Len(t, order, 10)
	// all dependencies must hold
	for _, e := range edges {
		u, v := e[0], e[1]
		assert.Less(t,
			position(order, u), position(order, v),
			"edge %s→%s should be respected", u, v,
		)
	}
}

// TestTopo_CycleDetection uses a 6-node cycle to verify ErrCycleDetected.
func TestTopo_CycleDetection(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// cycle on 6 nodes: a→b→c→d→e→f→a
	cycle := []string{"a", "b", "c", "d", "e", "f"}
	for i := 0; i < len(cycle); i++ {
		from := cycle[i]
		to := cycle[(i+1)%len(cycle)]
		_, err := g.AddEdge(from, to, 0)
		assert.NoError(t, err)
	}
	order, err := dfs.TopologicalSort(g)
	assert.Nil(t, order)
	assert.ErrorIs(t, err, dfs.ErrCycleDetected)
}
