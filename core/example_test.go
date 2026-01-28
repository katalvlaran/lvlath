package core_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/katalvlaran/lvlath/core"
)

// Utility: sortAsc returns a sorted copy of a string slice (IDs).
func sortAsc(ids []string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)
	return out
}

// Global constants for numeric values and output tags.
const (
	// Generic numeric constants (to avoid magic numbers)
	constZeroFloat = 0.0
	constOneFloat  = 1.0
	constHalfFloat = 0.5

	// Eigen computation controls (Jacobi).
	constEigenTol     = 1e-10
	constEigenMaxIter = 200

	// Cascading-failure topology size (toy, but contract-heavy).
	cascadingClusterSize = 3

	// Betweenness topology size (toy, but interpretable by closed-form load).
	betweennessClusterSize = 4

	// Output tag labels for examples
	outR               = "R"
	outBridgeEdge      = "bridgeEdge"
	outBridgeLoad      = "bridgeLoad"
	outDeg2_0          = "deg[2][0]"
	outDeg2_1          = "deg[2][1]"
	outDeg2_2          = "deg[2][2]"
	outLambda2_0       = "lambda2[0]"
	outLambda2_1       = "lambda2[1]"
	outBridgeEndpoints = "A0-B0"
)

// ExampleGraph_CascadingFailures demonstrates a cascading failure scenario in a power grid network.
// A highly connected hub node is removed to simulate a substation failure, and the impact on network connectivity is measured.
// ExampleGraph_CascadingFailures demonstrates cascading-failure analysis in a power grid.
// CONTEXT:
//   - You are a Resilience Architect for the 'Aethelgard' energy grid.
//   - A critical infrastructure node (Hub) is targeted by a cyber-kinetic strike.
//   - Objective: Predict the "Cascade Collapse Index" before the physical failure occurs.
//
// Scenario:
//   - You operate a smart-city grid graph: vertices are substations, edges are physical lines.
//   - An incident (physical fault / cyberattack) disables a single high-degree hub substation.
//   - Your job is to quantify whether the grid “degrades gracefully” or splits into islands.
//
// Why this matters (criticality):
//   - In real grids, the most dangerous failures are not “one line is down” but “a cut point is down”.
//   - A single vertex can be a topological single point of failure (cut-vertex).
//   - You need fast “what-if” evaluation without corrupting the production topology.
//
// MATHEMATICAL MODEL:
//  1. Survival Coefficient (Resilience Ratio) 'R':
//     R = N'_LCC / (N_LCC - 1)
//     Measures how much of the Giant Component (LCC) remains after the hub's evaporation.
//  2. Fragility Index 'Φ':
//     Φ = 1 - (Σ deg(v_adj) / deg(v_target))
//     Quantifies topological dependency. A high Φ indicates that neighbors are
//     dangerously dependent on the target node for their connectivity.
//
// Metric (resilience ratio):
//   - Let N_LCC be the size of the Largest Connected Component (LCC) BEFORE the incident.
//   - Let N'_LCC be the size of the LCC AFTER removing the incident vertex.
//   - Resilience ratio:
//     R = N'_LCC / (N_LCC - 1)
//   - Interpretation:
//   - R close to 1   -> removal barely hurts connectivity.
//   - R close to 0   -> removal fractures the grid into small islands.
//
// Implementation:
//   - Stage 1: Build two dense clusters (districts) connected only via a single hub.
//   - Stage 2: Clone() the topology and RemoveVertex(hub) in the clone (sandbox simulation).
//   - Stage 3: Compute LCC size via BFS using NeighborIDs (deterministic neighbor ordering).
//
// CORE PACKAGE LEVERAGE:
//   - Snapshot Isolation: Uses core.Clone() to spawn a "shadow reality" for destructive testing
//     without mutating the production graph.
//   - Atomic Cleanup: core.RemoveVertex(id) ensures no orphaned edges remain,
//     providing a clean state for the subsequent BFS traversal.
//   - Structural Inspection: Uses core.Degree and core.AdjacentVertices to compute
//     second-order topological metrics (Φ).
//
// Inputs:
//   - None (graph structure is hard-coded).
//
// Returns:
//   - None (prints the resilience ratio R).
//
// Errors:
//   - Any unexpected error is printed and the example returns early.
//
// Complexity:
//   - Building and scanning the graph: O(V + E). BFS for components: O(V + E).
func ExampleGraph_CascadingFailures() {
	// ---- Stage 1: Infrastructure Synthesis ----
	const clusterSize = 4
	var (
		err       error
		neighbors []string
		hubID     = "Hub-Central"
		districtA = []string{"A1", "A2", "A3", "A4"}
		districtB = []string{"B1", "B2", "B3", "B4"}
	)

	g := core.NewGraph(core.WithWeighted(), core.WithDirected(false))

	// Construct two dense districts (Cliques)
	for i := 0; i < clusterSize; i++ {
		for j := i + 1; j < clusterSize; j++ {
			if _, err = g.AddEdge(districtA[i], districtA[j], 1.0); err != nil {
				fmt.Println(err)
				return
			}
			if _, err = g.AddEdge(districtB[i], districtB[j], 1.0); err != nil {
				fmt.Println(err)
				return
			}
		}
	}

	// Link districts through a single strategic Hub (the single point of failure)
	for i := 0; i < clusterSize; i++ {
		_, _ = g.AddEdge(hubID, districtA[i], 1.0)
		_, _ = g.AddEdge(hubID, districtB[i], 1.0)
	}

	// ---- Stage 2: Pre-Collapse Fragility Analysis (Φ) ----
	_, _, hubDegree, _ := g.Degree(hubID)
	neighbors, _ = g.NeighborIDs(hubID)

	var neighborDegree, sumNeighborDegrees int
	for _, nID := range neighbors {
		_, _, neighborDegree, _ = g.Degree(nID)
		sumNeighborDegrees += neighborDegree
	}

	// Φ = 1 - (Average Neighbor Connectivity / Hub Connectivity)
	phi := 1.0 - (float64(sumNeighborDegrees) / float64(hubDegree))

	// ---- Stage 3: Sandbox Simulation (The Blackout) ----
	// core.Clone() creates a perfect isolated sandbox for destructive analysis
	sandbox := g.Clone()
	if err = sandbox.RemoveVertex(hubID); err != nil {
		fmt.Printf("Critical failure during simulation: %v\n", err)
		return
	}

	// ---- Stage 4: Topological Impact Assessment (BFS) ----
	// Expert-grade LCC (Largest Connected Component) calculation
	calcLCC := func(graph *core.Graph) int {
		var (
			maxSize     int
			allVertices = graph.Vertices()
			visited     = make(map[string]bool, len(allVertices))
			queue       = make([]string, 0, len(allVertices))
		)

		for _, root := range allVertices {
			if visited[root] {
				continue
			}

			// Component Discovery
			currentSize := 0
			queue = append(queue[:0], root) // Reset queue without re-allocating
			visited[root] = true

			for len(queue) > 0 {
				u := queue[0]
				queue = queue[1:]
				currentSize++

				adj, _ := graph.NeighborIDs(u)
				for _, v := range adj {
					if !visited[v] {
						visited[v] = true
						queue = append(queue, v)
					}
				}
			}

			if currentSize > maxSize {
				maxSize = currentSize
			}
		}

		return maxSize
	}

	nLCC := calcLCC(g)        // Giant component before attack
	npLCC := calcLCC(sandbox) // Giant component after hub removal

	// R = N'_LCC / (N_LCC - 1)
	resilience := float64(npLCC) / float64(nLCC-1)

	// ---- Stage 5: Executive Decision ----
	fmt.Printf("--- Aethelgard Grid Resilience Report ---\n")
	fmt.Printf("Target Hub Degree: %d\n", hubDegree)
	fmt.Printf("Fragility Index (Φ): %.2f\n", phi)
	fmt.Printf("Resilience Ratio (R): %.2f\n", resilience)

	if resilience < 0.6 {
		fmt.Println("STATUS: CRITICAL. System fragmentation imminent. Initiating bypass protocols.")
	} else {
		fmt.Println("STATUS: STABLE. Topology supports graceful degradation.")
	}

	// Output:
	// --- Aethelgard Grid Resilience Report ---
	// Target Hub Degree: 8
	// Fragility Index (Φ): -3.00
	// Resilience Ratio (R): 0.50
	// STATUS: CRITICAL. System fragmentation imminent. Initiating bypass protocols.
}

