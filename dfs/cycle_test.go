package dfs_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dfs"
)

// TestDirectedNilGraph verifies DetectCycles handles nil input without error.
func TestDetectCycles_DirectedNilGraph(t *testing.T) {
	// Since our DetectCycles now returns (bool, [][]string, error),
	// we capture the error and assert it's nil.
	has, cycles, err := dfs.DetectCycles(nil)
	assert.NoError(t, err) // no error when graph is nil
	assert.False(t, has)   // should indicate no cycle
	assert.Nil(t, cycles)  // cycles slice should be nil
}

// TestDetectCycles_DirectedNoCycle ensures no cycles in a simple directed chain.
func TestDetectCycles_DirectedNoCycle(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// Build a simple directed acyclic structure:
	// A -> B -> C -> G
	//     |
	//     D -> E -> F
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("B", "C", 0)
	_, _ = g.AddEdge("B", "D", 0)
	_, _ = g.AddEdge("C", "G", 0)
	_, _ = g.AddEdge("D", "E", 0)
	_, _ = g.AddEdge("E", "F", 0)

	has, cycles, err := dfs.DetectCycles(g)
	assert.NoError(t, err)  // neighbor lookups should not fail
	assert.False(t, has)    // no cycle expected
	assert.Empty(t, cycles) // cycles slice should be empty
}

// TestDetectCycles_SimpleTwoNodeCycle covers two-node cycle normalization.
func TestDetectCycles_SimpleTwoNodeCycle(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// A -> B -> A forms a simple directed 2-node cycle
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("B", "A", 0)

	has, cycles, err := dfs.DetectCycles(g)
	assert.NoError(t, err)
	assert.True(t, has) // cycle should be detected
	// Expect exactly one cycle, normalized to ["A","B","A"]
	assert.Equal(t,
		[][]string{{"A", "B", "A"}},
		cycles,
	)
}

// TestDetectCycles_ThreeNodeCycle covers a 3-node cycle.
func TestDetectCycles_ThreeNodeCycle(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// A -> B -> C -> A forms a 3-node cycle
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("B", "C", 0)
	_, _ = g.AddEdge("C", "A", 0)

	has, cycles, err := dfs.DetectCycles(g)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t,
		[][]string{{"A", "B", "C", "A"}},
		cycles,
	)
}

// TestDetectCycles_FourNodeCycle covers a 4-node cycle.
func TestDetectCycles_FourNodeCycle(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// V -> W -> X -> Y -> Z -> W forms a 4-node cycle
	_, _ = g.AddEdge("V", "W", 0)
	_, _ = g.AddEdge("W", "X", 0)
	_, _ = g.AddEdge("X", "Y", 0)
	_, _ = g.AddEdge("Y", "Z", 0)
	_, _ = g.AddEdge("Z", "W", 0)

	has, cycles, err := dfs.DetectCycles(g)
	assert.NoError(t, err)
	assert.True(t, has)
	// The canonical cycle should start at W
	assert.Equal(t,
		[][]string{{"W", "X", "Y", "Z", "W"}},
		cycles,
	)
}

// TestDetectCycles_Undirected_MultipleDisjointCycles covers two distinct cycles in the same undirected graph.
func TestDetectCycles_Undirected_MultipleDisjointCycles(t *testing.T) {
	g := core.NewGraph() // undirected by default
	// three-node cycle A--B--C--A
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("B", "C", 0)
	_, _ = g.AddEdge("C", "A", 0)

	// four-node cycle W--X--Y--Z--W
	_, _ = g.AddEdge("W", "X", 0)
	_, _ = g.AddEdge("X", "Y", 0)
	_, _ = g.AddEdge("Y", "Z", 0)
	_, _ = g.AddEdge("Z", "W", 0)

	has, cycles, err := dfs.DetectCycles(g)
	assert.NoError(t, err)
	assert.True(t, has)
	// We expect two cycles: ["A","B","C","A"] and ["W","X","Y","Z","W"], in any order
	assert.ElementsMatch(t,
		[][]string{{"A", "B", "C", "A"}, {"W", "X", "Y", "Z", "W"}},
		cycles,
	)
	assert.Len(t, cycles, 2)
}

