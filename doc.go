// Package lvlath is your in-memory playground for building, exploring,
// and analyzing graphs — from core primitives to advanced flow, TSP and
// time-series algorithms.
//
// 🚀 What is lvlath/graph?
//
//	A modern, thread-safe, zero-dependency library that brings together:
//		• Core primitives: create vertices & edges, mutate safely under locks
//		• Matrix views: adjacency & incidence matrices + converters
//		• Traversals: BFS, DFS
//		• Shortest paths: Dijkstra
//		• Minimum spanning trees: Prim, Kruskal
//		• Flow algorithms: Ford–Fulkerson, Edmonds–Karp, Dinic
//		• TSP solvers: Held–Karp (exact), Christofides (approx)
//		• Time-series: Dynamic Time Warping (DTW)
//
// ✨ Why choose lvlath?
//
//   - Beginner-friendly – minimal API, clear, intuitive naming
//   - Rock-solid guarantees – R/W locks, in-code docs & hooks
//   - Pure Go – no cgo, no hidden deps
//   - Extensible – add custom hooks (OnVisit, OnEnqueue…) for custom logic
//
// Under the hood, everything is organized under three subpackages:
//
//	algorithms/ — traversal (BFS/DFS), shortest path (Dijkstra) & MST (Prim/Kruskal)
//	converters/ —
//	core/       — fundamental Graph, Vertex, Edge types & thread-safe primitives
//	dtw/ —
//	flow/ —
//	gridgraph/ —
//	matrix/     — adjacency & incidence matrix representations + converters
//	tsp/ —
//
// Quick ASCII example:
//
//	    A───B
//	    │   │
//	    C───D
//
//	represents a square with four vertices and four edges.
//
// Next up: GridGraph, external converters, probabilistic models and beyond.
// Dive into README.md for full examples, a feature matrix, and our roadmap
// to parallelism, flow algorithms and beyond.
//
//	go get github.com/katalvlaran/lvlath/graph
package lvlath
