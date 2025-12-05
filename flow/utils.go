package flow

import (
	"context"
	"fmt"

	"github.com/katalvlaran/lvlath/core"
)

// buildCapMap constructs a nested map representing the residual capacities
// of graph `g`, aggregating parallel edges and ignoring loops.
//
// The returned capMap has structure: capMap[u][v] = total integer capacity
// from u → v after summing all parallel edges in `g` and discarding capacities ≤ Epsilon.
//
// Steps:
//  1. Normalize opts.Ctx and opts.Epsilon (via opts.normalize() in caller).
//  2. Initialize capMap with one inner map per vertex (O(V)).
//  3. For each vertex u in sorted order (O(V)):
//     a. Check ctx.Err() for early cancellation.
//     b. For each outgoing *Edge e := g.Neighbors(u) (O(deg(u)*log deg(u))):
//     If e.From == e.To (self-loop), skip immediately.
//     c := float64(e.Weight).
//     If c < -opts.Epsilon, return EdgeError (negative capacity).
//     capMap[u][e.To] += e.Weight.
//     c. After gathering, remove any capMap[u][v] where float64(cap) ≤ opts.Epsilon.
//
// Complexity:
//
//	Time:   O(V + E*log d_max) where d_max is max degree (for sorting neighbors).
//	Memory: O(V + E) for storing all capacities in capMap.
func buildCapMap(g *core.Graph, opts FlowOptions) (map[string]map[string]float64, error) {
	// Prepare context: default to Background if nil
	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	// Early exit if context already canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Initialize capMap: outer map sized to number of vertices
	vertices := g.Vertices()
	capMap := make(map[string]map[string]float64, len(vertices))
	for _, u := range vertices {
		// Create inner map for each vertex
		capMap[u] = make(map[string]float64)
	}

	// Iterate through each vertex u in sorted order
	for _, u := range vertices {
		// Check for cancellation before heavy work
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Retrieve neighbors (edges) for vertex u
		neighbors, err := g.Neighbors(u)
		if err != nil {
			return nil, err
		}

		// Sum up capacities for each target v by aggregating parallel edges
		for _, e := range neighbors {
			// If this is a self-loop (u == v), ignore it
			if e.From == e.To {
				continue
			}
			// Convert weight to float64 for Epsilon comparison
			c := float64(e.Weight)
			// If capacity is below negative Epsilon, return an EdgeError
			if c < -opts.Epsilon {
				return nil, fmt.Errorf("flow: negative capacity on edge %q→%q: %g", e.From, e.To, c)
			}
			// Aggregate the integer weight
			capMap[u][e.To] += e.Weight
		}

		// Now filter out any entries with total capacity ≤ Epsilon
		for v, totalInt := range capMap[u] {
			if float64(totalInt) <= opts.Epsilon {
				delete(capMap[u], v)
			}
		}
	}

	return capMap, nil
}

// buildCoreResidualFromCapMap constructs a new *core.Graph (residual graph)
// based on capMap and inherits all configuration flags from original `g`:
// directedness, weighted, multi-edge, loops, and mixed-edge settings.
//
// Steps:
//  1. CloneEmpty() copies vertices (and Metadata) + graph flags (O(V)).
//  2. For each u in capMap (O(V)), and for each (v, cap) in capMap[u]:
//     a. If float64(cap) > opts.Epsilon, add an edge u→v with weight cap.
//  3. Return the constructed residual graph.
//
// Complexity:
//
//	Time:   O(V + E_res) where E_res is number of residual edges after filtering.
//	Memory: O(V + E_res).
func buildCoreResidualFromCapMap(
	capMap map[string]map[string]float64,
	g *core.Graph,
	opts FlowOptions,
) (*core.Graph, error) {
	// CloneEmpty copies vertices (ID + Metadata) and preserves configuration flags:
	// Directed, Weighted, MultiEdges, Loops, MixedEdges.
	residual := g.CloneEmpty()

	// Iterate through capMap to add edges
	for u, inner := range capMap {
		for v, capUV := range inner {
			// Only add edges with capacity strictly greater than Epsilon
			if capUV > opts.Epsilon {
				// AddEdge preserves multi-edge & loop handling according to graph flags
				if _, err := residual.AddEdge(u, v, capUV); err != nil {
					return nil, err
				}
			}
		}
	}

	return residual, nil
}