// TestDetectCycles_DirectedMultipleLarge verifies detection of multiple disjoint cycles in a directed graph.
func TestDetectCycles_DirectedMultipleLarge(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// Cycle1: A->B->C->D->E->A
	cycle1 := []string{"A", "B", "C", "D", "E", "A"}
	for i := 0; i < len(cycle1)-1; i++ {
		_, _ = g.AddEdge(cycle1[i], cycle1[i+1], 0)
	}
	// Cycle2: F->G->H->F
	cycle2 := []string{"F", "G", "H", "F"}
	for i := 0; i < len(cycle2)-1; i++ {
		_, _ = g.AddEdge(cycle2[i], cycle2[i+1], 0)
	}
	// Connect cycles E -> F and add extra vertices I, J with no new edges
	_, _ = g.AddEdge("E", "F", 0)
	_ = g.AddVertex("I")
	_ = g.AddVertex("J")

	has, cycles, err := dfs.DetectCycles(g)
	assert.NoError(t, err)
	assert.True(t, has, "expected at least one cycle in directed graph")

	// Convert found cycles to comma-joined signatures for robust comparison
	sigs := make([]string, len(cycles))
	for i, c := range cycles {
		sigs[i] = strings.Join(c, ",")
	}
	// Expected signatures (canonical rotations)
	exp := []string{strings.Join(cycle1, ","), strings.Join(cycle2, ",")}
	assert.ElementsMatch(t, exp, sigs)
	assert.Len(t, cycles, 2)
}

// TestDetectCycles_UndirectedThreeNode verifies a 3-node undirected cycle is found.
func TestDetectCycles_UndirectedThreeNode(t *testing.T) {
	g := core.NewGraph() // undirected
	// Triangle A--B--C--A
	_, _ = g.AddEdge("A", "B", 0)
	_, _ = g.AddEdge("B", "C", 0)
	_, _ = g.AddEdge("C", "A", 0)

	has, cycles, err := dfs.DetectCycles(g)
	assert.NoError(t, err)
	assert.True(t, has)
	assert.Len(t, cycles, 1)
	// Only one cycle expected: ["A","B","C","A"]
	assert.Equal(t, []string{"A", "B", "C", "A"}, cycles[0])
}

// TestDetectCycles_UndirectedMultipleLarge verifies detection of multiple cycles in an undirected graph.
func TestDetectCycles_UndirectedMultipleLarge(t *testing.T) {
	g := core.NewGraph()
	// Cycle X: 4-node W--X--Y--Z--W
	cyc4 := []string{"W", "X", "Y", "Z", "W"}
	for i := 0; i < len(cyc4)-1; i++ {
		_, _ = g.AddEdge(cyc4[i], cyc4[i+1], 0)
	}
	// Cycle Y: 5-node P--Q--R--S--T--P
	cyc5 := []string{"P", "Q", "R", "S", "T", "P"}
	for i := 0; i < len(cyc5)-1; i++ {
		_, _ = g.AddEdge(cyc5[i], cyc5[i+1], 0)
	}

	has, cycles, err := dfs.DetectCycles(g)
	assert.NoError(t, err)
	assert.True(t, has)

	// Build a set of expected comma-joined cycle signatures
	exp := map[string]struct{}{}
	exp[strings.Join(cyc4, ",")] = struct{}{}
	exp[strings.Join(cyc5, ",")] = struct{}{}

	// Ensure exactly two cycles were found, each matching one expected signature
	assert.Len(t, cycles, 2)
	for _, c := range cycles {
		sig := strings.Join(c, ",")
		assert.Contains(t, exp, sig)
	}
}