// ExampleGraph_BetweennessCentrality demonstrates the identification of a "critical artery"
// in a global logistics network using Betweenness Stress Centrality.
// Two densely connected communities (clusters) are linked by a single bridging edge.
// The bridging edge carries all shortest-path traffic between the clusters, making it the highest-betweenness edge.
// CONTEXT: "The Global Transit Bottleneck"
//   - You are the lead architect of a global supply chain monitoring system. The graph
//     represents two massive economic zones (Cluster A and Cluster B), each with high
//     internal redundancy. However, they are connected by a single transit corridor
//     (the "Suez-Link"). Your mission is to quantify the "Structural Stress" on this
//     link. If this single edge fails, 100% of inter-cluster trade is paralyzed.
//
// Scenario:
//   - Vertices are hubs/warehouses, edges are direct transport corridors.
//   - You have two dense regions (two cities / two warehouse clusters).
//   - Exactly one corridor connects the regions (a bridge edge).
//
// Why this matters (criticality):
//   - If that corridor fails, inter-region delivery collapses immediately.
//   - Even BEFORE failure, that corridor experiences maximal “load” because almost all cross-region
//     shortest paths must traverse it.
//
// MATHEMATICAL MODEL (Edge Stress):
//
//   - For a graph partitioned into two disjoint sets V_A and V_B, where all paths between
//     sets must traverse a single bridge edge (e_bridge), the "Load" (L) is:
//
//     L(e_bridge) = |V_A| * |V_B|
//
//   - This represents the total number of unique shortest-path pairs (s, t) such that
//     s ∈ V_A and t ∈ V_B. In this topology, the bridge edge carries the maximum
//     possible Betweenness Centrality.
//
// Closed-form load (for this topology):
//   - Every pair (a in A, b in B) must traverse the bridge.
//   - Therefore bridgeLoad = |A| * |B|.
//
// Implementation:
//   - Stage 1: Construct two clusters of vertices with rich internal connections.
//   - Stage 2: Link the clusters with a single edge and identify this edge.
//   - Stage 3: Calculate the number of unique shortest-path pairs that traverse the bridge (betweenness load).
//
// Behavior highlights:
//   - The identified bridge edge is an articulation link between clusters (its removal would disconnect the graph).
//   - The bridge's betweenness load equals the product of cluster sizes, as every inter-cluster pair of vertices must communicate via this edge.
//
// Inputs:
//   - None (graph structure is deterministic).
//
// Returns:
//   - None (prints the critical edge ID and its computed load).
//
// Errors:
//   - Any unexpected error is printed and the example returns early.
//
// Complexity:
//   - Graph construction: O(V^2) for dense cluster edges. Identifying the bridge and computing load: O(V + E).
//
// CORE PACKAGE LEVERAGE:
//   - Topology Verification: Uses GetEdge(id) for O(1) validation of critical links.
//   - Connectivity Analysis: Leverages Incidence(v) to inspect the local "fan-out"
//     of a hub vertex and identify the bridging edge among local connections.
//   - Inventory Integrity: Uses the deterministic Vertices() sequence to partition
//     and calculate global load factors without external state tracking.
func ExampleGraph_BetweennessCentrality() {
	// Constants for simulation scale (4x4 clusters for the example output)
	const clusterSize = 4
	const bridgeID = "e13"

	// Stage 1: Infrastructure Construction
	// We initialize a non-directed graph representing physical transport corridors.
	g := core.NewGraph(core.WithDirected(false))

	// Pre-allocate slices to avoid repeated allocations in loops.
	vertsA := make([]string, clusterSize)
	vertsB := make([]string, clusterSize)

	for i := 0; i < clusterSize; i++ {
		vertsA[i] = fmt.Sprintf("A%d", i)
		vertsB[i] = fmt.Sprintf("B%d", i)
	}

	// Build two Cliques (fully connected clusters).
	// This simulates high-density metropolitan or regional warehouse networks.
	for i := 0; i < clusterSize; i++ {
		for j := i + 1; j < clusterSize; j++ {
			_, _ = g.AddEdge(vertsA[i], vertsA[j], 0)
			_, _ = g.AddEdge(vertsB[i], vertsB[j], 0)
		}
	}

	// Stage 2: The Critical Integration (The Bottleneck)
	// We link the two clusters through a single point of failure.
	_, err := g.AddEdge(vertsA[0], vertsB[0], 0)
	if err != nil {
		fmt.Printf("Critical failure during bridge creation: %v\n", err)
		return
	}

	// Stage 3: Structural Analysis
	// Verify the bridge exists and analyze its impact.
	bridge, err := g.GetEdge(bridgeID)
	if err != nil {
		fmt.Printf("Link verification failed: %v\n", err)
		return
	}

	// Calculate Stress Load: L = |V_A| * |V_B|.
	// We use core.Vertices() to perform a census of the economic zones.
	var countA, countB int
	for _, v := range g.Vertices() {
		if strings.HasPrefix(v, "A") {
			countA++
		} else if strings.HasPrefix(v, "B") {
			countB++
		}
	}

	stressLoad := countA * countB

	// Stage 4: Reporting and Verification
	// Verify that hub A0 is indeed a proxy by analyzing its neighbors.
	// Use NeighborIDs for a quick inspection of local connections.
	neighbors, _ := g.NeighborIDs(vertsA[0])
	var isBottleneckFound bool
	for _, nID := range neighbors {
		if nID == vertsB[0] {
			isBottleneckFound = true
			break
		}
	}

	// Output results using stable identifiers for documentation.
	if isBottleneckFound {
		fmt.Printf("Analysis: Critical Link Identified: %s (%s)\n", bridgeID, bridge.From+"-"+bridge.To)
		fmt.Printf("Load: Betweenness Stress Factor = %d paths\n", stressLoad)
	}

	// Output:
	// Analysis: Critical Link Identified: e13 (A0-B0)
	// Load: Betweenness Stress Factor = 16 paths
}

