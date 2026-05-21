<!--
  lvlath - Repository Documentation

  Purpose:
    This document is the repository-level specification, tutorial, and contract map
    for lvlath/dfs. It explains what the package guarantees, how the mathematics maps
    to the implementation, where determinism comes from, what partial results mean,
    and how to use DFS, Cycle Detection, and Topological Sorting correctly in
    production pipelines.

  Contract status:
    - Behaviors described here are part of the public contract.
    - Determinism rules described here are part of the public contract.
    - Error-classification rules described here are part of the public contract.
    - Result semantics for DFSResult and CycleDetectionResult are part of the public contract.
    - Any incompatible change must be explicit, documented, and versioned.

  Scope:
    - Deterministic Depth-First Search traversal and DFS forests.
    - Cycle witness detection and minimal-rotation canonicalization.
    - Directed acyclic graph (DAG) topological sorting.
    - Partial-result semantics and context cancellation.

  License:
    The lvlath repository is licensed under AGPL-3.0-only. See LICENSE.
-->

# 4. DFS (Depth-First Search)

> **Package:** `lvlath/dfs` | **Focus:** Determinism, Post-Order Semantics, Witness Cycles, Directed Topological Ordering

The `lvlath/dfs` package provides a deterministic, contract-driven implementation of Depth-First Search over `core.Graph`. It acts as the algorithmic backbone for workflows requiring deep path exploration, cycle detection, and dependency ordering.

Unlike casual textbook recursion that silently breaks under edge cases or relies on unstable map iteration, `lvlath/dfs` is engineered for high-load, mathematically exact, and debuggable graph analytics.

---

## 4.1. What & Why

Why use `lvlath/dfs` instead of a handwritten recursive loop?

### The Four Laws of DFS

1. **Absolute Determinism**
    *   *Others:* Yield different traversal paths on identical graphs depending on map hashing, unstable adjacency containers, or accidental traversal tie-breaks.
    *   *Lvlath:* Expands candidate relations in the exact order returned by `core.Graph.Neighbors(u)`. The package does not inject a hidden fallback sort during traversal. Cycle canonicalization yields byte-for-byte stable witness signatures, so the same graph produces the same traversal and witness output on every run.
2. **Post-Order Semantics by Contract**
    *   *Others:* Blur the distinction between discovery, entry, and completion.
    *   *Lvlath:* `DFSResult.Order` is strictly the *finish order* (post-order). Pre-order visibility is securely delegated to explicit `OnVisit` hooks.
3. **Witness Sets over Theorem Provers**
    *   *Others:* Attempt NP-hard exhaustive simple-cycle enumeration, leading to catastrophic exponential memory/time blowups in dense graphs.
    *   *Lvlath:* Cycle detection returns a deterministic *witness set* of canonical cycles. It proves cyclicity and locates logical loops without halting your production system.
4. **Strict Safety & Fail-Fast Policies**
    *   *Others:* Fail mid-traversal or return corrupted arrays on context timeout.
    *   *Lvlath:* Options are validated *before* state is allocated. If execution aborts, partial metadata (`Depth`, `Visited`, `Parent`) is safely retained, but the incomplete `Order` is explicitly cleared to prevent mathematical falsehoods downstream.

---

## 4.2. Mathematical Formulation

The `lvlath/dfs` package models strict graph-theoretic properties. Distances are hop-counts within the constructed DFS tree, and directed paths enforce strict edge orientation.

### 4.2.1. DFS-Tree Depth
For every visited vertex $v$ that is not a root of a DFS tree, its depth is exactly one greater than the vertex from which it was first entered:
$$ \forall v \in V \setminus \{Roots\}, \quad Depth[v] = Depth[Parent[v]] + 1 $$

### 4.2.2. Finish-Order Semantics
Let $entry(v)$ and $finish(v)$ be the timestamps of entering and leaving $v$. A vertex $v$ is appended to `Order` exactly at $finish(v)$. Thus, if $u$ is an ancestor of $v$ in the DFS tree, the invariant holds:
$$ finish(v) < finish(u) $$
This means descendants always appear *before* their ancestors in the `Order` slice.

