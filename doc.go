// Package graph (lvlath) is your in-memory playground for building,
// exploring, and analyzing graphs in Go.
//
// 🚀 What is lvlath/graph?
//
//	A modern, thread-safe, zero-dependency library that brings together:
//
//	  • Core primitives: create vertices & edges, mutate safely under locks
//	  • Matrix views: adjacency & incidence matrices + converters
//	  • Classic algorithms: BFS, DFS, Dijkstra, Prim & Kruskal
//
// ✨ Why choose lvlath?
//
//   - Beginner-friendly    — minimal API, clear, intuitive naming
//   - Rock-solid           — built-in R/W locks ensure thread-safety
//   - Extensible           — attach OnVisit/OnEnqueue hooks for custom logic
//   - Pure Go              — no cgo, no hidden dependencies
//
// Under the hood, everything is organized under three subpackages:
//
//	core/       — fundamental Graph, Vertex, Edge types & thread-safe primitives
//	matrix/     — adjacency & incidence matrix representations + converters
//	algorithms/ — traversal (BFS/DFS), shortest path (Dijkstra) & MST (Prim/Kruskal)
//
// Quick ASCII example:
//
//	    A───B
//	    │   │
//	    C───D
//
//	represents a square with four vertices and four edges.
//
// Dive into README.md for full examples, a feature matrix, and our roadmap
// to parallelism, flow algorithms and beyond.
//
//	go get github.com/katalvlaran/lvlath/graph
package graph
