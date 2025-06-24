// Package dfs implements robust cycle detection for both directed and undirected core.Graphs.
// DetectCycles enumerates all simple cycles using depth-first search with three-color
// marking and back-edge detection. It honors per-edge Directed flags when mixed-edge
// mode is enabled, correctly handles self-loops and trivial 2-cycles in undirected graphs,
// and produces canonical minimal rotations of each cycle via Booth’s algorithm in O(L) time.
// The final cycle list is sorted for deterministic output.
//
// Complexity:
//
//   - Time:   O(V + E + C·L)   (V=#vertices, E=#edges, C=#cycles, L=avg cycle length)
//   - Memory: O(V + L_max)     (recursion stack + state map + cycle storage)
package dfs

import (
	"fmt"
	"sort"

	"github.com/katalvlaran/lvlath/core"
)

// DetectCycles inspects graph g for all simple cycles.
// Returns (true, cycles, nil) if any cycles are found;
// if no cycles, returns (false, nil, nil).
// If a neighbor-fetch error occurs, returns (false, nil, error).
func DetectCycles(g *core.Graph) (bool, [][]string, error) {
	// 1) Nil graph is treated as cycle-free
	if g == nil {
		return false, nil, nil
	}

	// 2) Prepare visitation state:
	//    White=0 (unvisited), Gray=1 (in recursion stack), Black=2 (completed)
	verts := g.Vertices()                         // sorted list of vertex IDs
	state := make(map[string]int, len(verts))     // tracks visitation state per vertex
	path := make([]string, 0, len(verts))         // current DFS path (stack) for cycle reconstruction
	seen := make(map[string]struct{}, len(verts)) // deduplication set for cycle signatures
	var cycles [][]string                         // collected distinct cycles

	// 3) Launch DFS from each unvisited vertex
	for _, v := range verts {
		if state[v] == White {
			// If an error occurs during DFS (e.g., neighbor lookup fails),
			// we abort and return that error.
			if err := dfsVisit(g, v, "", state, &path, seen, &cycles); err != nil {
				return false, nil, fmt.Errorf("dfs: DetectCycles: %w", err)
			}
		}
	}

	// 4) Sort cycles lexicographically by their comma-joined signature,
	//    ensuring a deterministic output order.
	sort.Slice(cycles, func(i, j int) bool {
		return JoinSig(cycles[i]) < JoinSig(cycles[j])
	})

	// 5) Return whether any cycles were found
	if len(cycles) == 0 {
		return false, nil, nil
	}

	return true, cycles, nil
}

// dfsVisit performs recursive DFS from vertex 'id', tracking the 'parent' to skip trivial back-edges.
// It records any back-edge Gray→Gray cycles it encounters and appends them to 'cycles'.
// Arguments:
//   - g: the graph being traversed
//   - id: current vertex ID
//   - parent: the immediate predecessor of 'id' in the DFS (empty string for root calls)
//   - state: map to track each vertex's visitation state (White, Gray, Black)
//   - path: pointer to a slice representing the current DFS path stack
//   - seen: set of canonical cycle signatures (to avoid duplicates)
//   - cycles: pointer to a slice of discovered cycles (each cycle is a []string)
//
// Returns an error if neighbor iteration fails.
func dfsVisit(
	g *core.Graph,
	id, parent string,
	state map[string]int,
	path *[]string,
	seen map[string]struct{},
	cycles *[][]string,
) error {
	// 1) Mark current vertex as Gray (in progress)
	state[id] = Gray

	// 2) Push 'id' onto the DFS path stack for later cycle reconstruction
	*path = append(*path, id)

	// 3) Retrieve all incident edges; propagate any lookup error upward
	edges, err := g.Neighbors(id)
	if err != nil {
		// Wrap the error to provide context
		return fmt.Errorf("Neighbors(%q): %w", id, err)
	}

	// 4) Explore each edge from 'id'
	for _, e := range edges {
		// 4a) If this edge should be skipped (self-loop not allowed, trivial backtrack in undirected,
		//     or a directed edge not originating from 'id'), skip it.
		if shouldSkipEdge(e, id, parent, g) {
			continue
		}

		// 4b) Determine actual neighbor ID, handling mixed/mirrored edges:
		//     In an undirected or mixed-edge graph, if e.Directed == false and e.To == id,
		//     the neighbor is e.From; otherwise it's simply e.To.
		nbr := getNeighborID(e, id, g)

		// 4c) Examine neighbor's visitation state
		switch state[nbr] {
		case White:
			// 4c.i) Unvisited: recurse deeper
			if err = dfsVisit(g, nbr, id, state, path, seen, cycles); err != nil {
				return err // propagate error
			}
		case Gray:
			// 4c.ii) Found a back-edge Gray→Gray: potential cycle detected
			//        Check for trivial loops and trivial 2-cycles in undirected graphs

			// Find index of 'nbr' in current 'path' stack
			idx := IndexOf(*path, nbr)
			// Length of segment from 'nbr' to current vertex
			segLen := len(*path) - idx

			// Skip trivial self-loop [v, v] if loops are not allowed
			if segLen < 2 && !g.Looped() {
				continue
			}
			// Skip trivial 2-cycle [u, v, u] if the graph is undirected
			if segLen == 2 && !g.Directed() {
				continue
			}
			// 4c.iii) Valid cycle of length ≥2 (or loop): record it
			recordCycle(nbr, *path, seen, cycles)
		}
	}

	// 5) Backtrack: pop 'id' from path stack and mark it Black (fully explored)
	*path = (*path)[:len(*path)-1]
	state[id] = Black

	return nil
}