### 4.2.3. Forest Semantics under FullTraversal
With `WithFullTraversal()`, the graph is partitioned into a DFS forest. For each new tree root $r$ selected by the deterministic graph iteration:
$$ \forall r \in Roots, \quad Depth[r] = 0 \quad \text{and} \quad r \notin keys(Parent) $$

### 4.2.4. Cycle Witness Formulation
A cycle is extracted from the active path stack when a `Gray` $\to$ `Gray` back-edge is encountered. The witness is returned as a closed sequence:
$$ W =[v_0, v_1, ..., v_k, v_0] $$

### 4.2.5. Canonicalization: Directed vs Undirected
To deduplicate cycles starting at different vertices but representing the same loop, `lvlath` uses Booth's minimal rotation algorithm.
$$ C_{canon} = \begin{cases} \min_{lex}(Rotations(C)) & \text{if } G \text{ is directed} \\ \min_{lex}\Big(\min_{lex}(Rotations(C)), \min_{lex}(Rotations(Reverse(C)))\Big) & \text{if } G \text{ is undirected} \end{cases} $$
*Directed cycles strictly preserve their orientation. Undirected cycles consider reverse flow to be equivalent.*

### 4.2.6. Topological Order Invariant
For a directed acyclic graph (DAG), the final topological order guarantees that every directed edge points forward:
$$ \forall (u \to v) \in E_{directed}, \quad \text{Index}(u) < \text{Index}(v) \text{ in Output} $$

### 4.2.7. Complexity Summary
*   **DFS Traversal / DFSForest:** $T = O(V + E)$, $S = O(V)$, excluding hook and filter cost.
*   **Topological Sort:** $T = O(V + E)$, $S = O(V)$.
*   **Cycle Detection:** $T = O(V + E + \sum |witness_i|)$, $S = O(V + L_{max} + \sum |witness_i|)$.
---

## 4.3. Public API & Result Contract

### 4.3.1. Public Entry Points

```go
func DFS(g *core.Graph, startID string, opts ...Option) (*DFSResult, error)
func DFSForest(g *core.Graph, opts ...Option) (*DFSResult, error)

func DetectCycles(g *core.Graph) (*CycleDetectionResult, error)
func HasCycle(g *core.Graph) (bool, error)

func TopologicalSort(g *core.Graph, options ...TopoOption) ([]string, error)
func TopologicalSortContext(ctx context.Context, g *core.Graph) ([]string, error)
```

### 4.3.2. DFSResult Semantics

```go
type DFSResult struct {
	Order            []string
	Depth            map[string]int
	Parent           map[string]string
	Visited          map[string]bool
	SkippedNeighbors int
}
```

| Field              | Meaning                                     | Safe on Partial Result? | Key Invariant                                  |
|:-------------------|:--------------------------------------------|:-----------------------:|:-----------------------------------------------|
| `Order`            | Finish order (post-order).                  |   **NO (Set to nil)**   | A partial post-order is mathematically unsafe. |
| `Depth`            | DFS-tree depth from the root.               |           YES           | Records depth upon entry.                      |
| `Parent`           | The predecessor that discovered the vertex. |           YES           | DFS-tree roots do not appear in this map.      |
| `Visited`          | Vertices actually entered.                  |           YES           | Used to track coverage.                        |
| `SkippedNeighbors` | Count of explicitly filtered neighbors.     |           YES           | Diagnostic for policy boundaries.              |

> [!IMPORTANT]
> **Partial-Result Contract:** If traversal aborts due to `context.Canceled` or a hook error, `DFSResult` is returned alongside the error. The `Order` field is cleared (`nil`), but `Visited`, `Depth`, and `Parent` retain the structural progress up to the exact point of failure.

