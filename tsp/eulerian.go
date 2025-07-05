package tsp

// EulerianCircuit returns an Eulerian tour (circuit) of the undirected multigraph
// given by adjacency lists adj, starting and ending at vertex start.
// It implements Hierholzer’s algorithm in O(E).
func EulerianCircuit(adj [][]int, start int) []int {
	// Make a local copy of edge lists so we can remove edges
	local := make([][]int, len(adj))
	for u := range adj {
		local[u] = append([]int(nil), adj[u]...)
	}

	var circuit []int     // holds the final circuit
	stack := []int{start} // DFS stack, initialized with start

	for len(stack) > 0 {
		u := stack[len(stack)-1]
		if len(local[u]) == 0 {
			// no more edges: backtrack
			circuit = append(circuit, u)
			stack = stack[:len(stack)-1]
		} else {
			// traverse one edge u→v
			v := local[u][len(local[u])-1]
			local[u] = local[u][:len(local[u])-1]
			// remove the reverse edge v→u
			for i, x := range local[v] {
				if x == u {
					local[v] = append(local[v][:i], local[v][i+1:]...)
					break
				}
			}
			// continue DFS from v
			stack = append(stack, v)
		}
	}

	return circuit
}
