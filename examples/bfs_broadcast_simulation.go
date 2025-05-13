// Package main simulates a broadcast wave across a peer-to-peer network,
// using BFS layers to record the “round” at which each peer receives the message.
// Playground: https://go.dev/play/p/SRSzPfBv2s8
//
// Scenario:
//   - Peers: A, B, C, D, E, F, G
//   - Links define who can forward to whom.
//   - Round 0: only A has the data; each subsequent round, all neighbors receive it.
//
// Expectation: compute the round number each peer first hears the broadcast.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/katalvlaran/lvlath/algorithms"
	"github.com/katalvlaran/lvlath/core"
)

func main0() {
	// 1) Build peer network
	g := core.NewGraph(false, false)
	for _, e := range [][2]string{
		{"A", "B"}, {"A", "C"},
		{"B", "D"}, {"C", "D"},
		{"C", "E"}, {"E", "F"},
		{"D", "G"}, {"F", "G"},
	} {
		g.AddEdge(e[0], e[1], 0)
	}

	// 2) Prepare to record enqueue times as rounds
	rounds := make(map[string]int)
	opts := &algorithms.BFSOptions{
		OnEnqueue: func(v *core.Vertex, depth int) {
			// depth corresponds to broadcast round
			rounds[v.ID] = depth
		},
	}

	// 3) Run BFS with cancellation on timeout (just as an example)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	opts.Ctx = ctx

	if _, err := algorithms.BFS(g, "A", opts); err != nil {
		fmt.Println("BFS error:", err)
		return
	}

	// 4) Print broadcast schedule by round
	fmt.Println("Broadcast rounds:")
	for peer, r := range rounds {
		fmt.Printf("  Round %d → Peer %s\n", r, peer)
	}
}