### 4.3.3. CycleDetectionResult

```go
type CycleDetectionResult struct {
	HasCycle bool
	Cycles   [][]string
}
```
*   `HasCycle`: the summary boolean exposed directly in the result object.
*   `HasCycle(g)`: a convenience facade for callers that need only the boolean classification and not the witness payload.
*   `Cycles`: The deterministic, canonicalized witness list. Note that `len(Cycles) == 0` when `HasCycle == false`.

### 4.3.4. Error Protocol and Validation Priority
The `DFS` facade evaluates preconditions in a strict, predictable order:
1. `g == nil` $\to$ `ErrGraphNil`
2. Explicit invalid options $\to$ `ErrOptionViolation`
3. Single-source start vertex absent $\to$ `ErrStartVertexNotFound`
4. Runtime failures (Context, Hooks, Neighbor fetch).

### 4.3.5. Result Ownership

Returned result slices and maps belong to the caller after the function returns.

This has three practical consequences:
*   the package does not retain them for later mutation,
*   callers may safely copy, transform, or serialize them after return,
*   the package remains an algorithm layer rather than a topology-storage or builder layer.
---

## 4.4. Options & Policy Surface

Options implement a last-writer-wins functional assembly. Invalid explicit options (e.g., `WithContext(nil)`) fail-fast before allocating graph traversal state.

### 4.4.1. DFS Options
*   `WithContext(ctx)`: Ties the DFS recursion to standard cancellation rules.
*   `WithOnVisit(fn)`: Executes *immediately* when a vertex is entered (pre-order). Returning an error halts traversal.
*   `WithOnExit(fn)`: Executes right before a vertex is marked Black and pushed to `Order`.
*   `WithMaxDepth(limit)`: The horizon cut. Enforced *before* entering deeper vertices. Use `NoDepthLimit (-1)` for unrestricted.
*   `WithFilterNeighbor(fn)`: A policy firewall. Returning `false` blocks traversal into the candidate, treating the edge as non-existent for the current traversal.
*   `WithFullTraversal()`: Transforms single-source search into a graph-wide DFS forest. Used implicitly by `DFSForest()`.

### 4.4.2. Topological-Sort Options
*   `WithCancelContext(ctx)`: Topo-sort is specialized and uses a narrower option surface `TopoOption`.

---

## 4.5. Algorithmic Architecture

### 4.5.1. DFS Runtime Model (`dfsWalker`)
The `dfsWalker` maintains the traversal state strictly isolated from the option structs. It parses per-edge semantics using `neighborFromEdge`-this means in mixed-edge graphs, directed edges flow one way, while undirected edges behave bidirectionally, interpreted locally per-edge.

### 4.5.2. Cycle Detector Runtime Model (`cycleDetector`)
It employs the classic 3-color vertex state:
*   `White (0)`: Unvisited.
*   `Gray (1)`: Discovered, currently in the active DFS recursion path.
*   `Black (2)`: Fully explored and exited.
    A `Gray` $\to$ `Gray` edge triggers cycle reconstruction by scanning the active path stack, followed by $O(L)$ Booth's canonicalization.

### 4.5.3. Topological Sorter Runtime Model (`topoSorter`)
Valid only for directed graphs (`g.Directed() == true`). It leverages the 3-color model to simultaneously detect directed cycles (`ErrCycleDetected`) and record post-order. Once all vertices are `Black`, it performs an $O(V)$ in-place array reversal to yield the exact topological execution pipeline. Mixed-edge environments are handled strictly: undirected edges are bypassed entirely.

---

## 4.6. Pseudocode

