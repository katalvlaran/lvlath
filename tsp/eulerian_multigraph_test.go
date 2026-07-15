package tsp

import (
	"errors"
	"testing"
)

func TestCanonicalizeUndirectedMultigraphPreservesParallelEulerianMultigraph(t *testing.T) {
	raw := [][]int{
		{2, 1, 2, 1},
		{2, 0, 2, 0},
		{1, 0, 1, 0},
	}

	got, err := canonicalizeUndirectedMultigraph(raw)
	if err != nil {
		t.Fatalf("canonicalizeUndirectedMultigraph: %v", err)
	}

	for vertex := range got {
		if len(got[vertex]) != 4 {
			t.Fatalf("degree[%d] got %d want 4 adj=%v", vertex, len(got[vertex]), got)
		}
		if (len(got[vertex]) & 1) != 0 {
			t.Fatalf("degree[%d]=%d must be even adj=%v", vertex, len(got[vertex]), got)
		}
	}

	walk, err := EulerianCircuit(got, 0)
	if err != nil {
		t.Fatalf("EulerianCircuit: %v adj=%v", err, got)
	}
	if len(walk) == 0 || walk[0] != 0 || walk[len(walk)-1] != 0 {
		t.Fatalf("bad closed walk: %v", walk)
	}
}

func TestCanonicalizeUndirectedMultigraphRejectsOddDegreeAfterReciprocalRebuild(t *testing.T) {
	raw := [][]int{
		{1, 1, 2, 2},
		{2, 2},
		{0, 0},
	}

	_, err := canonicalizeUndirectedMultigraph(raw)
	if !errors.Is(err, ErrNonEulerian) {
		t.Fatalf("err got %v want %v", err, ErrNonEulerian)
	}
}

func TestChristofidesKernelMatchingProducesEulerianMultigraphSeedN16Seed2(t *testing.T) {
	const n = 16
	const seed int64 = 2

	dist := seededMetricCompleteForTest(n, seed)

	opts := DefaultOptions()
	opts.Algo = Christofides
	opts.Symmetric = true
	opts.MatchingAlgo = BlossomMatch
	opts.EnableLocalSearch = false
	opts.StartVertex = int(seed % n)

	_, mstAdjacency, err := MinimumSpanningTree(dist)
	if err != nil {
		t.Fatalf("MinimumSpanningTree: %v", err)
	}

	odd := collectOddDegreeVertices(mstAdjacency)
	if len(odd)&1 == 1 {
		t.Fatalf("MST odd set has odd cardinality: odd=%v degrees=%v", odd, degreesOfAdjForDebug(mstAdjacency))
	}

	if _, err = applyChristofidesMatching(odd, dist, mstAdjacency, opts); err != nil {
		t.Fatalf("applyChristofidesMatching: %v odd=%v", err, odd)
	}

	eulerianAdjacency, err := canonicalizeUndirectedMultigraph(mstAdjacency)
	if err != nil {
		t.Fatalf("canonicalizeUndirectedMultigraph: %v odd=%v rawDegrees=%v adj=%v",
			err, odd, degreesOfAdjForDebug(mstAdjacency), mstAdjacency)
	}

	for vertex := range eulerianAdjacency {
		if len(eulerianAdjacency[vertex])&1 == 1 {
			t.Fatalf("canonical degree[%d]=%d odd=%v rawDegrees=%v canonicalDegrees=%v",
				vertex,
				len(eulerianAdjacency[vertex]),
				odd,
				degreesOfAdjForDebug(mstAdjacency),
				degreesOfAdjForDebug(eulerianAdjacency),
			)
		}
	}

	if _, err = EulerianCircuit(eulerianAdjacency, opts.StartVertex); err != nil {
		t.Fatalf("EulerianCircuit: %v odd=%v canonicalDegrees=%v canonicalAdj=%v",
			err, odd, degreesOfAdjForDebug(eulerianAdjacency), eulerianAdjacency)
	}
}

func TestChristofidesBlossomMetricFuzzSeedN16Seed2(t *testing.T) {
	const n = 16
	const seed int64 = 2

	dist := seededMetricCompleteForTest(n, seed)

	opts := DefaultOptions()
	opts.Algo = Christofides
	opts.Symmetric = true
	opts.MatchingAlgo = BlossomMatch
	opts.EnableLocalSearch = false
	opts.StartVertex = int(seed % n)

	result, err := ChristofidesSolve(dist, opts)
	if err != nil {
		t.Fatalf("ChristofidesSolve: %v", err)
	}
	if err = validateResultForTest(result, n, opts.StartVertex, Christofides); err != nil {
		t.Fatalf("result: %v result=%+v", err, result)
	}
}

func TestEulerianCircuitPairsParallelEdgesByOppositeDirection(t *testing.T) {
	adj := [][]int{
		{1, 4},
		{0, 4},
		{7, 9},
		{4, 6, 10, 10},
		{0, 1, 3, 8},
		{7, 11},
		{3, 12},
		{2, 5},
		{4, 10},
		{2, 12},
		{3, 3, 8, 14},
		{5, 14},
		{6, 9},
		{14, 15},
		{10, 11, 13, 15},
		{13, 14},
	}

	walk, err := EulerianCircuit(adj, 2)
	if err != nil {
		t.Fatalf("EulerianCircuit: %v", err)
	}

	halfEdges := 0
	for vertex := range adj {
		if len(adj[vertex])&1 == 1 {
			t.Fatalf("degree[%d]=%d must be even", vertex, len(adj[vertex]))
		}
		halfEdges += len(adj[vertex])
	}

	if len(walk) != halfEdges/2+1 {
		t.Fatalf("walk length got %d want %d walk=%v", len(walk), halfEdges/2+1, walk)
	}
	if walk[0] != 2 || walk[len(walk)-1] != 2 {
		t.Fatalf("walk must start and end at 2: %v", walk)
	}
}

func degreesOfAdjForDebug(adj [][]int) []int {
	degrees := make([]int, len(adj))
	for vertex := range adj {
		degrees[vertex] = len(adj[vertex])
	}

	return degrees
}