// shouldSkipEdge determines if 'e' should be ignored during cycle detection from vertex 'id'.
// Three cases are handled:
//  1. Self-loop when loops are disabled.
//  2. Trivial backtrack in an undirected graph (edge back to parent).
//  3. Directed-edge that does not originate from 'id' (for mixed-edge graphs).
func shouldSkipEdge(e *core.Edge, id, parent string, g *core.Graph) bool {
	// 1) Self-loop: skip if loops are not enabled
	if e.From == e.To && !g.Looped() {
		return true
	}
	// 2) Trivial backtrack in undirected graph: skip neighbor == parent
	if !e.Directed && !g.Directed() && e.To == parent {
		return true
	}
	// 3) Directedness check: if edge is marked Directed but 'id' is not its source
	if e.Directed && e.From != id {
		return true
	}

	return false
}

// recordCycle extracts and deduplicates the cycle that ends at 'start'.
// 'path' is the current DFS path stack, containing [ ... start ... current ].
// We perform the following steps:
//  1. Find index of 'start' in 'path'.
//  2. Extract the sub-slice path[idx:] and append 'start' to close the loop.
//  3. Canonicalize the cycle (minimal rotation or its reverse).
//  4. If the canonical signature has not been seen before, append to 'cycles'.
func recordCycle(
	start string,
	path []string,
	seen map[string]struct{},
	cycles *[][]string,
) {
	// 1) Locate index of 'start' in 'path'
	idx := IndexOf(path, start)

	// 2) Extract cycle segment from idx to end, then close it by appending 'start'
	seq := append([]string(nil), path[idx:]...) // copy slice from idx to end
	seq = append(seq, start)                    // close cycle

	// 3) Canonicalize the sequence (handles rotations and reversed order)
	sig, canon := canonical(seq)
	// 4) If this canonical signature is new, add to 'cycles'
	if _, exists := seen[sig]; !exists {
		seen[sig] = struct{}{}
		*cycles = append(*cycles, canon)
	}
}

// canonical computes the lexicographically minimal rotation of 'cycle' and its reversal.
// Returns:
//   - sig: the comma-joined signature of the minimal closed cycle,
//   - canon: the closed cycle slice [v0, v1, ..., v0] in canonical order.
//
// Steps:
//  1. Let n = len(cycle) - 1 (since cycle[0] == cycle[n]).
//  2. Extract base = cycle[:n] (drop the duplicate last element).
//  3. Compute minimal forward rotation rotF = MinimalRotation(base).
//  4. Compute minimal rotation of reversed sequence rotB = MinimalRotation(Reverse(base)).
//  5. Compare rotF vs rotB lexicographically and pick the smaller as 'picker'.
//  6. Close 'picker' by appending picker[0] at the end to form a closed loop.
//  7. Build signature sig = JoinSig(closed).
func canonical(cycle []string) (string, []string) {
	// Length of the true cycle (excluding the duplicated last element)
	n := len(cycle) - 1
	base := cycle[:n] // drop trailing repeat for rotation

	// 1) Compute minimal forward rotation
	rotF := MinimalRotation(base)
	// 2) Compute minimal rotation of reversed sequence
	rotB := MinimalRotation(Reverse(base))

	// 3) Choose lexicographically smaller rotation
	picker := rotF
	if Compare(rotB, rotF) < 0 {
		picker = rotB
	}

	// 4) Close cycle by appending first element to the end
	closed := append(append([]string(nil), picker...), picker[0])
	// 5) Build signature by joining with commas
	sig := JoinSig(closed)

	return sig, closed
}

// getNeighborID returns the actual neighbor of 'id' via edge 'e'.
// In an undirected or mixed-edge graph, if e.Directed == false and e.To == id,
// then the neighbor is e.From; otherwise it's simply e.To.
// This ensures correct traversal direction for cycle detection.
func getNeighborID(e *core.Edge, id string, g *core.Graph) string {
	if !g.Directed() && !e.Directed && e.To == id {
		return e.From
	}

	return e.To
}