### 4.6.1. DFS Traversal Kernel
```text
FUNCTION Traverse(u, depth):
  if ctx.Done():
    clear authoritative finish order
    return ctx.Err()

  if MaxDepth is bounded and depth > MaxDepth:
    return OK

  Visited[u] = true
  Depth[u] = depth

  if OnVisit is defined:
    OnVisit(u)

  neighbors = g.Neighbors(u)

  for each edge in neighbors:
    v, ok = ResolveNeighbor(edge, u)
    if not ok:
      continue

    if v == u and loops are disallowed by graph policy:
      continue

    if FilterNeighbor is defined and FilterNeighbor(v) == false:
      SkippedNeighbors++
      continue

    if Visited[v]:
      continue

    Parent[v] = u
    Traverse(v, depth + 1)

  if OnExit is defined:
    OnExit(u)

  append u to Order
```

### 4.6.2. DetectCycles Witness Extraction
```text
FUNCTION VisitCycle(u):
  State[u] = Gray
  push u to ActivePath

  for each edge in Neighbors(u):
    v = neighborFromEdge(edge, u)
    if Undirected Parent Backtrack: continue

    if State[v] == White:
       VisitCycle(v)
    else if State[v] == Gray:
       // Witness Detected!
       cycleBase = ActivePath[indexOf(v) ... end]
       canon = Canonicalize(cycleBase)
       SaveWitness(canon)

  pop u from ActivePath
  State[u] = Black
```

---

## 4.7. ASCII Diagrams

### 4.7.1. Recursion and Finish Order (Post-Order)
The most common mistake is expecting `Order` to track discovery. It strictly tracks *completion*.
```text
       [A]
      /   \
    [B]   [C]
   /   \
 [D]   [E]

Discovery (Pre-Order via OnVisit): A -> B -> D -> E -> C
Finish Order (Returned DFS Order): D -> E -> B -> C -> A
```

### 4.7.2. Forest Traversal (WithFullTraversal)
A full traversal iterates through the graph's deterministic vertex catalog, spinning up new trees for unvisited disconnected components.
```text
Component 1:   [A] -> [B]       Depth: A=0, B=1
Component 2:   [X]              Depth: X=0
Component 3:   [M] -> [N]       Depth: M=0, N=1

Roots: A, X, M (Absent from Parent map)
```

### 4.7.3. MaxDepth Horizon Limit
A hard firewall enforcing bounding radius without mutating the graph.
```text
MaxDepth = 1

    [S] (depth 0)
   /   \
 [A]   [B] (depth 1 - Visited, appended to Order)
  |     |
 [X]   [Y] (depth 2 - Ignored completely, White)
```

### 4.7.4. Cycle Extraction (Gray Back-Edge)
Extracting the loop from the active stack.
```text
Active Stack: [ S -> A -> B -> C ]
Current Node: C
Neighbors(C) sees 'A', which is GRAY (already in stack).

Extraction: Suffix from A to C:[A, B, C]
Closed Output: [A, B, C, A]
```

### 4.7.5. Topological Reversal
Topological sorting reverses the finish order exactly once at the very end.
```text
DAG:  [Build] -> [Test] -> [Deploy]

Finish Order (Black): [Deploy] -> [Test] -> [Build]
Reversal (O(V)):      [Build]  -> [Test] -> [Deploy]
```

---

## 4.8. Go Example Scenarios

The examples below demonstrate verified, production-grade usage of `lvlath/dfs`.

### 4.8.1. Infrastructure Inspection
Demonstrates single-source traversal, `WithFilterNeighbor` to simulate a quarantine boundary, and hook-based audit tracing.

```go
func ExampleDFS_infrastructureInspection() {
	graph, _ := core.NewGraph(core.WithDirected(true))

	_, _ = graph.AddEdge("gateway", "auth", 0)
	_, _ = graph.AddEdge("gateway", "billing", 0)
	_, _ = graph.AddEdge("auth", "profile", 0)
	_, _ = graph.AddEdge("edge", "quarantine-lab", 0)
	_, _ = graph.AddEdge("edge", "traffic", 0)
	// (See example_test.go for full graph definition)

	entered := make([]string, 0, graph.VertexCount())

	// Policy Firewall: Do not traverse quarantine boundaries
	filter := func(vertexID string) bool {
		return !strings.HasPrefix(vertexID, "quarantine-")
	}

	onVisit := func(vertexID string) error {
		entered = append(entered, vertexID)
		return nil
	}

	result, err := dfs.DFS(
		graph,
		"gateway",
		dfs.WithFilterNeighbor(filter),
		dfs.WithOnVisit(onVisit),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("skipped:", result.SkippedNeighbors)
	fmt.Println("postorder:", strings.Join(result.Order, " -> "))
}
```

