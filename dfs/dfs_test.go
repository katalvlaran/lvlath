package dfs_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/dfs"
)

// buildChain creates a directed chain graph of length n: 0→1→2→…→n-1
func buildChain(n int) *core.Graph {
	g := core.NewGraph(core.WithDirected(true))
	for i := 0; i < n-1; i++ {
		u := "N" + strconv.Itoa(i)
		v := "N" + strconv.Itoa(i+1)
		g.AddVertex(u)
		g.AddVertex(v)
		g.AddEdge(u, v, 0)
	}

	return g
}

// buildBinaryTree creates a complete binary tree of depth d (nodes = 2^d-1).
// IDs: "T-1","T-2",…,"T-N".
func buildBinaryTree(depth int) *core.Graph {
	g := core.NewGraph(core.WithDirected(true))
	// numbering from 1 to (2^depth -1)
	maxD := (1 << depth) - 1
	for i := 1; i <= maxD; i++ {
		id := fmt.Sprintf("T-%d", i)
		g.AddVertex(id)
		parent := fmt.Sprintf("T-%d", i/2)
		if i > 1 {
			g.AddEdge(parent, id, 0)
		}
	}

	return g
}

func TestDFS_NilGraph(t *testing.T) {
	res, err := dfs.DFS(nil, "A")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, dfs.ErrGraphNil)
}

func TestDFS_StartNotFound(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	res, err := dfs.DFS(g, "X")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, dfs.ErrStartVertexNotFound)
}

func TestDFS_SingleVertex_NoEdges(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	err := g.AddVertex("X")
	assert.NoError(t, err)

	res, err := dfs.DFS(g, "X")
	assert.NoError(t, err)
	assert.Equal(t, []string{"X"}, res.Order)
	assert.True(t, res.Visited["X"])
	assert.Equal(t, 0, res.Depth["X"])
	_, hasParent := res.Parent["X"]
	assert.False(t, hasParent, "start vertex should have no parent")
}

func TestDFS_SelfLoop(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true), core.WithLoops())
	err := g.AddVertex("A")
	assert.NoError(t, err)
	edgeID, err := g.AddEdge("A", "A", 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, edgeID)

	res, err := dfs.DFS(g, "A")
	assert.NoError(t, err)
	// Self-loop should not create additional entries
	assert.Equal(t, []string{"A"}, res.Order)
	assert.True(t, res.Visited["A"])
}

func TestDFS_ChainAndDepthParent(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)

	res, err := dfs.DFS(g, "A")
	assert.NoError(t, err)
	// Post-order: C, B, A
	assert.Equal(t, []string{"C", "B", "A"}, res.Order)
	assert.Equal(t, "B", res.Parent["C"])
	assert.Equal(t, 2, res.Depth["C"])
}

func TestDFS_Disconnected(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	g.AddEdge("A", "B", 0)
	err := g.AddVertex("C")
	assert.NoError(t, err)

	res, err := dfs.DFS(g, "A")
	assert.NoError(t, err)
	// Only reachable vertices
	assert.Equal(t, []string{"B", "A"}, res.Order)
	assert.False(t, res.Visited["C"], "disconnected vertex should not be visited")
}

func TestDFS_MaxDepth(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	g.AddEdge("A", "B", 0)
	g.AddEdge("B", "C", 0)

	res, err := dfs.DFS(g, "A", dfs.WithMaxDepth(0))
	assert.NoError(t, err)
	// Depth limit = 0, only A
	assert.Equal(t, []string{"A"}, res.Order)
	assert.False(t, res.Visited["B"])
}

func TestDFS_FilterNeighbor(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	g.AddEdge("A", "B", 0)
	g.AddEdge("A", "C", 0)

	// Skip C
	res, err := dfs.DFS(g, "A", dfs.WithFilterNeighbor(func(id string) bool {
		return id != "C"
	}))
	assert.NoError(t, err)
	// Only B then A
	assert.Equal(t, []string{"B", "A"}, res.Order)
	assert.False(t, res.Visited["C"], "filtered neighbor should not be visited")
}

