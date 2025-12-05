// Package core_test verifies thread-safety of core.Graph under concurrent operations.
package core_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/katalvlaran/lvlath/core"
	"github.com/stretchr/testify/require"
)

// TestConcurrentAddEdge ensures that concurrent AddEdge calls
// on a graph allowing multi-edges are safe and all neighbors appear.
func TestConcurrentAddEdge(t *testing.T) {
	// Create graph with multi-edge support
	g := core.NewGraph(core.WithMultiEdges())
	const num = 200 // number of concurrent adds
	var wg sync.WaitGroup
	wg.Add(num)

	// Launch num goroutines to add edges from X to V{i}
	for i := 0; i < num; i++ {
		go func(id int) {
			defer wg.Done() // signal completion
			_, err := g.AddEdge("X", fmt.Sprintf("V%d", id), 0)
			require.NoError(t, err)
		}(i)
	}
	wg.Wait() // wait for all adds to finish

	// Retrieve neighbors of X; expect num edges
	nbs, err := g.Neighbors("X")
	require.NoError(t, err) // no error from Neighbors
	require.Len(t, nbs, num, "expected %d unique neighbors", num)
}

// TestConcurrentAddRemoveEdge mixes AddEdge and RemoveEdge calls
// to verify no races or panics occur under concurrent modification.
func TestConcurrentAddRemoveEdge(t *testing.T) {
	// Create graph with weights and multi-edge support
	g := core.NewGraph(core.WithWeighted(), core.WithMultiEdges())
	// Pre-add a base vertex to anchor edges
	require.NoError(t, g.AddVertex("Base"))

	const rounds = 100 // number of add/remove rounds
	var wg sync.WaitGroup
	wg.Add(2 * rounds)

	for i := 0; i < rounds; i++ {
		// Concurrent edge addition
		go func(id int) {
			defer wg.Done()
			_, _ = g.AddEdge("Base", fmt.Sprintf("V%d", id), float64(id))
		}(i)

		// Concurrent edge removal
		go func() {
			defer wg.Done()
			// Iterate current edges and try to remove each
			for _, e := range g.Edges() {
				_ = g.RemoveEdge(e.ID)
			}
		}()
	}
	wg.Wait() // wait for all operations to complete
	// Graph remains consistent and race-free if no panic
}

// TestConcurrentNeighborsAndClone validates concurrent reads
// (Neighbors) and clones do not race with each other.
func TestConcurrentNeighborsAndClone(t *testing.T) {
	// Create graph with loops, weights, and multi-edge support
	g := core.NewGraph(core.WithWeighted(), core.WithMultiEdges(), core.WithLoops())
	// Prepare 50 self-loops on A
	for i := 0; i < 50; i++ {
		_, _ = g.AddEdge("A", "A", float64(i))
	}

	const readers = 50 // number of concurrent readers
	const cloners = 20 // number of concurrent cloners
	var wg sync.WaitGroup
	wg.Add(readers + cloners)

	// Launch concurrent reader goroutines
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			// Retrieve neighbors of A; each should see 50 loops
			nbs, err := g.Neighbors("A")
			require.NoError(t, err)
			require.Len(t, nbs, 50)
		}()
	}

	// Launch concurrent clone goroutines
	for i := 0; i < cloners; i++ {
		go func() {
			defer wg.Done()
			// Clone the graph; safe for concurrent reads
			_ = g.Clone()
		}()
	}

	wg.Wait() // wait for all readers and cloners
}