### 4.8.2. Full Traversal Inventory Sweep
Demonstrates `DFSForest` picking up disconnected islands and managing multiple roots.

```go
func ExampleDFS_fullTraversalInventorySweep() {
	graph, _ := core.NewGraph(core.WithDirected(true))
	// 3 Isolated islands: zone-a, zone-m, zone-z
	_, _ = graph.AddEdge("zone-a:0-gw", "zone-a:1-api", 0)
	_, _ = graph.AddEdge("zone-m:0-gw", "zone-m:1-batch", 0)
	_, _ = graph.AddEdge("zone-z:0-gw", "zone-z:1-web", 0)

	result, err := dfs.DFSForest(graph)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	roots :=[]string{}
	for vertexID := range result.Visited {
		if _, hasParent := result.Parent[vertexID]; !hasParent {
			roots = append(roots, vertexID) // No parent = Root
		}
	}

	// DFSForest resets depth at each root
	fmt.Printf("zone-a:0-gw depth = %d\n", result.Depth["zone-a:0-gw"])
	fmt.Printf("zone-m:0-gw depth = %d\n", result.Depth["zone-m:0-gw"])
}
```

### 4.8.3. Depth-Limited Blast Radius
Simulates a bounded incident impact analysis.

```go
func ExampleDFS_depthLimitedBlastRadius() {
	graph, _ := core.NewGraph(core.WithDirected(true))
	_, _ = graph.AddEdge("incident", "api", 0)
	_, _ = graph.AddEdge("api", "auth", 0)
	_, _ = graph.AddEdge("auth", "db", 0) // Depth 3

	// Limit to depth 2: "db" will not be visited
	result, err := dfs.DFS(graph, "incident", dfs.WithMaxDepth(2))
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("outer-ring-entered (db):", result.Visited["db"]) // Output: false
}
```

### 4.8.4. Release Pipeline
Validates deterministic topological sort execution plans for DAGs.

```go
func ExampleTopologicalSort_releasePipeline() {
	graph, _ := core.NewGraph(core.WithDirected(true))
	_, _ = graph.AddEdge("01-spec-freeze", "02-schema-lock", 0)
	_, _ = graph.AddEdge("02-schema-lock", "03-codegen", 0)
	_, _ = graph.AddEdge("03-codegen", "04-unit-tests", 0)

	order, err := dfs.TopologicalSort(graph)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("release-plan:", strings.Join(order, " -> "))
	// Output: 01-spec-freeze -> 02-schema-lock -> 03-codegen -> 04-unit-tests
}
```

### 4.8.5. Escalation Loop Witness
Demonstrates `DetectCycles` retrieving canonical closed-loop representations.

```go
func ExampleDetectCycles_escalationLoopWitness() {
	graph, _ := core.NewGraph(core.WithDirected(true))
	_, _ = graph.AddEdge("triage", "security", 0)
	_, _ = graph.AddEdge("security", "approvals", 0)
	_, _ = graph.AddEdge("approvals", "triage", 0) // Loop back

	result, err := dfs.DetectCycles(graph)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("hasCycle:", result.HasCycle)
	fmt.Println("witness:", strings.Join(result.Cycles[0], " -> "))
	// Output: approvals -> triage -> security -> approvals
}
```

---

## 4.9. Pitfalls & Best Practices (Architectural Mastery)

To use `lvlath/dfs` correctly in production pipelines, treat its traversal artifacts and witness contracts as first-class mathematical outputs rather than incidental implementation leftovers.