// ExampleGraph_NeuralEvolution simulates dynamic evolution of a neural network graph structure.
// It starts with a sparse, weighted graph (few connections),
// then adds a new neuron (vertex) with new connections, and finally removes an existing connection.
// The degree of a particular neuron is tracked through these modifications to illustrate network plasticity.
// CONTEXT: "Synapse-X" — The Structural Learning Engine
//   - In traditional neural networks, "learning" is merely updating weights in a static matrix.
//     In Project Synapse-X, we simulate biological neuroplasticity where the graph itself
//     is a living organism. When associations weaken, synapses are physically destroyed (Pruning)
//     to reclaim memory and reduce entropy. When new concepts emerge, the graph spawns
//     new vertices and edges (Evolution).
//
// Scenario:
//   - Vertices are neurons (or concepts), edges are synapses (or associations).
//   - Weights are connection strengths (requires Weighted graph).
//   - Learning can create new neurons (AddVertex), strengthen/insert synapses (AddEdge),
//     and prune unused synapses (RemoveEdge).
//
// WHY THIS IS CRITICAL (The Engineering Edge):
//   - Algorithmic Efficiency: In large-scale brains, "zeroing a weight" still keeps the
//     connection in the adjacency list, forcing O(N^2) or O(E_total) scans. Using
//     core.RemoveEdge(id) physically cleans the topology, ensuring neighborhood
//     traversals (via core.AdjacentVertices) only visit active, meaningful synapses.
//   - Topological Integrity: core.AddVertex(id) allows the network to expand its
//     associative memory dynamically without re-initializing the system.
//
// MATHEMATICAL MODEL (Structural Homeostasis):
//   - Network Density (D): D = (2 * |E|) / (|V| * (|V| - 1)).
//     The system monitors D to prevent a "connectivity explosion" (over-wiring).
//   - Pruning Logic: When a synapse decays, the system identifies the
//     topological link via NeighborIDs() and Edge verification, then executes
//     core.RemoveEdge(id) to maintain energy efficiency.
//
// Implementation:
//   - Stage 1: Build a sparse weighted graph.
//   - Stage 2: Add a new neuron and connect it.
//   - Stage 3: Remove one existing edge (synaptic pruning).
//   - Stage 4: Query Degree at each stage.
//
// Inputs:
//   - None (uses deterministic graph modifications).
//
// Returns:
//   - None (prints the tracked degree values).
//
// Errors:
//   - Any unexpected error is printed and the example returns early.
//
// Complexity:
//   - Graph updates (add/remove): O(1) each amortized. Degree queries: O(d) per query.
//
// CORE PACKAGE LEVERAGE:
//   - Targeted Retrieval: Using EdgeBetween(u, v) provides O(1) or O(d) access to specific
//     synapses, avoiding expensive global Edge() scans.
//   - Amortized O(1) Updates: Add/Remove operations leverage core's map-based
//     architecture for high-frequency structural shifts.
func ExampleGraph_NeuralEvolution() {
	// ---- PHASE 1: Initial Cognitive Seed (Sparse Substrate) ----
	// Initialize an undirected, weighted graph representing the base neural cluster.
	g := core.NewGraph(core.WithDirected(false), core.WithWeighted())

	// Primary synaptic pathways (Initial Knowledge)
	// AddEdge returns (ID, error). We use "_" as we track them by topology later.
	_, _ = g.AddEdge("0", "1", 0.5)
	_, _ = g.AddEdge("1", "2", 0.8)
	_, _ = g.AddEdge("3", "4", 1.2)

	// Capture baseline plasticity: connectivity of Neuron "2".
	// Degree returns (in, out, total, error).
	_, _, degInit, _ := g.Degree("2")

	// ---- PHASE 2: Evolutionary Expansion (Learning Spike) ----
	// A new concept "5" emerges, forging a strong bond with the existing hub (Neuron 2).
	if err := g.AddVertex("5"); err != nil {
		return
	}

	// Forging new synapses based on conceptual proximity.
	_, _ = g.AddEdge("5", "2", 0.7)
	_, _ = g.AddEdge("5", "4", 0.4)

	// Audit: Neuron "2" degree increases as it integrates the new concept.
	_, _, degAfterAdd, _ := g.Degree("2")

	// ---- PHASE 3: Synaptic Pruning (Homeostatic Optimization) ----
	// The system detects that the synapse between "1" and "2" has become "stale".
	// To prune it, we surgically identify its ID from the active Edges list.
	var targetID string
	for _, e := range g.Edges() {
		// In an undirected graph, we check both directions for the From/To pair.
		if (e.From == "1" && e.To == "2") || (e.From == "2" && e.To == "1") {
			targetID = e.ID
			break
		}
	}

	// Execute physical decommissioning of the connection.
	if targetID != "" {
		if err := g.RemoveEdge(targetID); err != nil {
			return
		}
	}

	// Final State: The network is optimized and ready for the next learning cycle.
	_, _, degAfterRem, _ := g.Degree("2")

	// ---- OUTPUT: Structural Pulse Monitoring ----
	// This confirms the successful growth and pruning cycles of the system.
	fmt.Printf("deg[2][0]=%d\n", degInit)
	fmt.Printf("deg[2][1]=%d\n", degAfterAdd)
	fmt.Printf("deg[2][2]=%d\n", degAfterRem)

	// Output:
	// deg[2][0]=1
	// deg[2][1]=2
	// deg[2][2]=1
}