func TestDFS_OnExitError(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	g.AddEdge("A", "B", 0)

	res, err := dfs.DFS(g, "A", dfs.WithOnExit(func(id string) error {
		if id == "B" {
			return errors.New("halt at B on exit")
		}

		return nil
	}))
	assert.NotNil(t, res)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "OnExit hook for \"B\"")
	assert.Empty(t, res.Order, "no post-order on hook error")
}

func TestDFS_Cancellation(t *testing.T) {
	g := core.NewGraph(core.WithDirected(true))
	// build chain
	for i := 0; i < 1000; i++ {
		src := fmt.Sprintf("N%d", i)
		dst := fmt.Sprintf("N%d", i+1)
		g.AddVertex(src)
		g.AddVertex(dst)
		g.AddEdge(src, dst, 0)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := dfs.DFS(g, "N0", dfs.WithContext(ctx))
	assert.NotNil(t, res)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, res.Order, "no completion when canceled immediately")
}

func TestDFS_LargeChain_PostOrderDepthParent(t *testing.T) {
	const n = 10
	g := buildChain(n)
	res, err := dfs.DFS(g, "N0")
	assert.NoError(t, err)

	// Post order: N9, N8, …, N0
	expected := make([]string, n)
	for i := n - 1; i >= 0; i-- {
		expected[n-1-i] = "N" + strconv.Itoa(i)
	}
	assert.Equal(t, expected, res.Order, "Chain post-order reversed")

	// Checking the depth and parent of the last node
	assert.Equal(t, n-1, res.Depth["N"+strconv.Itoa(n-1)])
	assert.Equal(t, "N"+strconv.Itoa(n-2), res.Parent["N"+strconv.Itoa(n-1)])
}

func TestDFS_BinaryTree_TraversalAndVisited(t *testing.T) {
	const depth = 4 // 15 nodes
	g := buildBinaryTree(depth)
	res, err := dfs.DFS(g, "T-1")
	assert.NoError(t, err)

	// All edges must be visited
	assert.Len(t, res.Visited, (1<<depth)-1)
	for i := 1; i < (1 << depth); i++ {
		id := fmt.Sprintf("T-%d", i)
		assert.True(t, res.Visited[id], "vertex %s must be visited", id)
	}

	// Post order: size 15, root should be last
	assert.Len(t, res.Order, (1<<depth)-1)
	assert.Equal(t, "T-1", res.Order[len(res.Order)-1], "root must finish last")
}

func TestDFS_OnVisitOnExitHooks(t *testing.T) {
	g := buildBinaryTree(3) // 7 nodes
	var pre, post []string

	res, err := dfs.DFS(g, "T-1",
		dfs.WithOnVisit(func(id string) error {
			pre = append(pre, id)
			if id == "T-4" {
				return errors.New("stop at T-4")
			}

			return nil
		}),
		dfs.WithOnExit(func(id string) error {
			post = append(post, id)

			return nil
		}),
	)
	assert.NotNil(t, res)
	assert.ErrorContains(t, err, "OnVisit hook for \"T-4\"")
	// Make sure pre-order contains root and T-2,T-4
	assert.Contains(t, pre, "T-1")
	assert.Contains(t, pre, "T-4")
	// Since the error occurred in OnVisit, post-order remains empty
	assert.Empty(t, post)
	assert.Empty(t, res.Order)
}

func TestDFS_CancellationImmediate(t *testing.T) {
	g := buildChain(100)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediate

	res, err := dfs.DFS(g, "N0", dfs.WithContext(ctx))
	assert.NotNil(t, res)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, res.Order, "no nodes should finish when canceled immediately")
}

func TestDFS_DisconnectedComponent(t *testing.T) {
	// Create two separate sub graphs with 5 nodes each
	g := buildChain(5)
	for i := 5; i < 10; i++ {
		id := "M" + strconv.Itoa(i)
		g.AddVertex(id)
	}
	res, err := dfs.DFS(g, "N0")
	assert.NoError(t, err)
	// Must be only N0..N4
	assert.ElementsMatch(t,
		[]string{"N4", "N3", "N2", "N1", "N0"},
		res.Order,
	)
	for i := 5; i < 10; i++ {
		assert.False(t, res.Visited["M"+strconv.Itoa(i)], "disconnected M%d should not be visited", i)
	}
}