> [!CAUTION]
> **1. Discovery vs. Finish Order**
> `DFSResult.Order` strictly records the **Post-Order** (when vertices *finish* traversal).
> *   **The Pitfall:** Do not confuse this with entry/discovery order.
> *   **The Solution:** If you need to track the discovery sequence, use the `WithOnVisit` hook to append vertices to your own slice.

> [!CAUTION]
> **2. Results on Aborting Failure**
> When a traversal is aborted (e.g., via `context.Cancellation`), the `DFSResult.Order` is set to `nil`.
> *   **Why:** A partial post-order is mathematically meaningless for most algorithms.
> *   **The Practice:** Read `Visited`, `Depth`, or `Parent` instead to analyze the state of the graph reached before the crash.

> [!IMPORTANT]
> **3. Traversal Policy vs. Topology Mutation**
> Never mutate the underlying graph structure just to simulate a runtime boundary or "quarantine" a node.
> *   **The Way:** Use the `FilterNeighbor` option to enforce traversal policies. This keeps your core topology stable and thread-safe while allowing dynamic branch pruning.

> [!IMPORTANT]
> **4. Error Handling: Sentinels over Strings**
> Never parse error strings. `lvlath/dfs` wraps user hooks and timeouts using `%w`, keeping the chain intact.
> *   **The Rule:** Always use `errors.Is(err, dfs.ErrCycleDetected)` or `errors.Is(err, dfs.ErrOptionViolation)` for logic branching.

> [!WARNING]
> **5. Witness Discovery vs. Exhaustive Sets**
> `DetectCycles` is an optimized tool for proving cyclicity, not a search engine for all possible paths.
> *   **The Contract:** It returns a deterministic **witness set** to prove a loop exists. It does not promise a mathematically exhaustive catalog of all simple cycles (which is an NP-hard task).

> [!CAUTION]
> **6. The Directed-Graph Gate in Topological Sorting**
> Topological sorting is defined only for directed graphs.
> *   **The Gate:** If the graph fails the package’s directed-policy check, `TopologicalSort` returns `ErrGraphNotDirected`.
> *   **The Constraint:** Only directed outgoing relations contribute to the ordering. Undirected relations are ignored in the final topological plan.

> [!IMPORTANT]
> **7. Stability & Concurrency**
> The package reads the graph progressively and does not take a global immutable snapshot.
> *   **The Requirement:** Do not mutate graph topology concurrently during traversal. For reproducible results, the graph must remain topology-stable during execution.

> [!NOTE]
> **8. Ownership & Determinism**
> *   **Ownership:** Once returned, the maps and slices inside `DFSResult` belong entirely to the caller. The package does not retain or mutate them.
> *   **Determinism:** Treat it as a **contract**. If the output order changes for the same graph, it indicates a change in graph ordering or a regression in the implementation.

---

## 4.10. Practical Recipes

*   **Blast Radius (Impact Analysis):** Run `DFS(start, WithMaxDepth(k))` to bound the analysis area around an incident node without scanning the entire topology.
*   **Deterministic Crawler:** Use `WithContext(ctx)` and trigger `cancel()` from within an `OnVisit` hook once a domain condition is met. Process the partial `Visited` map.
*   **Release Planning:** Model CI/CD steps as a directed graph. Execute `TopologicalSort`. If `ErrCycleDetected` is returned, halt the pipeline to prevent circular deadlocks.
*   **Cycle Auditing:** Use `DetectCycles` on resource-allocation graphs. The `Cycles` output provides the exact sequence of locks forming a deadlock.
*   **Performance Hooks:** Keep `OnVisit` and `OnExit` hooks lightweight and side-effect aware, as they execute directly in the hot traversal paths.

---
**lvlath/dfs**: deterministic by contract, strict in witness semantics, and safe to compose.

> Next: [5. Shortest Paths: Dijkstra ->](DIJKSTRA.md)