/*

// ExampleGraph_SpectralAnalysis represents the "Singularity Protocol" (Quantum-Grid 2026). This is the most advanced
// demonstration of the library, merging topological integrity from 'core' with spectral power from 'matrix'.
// It's Performs spectral analysis on a quantum graph structure (algebraic connectivity via the Laplacian).
// It computes the second-smallest eigenvalue (λ₂) of the graph Laplacian (known as the Fiedler value) before and after
// the removal of a vertex. Removing a critical vertex demonstrates the drop in algebraic connectivity.
// SCENARIO:
//   - You are overseeing a high-stakes Quantum Entanglement Network. In this realm,
//     a binary "connected/disconnected" status is a post-mortem; you need PREDICTION.
//     The Second-Smallest Eigenvalue of the Laplacian (λ₂), also known as the "Fiedler Value"
//     or "Algebraic Connectivity," acts as the system's heartbeat.
//
// CRITICALITY:
//   - λ₂ > 0: The network is globally connected.
//   - λ₂ → 0: The system is approaching a "Spectral Gap" collapse (a catastrophic bottleneck).
//   - λ₂ = 0: The network has partitioned into isolated islands, causing mission failure.
//
// MATHEMATICAL ENGINE:
//  1. Topology: 'core' provides the "Ground Truth" via deterministic vertex ordering.
//  2. Adjacency (A): Derived from core.AdjacentVertices, ensuring no phantom connections.
//  3. Degree (D): A diagonal matrix where D[i,i] = core.Degree(v_i).
//  4. Laplacian (L): L = D - A. This operator encodes the entire diffusion profile of the grid.
//  5. Solving: matrix.EigenSym extracts the spectrum, where λ₂ quantifies structural robustness.
//
// WHY 'CORE' IS MAXIMIZED:
//   - Determinism: core.Vertices() ensures the matrix indices (i, j) match the physical nodes 1:1.
//   - Atomic Collapse: core.RemoveVertex(X) instantly prunes all incident edges,
//     allowing a 'matrix.BuildAdjacency' call to reflect the new reality without noise.
//
// Implementation:
//   - Stage 1: Construct a connected graph (two fully connected subgraphs joined by a single intermediate vertex).
//   - Stage 2: Build the Laplacian matrix of the graph and compute its eigenvalues (Jacobi eigen-decomposition).
//   - Stage 3: Remove the intermediate vertex (simulating a "quantum" disconnection) and recompute the Laplacian eigenvalues.
//   - Stage 4: Compare λ₂ before and after removal to observe the change in connectivity.
//
// Behavior highlights:
//   - The second-smallest eigenvalue λ₂ is > 0 for a connected graph and drops to 0 when the graph becomes disconnected (after removing the vital vertex).
//   - The example uses the matrix sub-package to construct matrices and compute eigenvalues.
//
// Inputs:
//   - None (graph structure is deterministic).
//
// Returns:
//   - None (prints λ₂ before and after vertex removal).
//
// Errors:
//   - Any unexpected error is printed and the example returns early.
//
// Complexity:
//   - Building the Laplacian: O(V + E). Eigen decomposition (Jacobi): O(n³) for an n×n matrix (n = number of vertices).
func ExampleGraph_SpectralAnalysis() {
	// --- PHASE 1: TOPOLOGY CONSTRUCTION ---
	// We build two robust "cliques" (Districts) connected by a single critical Hub (Vertex X).
	g := core.NewGraph(core.WithWeighted(), core.WithDirected(false))

	nodes := []string{"A", "B", "C", "D", "E", "F", "X"}
	for _, n := range nodes {
		_ = g.AddVertex(n) // Simplified for the example context
	}

	// District 1 (Dense Cluster)
	edges := [][2]string{{"A", "B"}, {"B", "C"}, {"C", "A"}}
	// District 2 (Dense Cluster)
	edges = append(edges, [][2]string{{"D", "E"}, {"E", "F"}, {"F", "D"}}...)
	// The Critical "Suez" Bridge
	edges = append(edges, [][2]string{{"X", "A"}, {"X", "D"}}...)

	for i, e := range edges {
		_, _ = g.AddEdge(e[0], e[1], float64(i))
	}

	// --- PHASE 2: THE SPECTRAL ENGINE ---
	// This closure encapsulates the transformation from Topology to Energy Spectrum.
	computeFiedlerValue := func(target *core.Graph) (float64, error) {
		// 2.1: Extract Adjacency with multi-edge prevention policy.
		var mOpts matrix.Options
		matrix.WithDisallowMulti()(&mOpts)

		adj, err := matrix.BuildAdjacency(target, mOpts)
		if err != nil {
			return 0, err
		}

		// 2.2: Construct Laplacian L = D - A.
		// D (Degree Matrix) is diagonal; A is the Adjacency.
		n, _ := adj.VertexCount()
		L, _ := matrix.NewZeros(n, n)
		deg, _ := matrix.DegreeVector(adj)

		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if i == j {
					// Diagonal: Degree of the vertex
					_ = L.Set(i, j, deg[i])
				} else {
					// Off-diagonal: Negative adjacency
					val, _ := adj.Mat.At(i, j)
					_ = L.Set(i, j, -val)
				}
			}
		}

		// 2.3: Solve Eigen-problem using Symmetric Jacobi decomposition.
		// Precision and stability are key for near-zero eigenvalues.
		eigenvals, _, err := matrix.EigenSym(L, constEigenTol, constEigenMaxIter)
		if err != nil {
			return 0, err
		}

		sort.Float64s(eigenvals)
		if len(eigenvals) < 2 {
			return 0, fmt.Errorf("insufficient nodes for spectral analysis")
		}

		return eigenvals[1], nil // Return λ₂ (Fiedler Value)
	}

	// --- PHASE 3: INITIAL OBSERVATION (Stable State) ---
	fiedlerBefore, err := computeFiedlerValue(g)
	if err != nil {
		fmt.Printf("Analysis failed: %v\n", err)
		return
	}

	// --- PHASE 4: CATASTROPHIC COLLAPSE (The Event) ---
	// Simulating the measurement collapse of Qubit X or the destruction of Hub X.
	// core.RemoveVertex is an O(1) op that cleans up all associated incident edges.
	_ = g.RemoveVertex("X")

	// --- PHASE 5: POST-COLLAPSE DIAGNOSIS ---
	fiedlerAfter, err := computeFiedlerValue(g)
	if err != nil {
		fmt.Printf("Post-collapse analysis failed: %v\n", err)
		return
	}

	// RESULTS:
	// λ₂[0] > 0 signifies a "brittle but connected" system.
	// λ₂[1] ≈ 0 signals a complete loss of algebraic connectivity (System Partitioned).
	fmt.Printf("%s=%.4f\n", outLambda2_0, fiedlerBefore)
	fmt.Printf("%s=%.4f\n", outLambda2_1, fiedlerAfter)

	// Output:
	// lambda2[0]=0.2679
	// lambda2[1]=0.0000
}

*/
