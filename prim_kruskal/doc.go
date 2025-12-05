// Package prim_kruskal provides two battle-tested algorithms for computing the Minimum Spanning Tree (MST)
// on an undirected, weighted *core.Graph: Prim’s algorithm and Kruskal’s algorithm.
//
// What & Why
//
//   - What is an MST?
//     Given an undirected, connected, weighted graph G = (V, E), an MST is a subset T ⊆ E such that
//     T connects all vertices in V (i.e., spans the graph) and the sum of weights of edges in T is minimized.
//
//   - Why MST matters:
//
//   - Network Design: Build cost-efficient communication or transportation networks (e.g., fiber-optic backbones, road systems).
//
//   - Clustering: In machine learning, MST helps form clusters by cutting the largest edges in the tree.
//
//   - Subroutines: MST is a building block in many approximation algorithms (e.g., Steiner trees, k-centers) and graph partitioning tasks.
//
//   - Theoretical Insights: MST reveals structural properties of weighted graphs (e.g., cut-property, cycle-property).
//
// Algorithms Provided
//
//   - Kruskal(g *core.Graph) ([]core.Edge, float64, error)
//
//   - Strategy: Sort all edges by weight, then iterate from smallest to largest. Use a Disjoint-Set (Union-Find) data structure
//     to merge vertices component-by-component, skipping edges whose endpoints are already connected. Stop once |V|−1 edges have been added.
//
//   - Complexity:
//
//   - Time: O(E log E + α(V)*E) ≈ O(E log V) because sorting dominates (E = number of edges, V = number of vertices, α = inverse Ackermann).
//
//   - Space: O(V + E) for storing parent/rank arrays and the sorted edge list.
//
//   - Determinism: graph.Edges() returns edges in ascending ID order; we perform a stable sort by weight, ensuring that ties break predictably.
//
//   - Prim(g *core.Graph, root string) ([]core.Edge, float64, error)
//
//   - Strategy: Grow a single tree starting from a specified root vertex. Maintain a min-heap (priority queue) of candidate edges
//     that connect the current tree to an outside vertex. At each step, extract the smallest-weight edge that adds a new vertex.
//     Continue until |V|−1 edges have been added.
//
//   - Complexity:
//
//   - Time: O(E log V) because each edge may be pushed/popped on the heap once (heap operations cost O(log V)).
//
//   - Space: O(V + E) for the visited set and heap storage.
//
//   - Use-Case: When the graph is large but you know a reasonable starting point (root), and you want to avoid sorting all edges upfront.
//
// When to Choose Which Algorithm
//
//   - Prim (O(E log V))
//
//   - Preferred for very sparse graphs (E ≈ O(V)), since heap operations on O(V) vertices/edges are efficient.
//
//   - Requires a valid starting vertex (root). If you know a “good” root (e.g., a central hub), Prim often outperforms Kruskal.
//
//   - Kruskal (O(E log E + α(V)*E))
//
//   - Easy to implement when you simply need one global pass over all edges.
//
//   - Preferred when you want a conceptually straightforward sort-and-union approach, or if you only occasionally compute MST (no need to maintain a heap).
//
//   - If E » V (very dense graph), sorting all edges (E log E) may be heavier than Prim’s edge-driven expansion (E log V).
//
// Error Conditions
//
//	Both Kruskal and Prim return meaningful sentinel errors to signal invalid inputs or unreachable MST scenarios:
//
//	- ErrInvalidGraph
//	    - Graph is nil, OR
//	    - graph.Directed() == true (MST requires undirected), OR
//	    - !graph.Weighted() (MST requires nonzero weights), OR
//	    - graph.HasDirectedEdges() == true (if mixed-mode per-edge overrides exist; MST requires purely undirected).
//
//	- ErrEmptyRoot (Prim only)
//	    - root == "" (no starting vertex specified).
//
//	- core.ErrVertexNotFound (Prim only)
//	    - root does not exist in graph.Vertices().
//
//	- ErrDisconnected
//	    - |V| == 0 (empty graph), OR
//	    - |V| > 1 but the graph is not fully connected (no spanning tree can cover all vertices).
//
// GoDoc Summary
//
//   - Kruskal(graph *core.Graph) ([]core.Edge, float64, error)
//     Compute MST via global edge sort + union-find.
//     Returns (edges, totalWeight, nil) on success, else returns (nil, 0, ErrInvalidGraph/ErrDisconnected).
//
//   - Prim(graph *core.Graph, root string) ([]core.Edge, float64, error)
//     Compute MST via a min-heap expansion from a specified root vertex.
//     Returns (edges, totalWeight, nil) on success, else returns (nil, 0, ErrInvalidGraph/ErrEmptyRoot/core.ErrVertexNotFound/ErrDisconnected).
//
// Package prim_kruskal strives for correctness, determinism, and performance:
//
//   - All vertex and edge lists from core.Graph are sorted (by ID) to ensure repeatable behavior.
//   - Kruskal uses a stable sort by weight so that, for equal weights, edges appear in original insertion order.
//   - Prim uses a standard min-heap (heap.Interface) to achieve O(E log V) time with minimal memory overhead.
//
// For examples of usage, see the example_test.go file in this package
// For deep dive (pseudocode, math formulation, examples), see docs/PRIM_&_KRUSKAL.md..
package prim_kruskal
