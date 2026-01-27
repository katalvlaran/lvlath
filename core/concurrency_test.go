// SPDX-License-Identifier: MIT
// Package core_test verifies thread-safety of core.Graph under concurrent operations.
//
// Purpose:
//   - Demonstrate race-free behavior under concurrent Add/Remove/Read/Clone.
//   - Enforce the test rule: goroutines never call *testing.T (errors flow via channels).

package core_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/katalvlaran/lvlath/core"
)

// TestConcurrentAddEdge ENSURES concurrent AddEdge is safe and observable via Neighbors.
//
// Implementation:
//   - Stage 1: Create a multi-edge graph to avoid endpoint-collision rejections.
//   - Stage 2: Spawn N goroutines calling AddEdge(from, V{i}, 0).
//   - Stage 3: Collect errors via channel (no *testing.T inside goroutines).
//   - Stage 4: Assert Neighbors(from) length equals N.
//
// Behavior highlights:
//   - Validates internal locking/atomic edge-ID generation under contention.
//   - Validates read API (Neighbors) after concurrent writes.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal if any AddEdge returns error.
//   - Fatal if Neighbors returns error or wrong length.
//
// Determinism:
//   - Deterministic assertions; scheduling nondeterminism is tolerated.
//
// Complexity:
//   - Time O(N) adds + O(N) neighbor enumeration, Space O(N) for channel buffering.
//
// Notes:
//   - The test asserts COUNT, not ordering; ordering determinism is covered elsewhere.
//
// AI-Hints:
//   - Keep N moderate; pair with `go test -race` to validate locking rigorously.
func TestConcurrentAddEdge(t *testing.T) {
	g := core.NewGraph(core.WithMultiEdges())

	var (
		wg    sync.WaitGroup
		errCh chan error
	)

	wg.Add(NConcurrentAdds)
	errCh = make(chan error, NConcurrentAdds)

	var i int
	for i = 0; i < NConcurrentAdds; i++ {
		go func(id int) {
			defer wg.Done()

			_, err := g.AddEdge(VertexX, fmt.Sprintf("V%d", id), Weight0)
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	MustNoErrorsFromChan(t, errCh, "Concurrent AddEdge")

	var (
		nbs []*core.Edge
		err error
	)

	nbs, err = g.Neighbors(VertexX)
	MustNoError(t, err, "Neighbors(X)")
	MustEqualInt(t, len(nbs), NConcurrentAdds, "Neighbors(X) length after concurrent AddEdge")
}

// TestConcurrentAddRemoveEdge MIXES AddEdge and RemoveEdge to validate no panics/races.
//
// Implementation:
//   - Stage 1: Create a weighted multi-edge graph.
//   - Stage 2: Ensure a base vertex exists.
//   - Stage 3: For each round:
//   - Spawn one goroutine adding an edge Base->V{i}.
//   - Spawn one goroutine iterating current Edges() and attempting RemoveEdge.
//   - Stage 4: Collect only UNEXPECTED errors via channel.
//
// Behavior highlights:
//   - Accepts ErrEdgeNotFound during removals as a valid interleaving outcome.
//   - Detects unexpected internal-state violations under contention.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal if AddEdge returns error.
//   - Fatal if RemoveEdge returns an unexpected error.
//
// Determinism:
//   - Nondeterministic schedule; deterministic acceptance policy.
//
// Complexity:
//   - Time O(R * (workload)), Space O(R) channel buffering.
//
// Notes:
//   - This is primarily a race/panic detector; run with `-race`.
//
// AI-Hints:
//   - Keep the “acceptable error set” explicit; never silently ignore unknown errors.
func TestConcurrentAddRemoveEdge(t *testing.T) {
	g := core.NewGraph(core.WithWeighted(), core.WithMultiEdges())

	MustNoError(t, g.AddVertex(VertexBase), "AddVertex(Base)")

	var (
		wg    sync.WaitGroup
		errCh chan error
	)

	wg.Add(2 * NConcurrentRounds)
	errCh = make(chan error, 2*NConcurrentRounds)

	var i int
	for i = 0; i < NConcurrentRounds; i++ {
		go func(id int) {
			defer wg.Done()

			_, err := g.AddEdge(VertexBase, fmt.Sprintf("V%d", id), float64(id))
			if err != nil {
				errCh <- err
			}
		}(i)

		go func() {
			defer wg.Done()

			var e *core.Edge
			for _, e = range g.Edges() {
				err := g.RemoveEdge(e.ID)
				if err == nil {
					continue
				}
				if err == core.ErrEdgeNotFound {
					continue
				}
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)

	MustNoErrorsFromChan(t, errCh, "Concurrent Add/Remove")
}

// TestConcurrentNeighborsAndClone VALIDATES concurrent Neighbors and Clone do not race.
//
// Implementation:
//   - Stage 1: Create loop-enabled multi-edge weighted graph.
//   - Stage 2: Add NLoops self-loops on A.
//   - Stage 3: Spawn NReaders goroutines calling Neighbors(A) and verifying length.
//   - Stage 4: Spawn NCloners goroutines calling Clone().
//   - Stage 5: Report all errors via channel.
//
// Behavior highlights:
//   - Ensures read-heavy workloads remain safe under clone pressure.
//
// Inputs:
//   - None (uses package constants).
//
// Returns:
//   - None.
//
// Errors:
//   - Fatal if Neighbors/Clone triggers errors or length mismatch.
//
// Determinism:
//   - Schedule nondeterministic; contract checks deterministic.
//
// Complexity:
//   - Time O(NReaders*NLoops) neighbor enumeration, Space O(NReaders+NCloners).
//
// Notes:
//   - Neighbors length equality assumes loops are represented once in adjacency for self-loop vertex.
//
// AI-Hints:
//   - If internal representation changes, update ONLY the contract statement, not the test harness style.
func TestConcurrentNeighborsAndClone(t *testing.T) {
	g := core.NewGraph(core.WithWeighted(), core.WithMultiEdges(), core.WithLoops())

	var (
		i   int
		err error
	)

	for i = 0; i < NLoops; i++ {
		_, err = g.AddEdge(VertexA, VertexA, float64(i))
		MustNoError(t, err, "AddEdge(A,A,w) setup")
	}

	var (
		wg    sync.WaitGroup
		errCh chan error
	)

	wg.Add(NReaders + NCloners)
	errCh = make(chan error, NReaders+NCloners)

	for i = 0; i < NReaders; i++ {
		go func() {
			defer wg.Done()

			nbs, err := g.Neighbors(VertexA)
			if err != nil {
				errCh <- err
				return
			}
			if len(nbs) != NLoops {
				errCh <- fmt.Errorf("Neighbors(A) length mismatch: got=%d want=%d", len(nbs), NLoops)
				return
			}
		}()
	}

	for i = 0; i < NCloners; i++ {
		go func() {
			defer wg.Done()

			_ = g.Clone()
		}()
	}

	wg.Wait()
	close(errCh)

	MustNoErrorsFromChan(t, errCh, "Concurrent Neighbors/Clone")
}
