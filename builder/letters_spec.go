// SPDX-License-Identifier: MIT
// Package: lvlath/builder
//
// letters_spec.go - canonical per-glyph vertex/edge specifications (data-only).
//
// Purpose:
//   - This file is the single source of truth for glyph geometry on the approved
//     5×7 grid. It provides a deterministic, minimal-yet-sufficient set of
//     vertices and edges for all 52 Latin letters (A..Z, a..z) and digits 0..9.
//   - Vertex IDs strictly follow "<Glyph>_<Horiz>_<Vert>" where
//       Horiz ∈ {L, LC, C, RC, R}   (Leftmost → Rightmost)
//       Vert  ∈ {T, PT, UM, M, PM, UB, B} (Topmost → Bottommost)
//     These symbolic tokens avoid magic strings and are shared across the repo.
//   - Data here are immutable; building logic (weights, directed mirroring,
//     idempotency, namespacing) lives in impl_letters.go.
//
// Contract (for consumers such as impl_letters.go):
//   - For each glyph, letterSpec.IDs is the canonical vertex-add order
//     (lexicographic, computed once at makeSpec).
//   - letterSpec.Edges is the canonical emission order (drawing order).
//   - Edges are undirected in semantics; builders mirror them explicitly for
//     Directed() graphs.
//   - No weight policy is applied here (data-only); builders choose weight=1
//     for Weighted() graphs, otherwise 0.
//   - OptionalEdges is reserved for stylistic extras if/when formalized.
//
// Determinism:
//   - All vertex IDs follow a stable pattern and are sorted once (O(V log V), V is tiny).
//   - Edge lists are hand-ordered and stable; do not mutate after review.
//
// Complexity (for consumers):
//   - Add vertices: O(V). Emit edges: O(E). Constants are small for 5×7 glyphs.
//
// AI-Hints:
//   - To extend the alphabet (e.g., punctuation, math signs), add data-only
//     entries with the same pattern, then builders will pick them up automatically.
//   - Keep changes append-only and non-breaking: never rename existing IDs,
//     never reorder existing edges; only add new glyphs or OptionalEdges.
//
// Notes:
//   - Shapes are expressed as one or more polylines over the 5×7 grid.
//   - To keep data concise and precise, helper functions poly() and bar() generate edge chains.
//   - Optional edges can be added later (e.g., stylistic tails) via OptionalEdges when formalized.

package builder

import (
	"sort"
)

// -----------------------------------------------------------------------------
// Grid tokens - no magic strings.
// -----------------------------------------------------------------------------

// Horizontal positions (Leftmost…Rightmost).
const (
	L  = "L"  // Leftmost
	LC = "LC" // LeftCenter
	C  = "C"  // Center
	RC = "RC" // RightCenter
	R  = "R"  // Rightmost
)

// Vertical positions (Topmost … Bottommost).
const (
	T  = "T"  // Topmost
	PT = "PT" // PreTop
	UM = "UM" // UpperMedium
	M  = "M"  // Medium
	PM = "PM" // PreMedium
	UB = "UB" // UpperBottom
	B  = "B"  // Bottommost
)

// JOINER is the only allowed delimiter inside canonical glyph vertex IDs.
// Changing JOINER is a breaking change for fixtures and golden tests.
const JOINER = "_"

// -----------------------------------------------------------------------------
// Data model
// -----------------------------------------------------------------------------

// letterSpec groups canonical vertices and edges for one glyph.
type letterSpec struct {
	IDs           []string   // deterministic vertex-add order (sorted)
	Edges         []edgePair // minimal skeleton, drawing order
	OptionalEdges []edgePair // allowed extras (currently empty unless stated)
}

// -----------------------------------------------------------------------------
// ID helpers (stable ID composition; avoids magic strings in call sites)
// -----------------------------------------------------------------------------

// id composes "<glyph>_<horizontal>_<vertical>" deterministically.
// The function is intentionally trivial to keep the data file pure.
func id(glyph, horizontal, vertical string) string {
	return glyph + JOINER + horizontal + JOINER + vertical
}

// -----------------------------------------------------------------------------
// Spec builder (collect unique IDs + stable lexicographic order)
// -----------------------------------------------------------------------------

// makeSpec constructs a letterSpec by:
//   - collecting all unique vertex IDs from edges and optionalEdges,
//   - sorting them lexicographically for a stable vertex-add order,
//   - preserving the given edge order as "emission order".
//
// Complexity: O(V log V) for sorting tiny V (dozens at most).
func makeSpec(edges []edgePair, optionalEdges []edgePair) letterSpec {
	// 1) Collect unique vertex IDs into a set.
	set := make(map[string]struct{}, len(edges)*2+len(optionalEdges)*2) // reserve capacity
	// Collect from required edges.
	for _, ep := range edges {
		// insert U endpoint (idempotent)
		set[ep.U] = struct{}{}
		// insert V endpoint (idempotent)
		set[ep.V] = struct{}{}
	}
	// Collect from optional edges (keeps IDs present even if not emitted).
	for _, ep := range optionalEdges {
		set[ep.U] = struct{}{}
		set[ep.V] = struct{}{}
	}

	// 2) Build the sorted slice for canonical vertex order.
	ids := make([]string, 0, len(set)) // allocate exactly once
	for token := range set {
		ids = append(ids, token) // append each unique ID
	}
	// sort lexicographically for a deterministic add order
	sort.Strings(ids)

	// 3) Return fully-populated, immutable spec.
	return letterSpec{
		IDs:           ids,           // stable, sorted
		Edges:         edges,         // keep caller-provided emission order
		OptionalEdges: optionalEdges, // keep caller-provided emission order
	}
}

// -----------------------------------------------------------------------------
// Canonical letter registry (ALL 52 letters) - edges only, stable order.
// Each entry uses ONE uniform comment style:
//
//	// <Letter> - canonical minimal 5×7 skeleton; edges listed in drawing order.
//
// -----------------------------------------------------------------------------
var letterSpecs = map[rune]letterSpec{
	// ===== UPPERCASE (A..Z) =====
	//-----------------------------

	// A - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1...1
	// 11111
	// 1...1
	// 1...1
	// 1...1
	'A': makeSpec([]edgePair{
		{id("A", L, B), id("A", L, M)},
		{id("A", L, M), id("A", L, PT)},
		{id("A", L, PT), id("A", LC, T)},
		{id("A", LC, T), id("A", RC, T)},
		{id("A", RC, T), id("A", R, PT)},
		{id("A", R, PT), id("A", R, M)},
		{id("A", R, M), id("A", R, B)},
		{id("A", L, M), id("A", R, M)},
	}, nil),

	// B - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1111.
	// 1...1
	// 1...1
	// 1111.
	// 1...1
	// 1...1
	// 1111.
	'B': makeSpec([]edgePair{
		{id("B", L, B), id("B", L, M)},
		{id("B", L, M), id("B", L, T)},
		{id("B", L, T), id("B", RC, T)},
		{id("B", RC, T), id("B", R, PT)},
		{id("B", R, PT), id("B", R, UM)},
		{id("B", R, UM), id("B", RC, M)},
		{id("B", RC, M), id("B", RC, B)},
		{id("B", RC, M), id("B", R, PM)},
		{id("B", R, PM), id("B", R, UB)},
		{id("B", R, UB), id("B", RC, B)},
		{id("B", RC, B), id("B", L, B)},
	}, nil),

	// C - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1....
	// 1....
	// 1....
	// 1...1
	// .111.
	'C': makeSpec([]edgePair{
		{id("C", R, PT), id("C", RC, T)},
		{id("C", RC, T), id("C", LC, T)},
		{id("C", LC, T), id("C", L, PT)},
		{id("C", L, PT), id("C", L, UB)},
		{id("C", L, UB), id("C", LC, B)},
		{id("C", LC, B), id("C", RC, B)},
		{id("C", RC, B), id("C", R, UB)},
	}, nil),

	// D - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1111.
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// 1111.
	'D': makeSpec([]edgePair{
		{id("D", L, B), id("D", L, T)},
		{id("D", L, T), id("D", RC, T)},
		{id("D", RC, T), id("D", R, PT)},
		{id("D", R, PT), id("D", R, UB)},
		{id("D", R, UB), id("D", RC, B)},
		{id("D", RC, B), id("D", L, B)},
	}, nil),

	// E - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 11111
	// 1....
	// 1....
	// 111..
	// 1....
	// 1....
	// 11111
	'E': makeSpec([]edgePair{
		{id("E", R, B), id("E", L, B)},
		{id("E", L, B), id("E", L, M)},
		{id("E", L, M), id("E", L, T)},
		{id("E", L, T), id("E", R, T)},
		{id("E", L, M), id("E", C, M)},
	}, nil),

	// F - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 11111
	// 1....
	// 1....
	// 111..
	// 1....
	// 1....
	// 1....
	'F': makeSpec([]edgePair{
		{id("F", L, B), id("F", L, M)},
		{id("F", L, M), id("F", L, T)},
		{id("F", L, T), id("F", R, T)},
		{id("F", L, M), id("F", C, M)},
	}, nil),

	// G - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1....
	// 1....
	// 1..11
	// 1...1
	// .111.
	'G': makeSpec([]edgePair{
		{id("G", R, PT), id("G", RC, T)},
		{id("G", RC, T), id("G", LC, T)},
		{id("G", LC, T), id("G", L, PT)},
		{id("G", L, PT), id("G", L, UB)},
		{id("G", L, UB), id("G", LC, B)},
		{id("G", LC, B), id("G", RC, B)},
		{id("G", RC, B), id("G", R, UB)},
		{id("G", R, UB), id("G", R, PM)},
		{id("G", R, PM), id("G", RC, PM)},
	}, nil),

	// H - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 1...1
	// 1...1
	// 11111
	// 1...1
	// 1...1
	// 1...1
	'H': makeSpec([]edgePair{
		{id("H", L, B), id("H", L, M)},
		{id("H", L, M), id("H", L, T)},
		{id("H", R, B), id("H", R, M)},
		{id("H", R, M), id("H", R, T)},
		{id("H", L, M), id("H", R, M)},
	}, nil),

	// I - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// ..1..
	// ..1..
	// ..1..
	// ..1..
	// ..1..
	// .111.
	'I': makeSpec([]edgePair{
		{id("I", RC, B), id("I", C, B)},
		{id("I", C, B), id("I", LC, B)},
		{id("I", RC, T), id("I", C, T)},
		{id("I", C, T), id("I", LC, T)},
		{id("I", C, B), id("I", C, T)},
	}, nil),

	// J - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// ....1
	// ....1
	// ....1
	// ....1
	// ....1
	// 1...1
	// .111.
	'J': makeSpec([]edgePair{
		{id("J", R, T), id("J", R, UB)},
		{id("J", R, UB), id("J", RC, B)},
		{id("J", RC, B), id("J", LC, B)},
		{id("J", LC, B), id("J", L, UB)},
	}, nil),

	// K - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 1..1.
	// 1.1..
	// 11...
	// 1.1..
	// 1..1.
	// 1...1
	'K': makeSpec([]edgePair{
		{id("K", L, B), id("K", L, M)},
		{id("K", L, M), id("K", L, T)},
		{id("K", L, M), id("K", R, T)},
		{id("K", L, M), id("K", R, B)},
	}, nil),

	// L - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1....
	// 1....
	// 1....
	// 1....
	// 1....
	// 1....
	// 11111
	'L': makeSpec([]edgePair{
		{id("L", L, T), id("L", L, B)},
		{id("L", L, B), id("L", R, B)},
	}, nil),

	// M - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 11.11
	// 11.11
	// 1.1.1
	// 1...1
	// 1...1
	// 1...1
	'M': makeSpec([]edgePair{
		{id("M", L, B), id("M", L, T)},
		{id("M", L, T), id("M", C, M)},
		{id("M", C, M), id("M", R, T)},
		{id("M", R, T), id("M", R, B)},
	}, nil),

	// N - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 11..1
	// 1.1.1
	// 1.1.1
	// 1.1.1
	// 1..11
	// 1...1
	'N': makeSpec([]edgePair{
		{id("N", L, B), id("N", L, T)},
		{id("N", L, T), id("N", R, B)},
		{id("N", R, B), id("N", R, T)},
	}, nil),

	// O - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// .111.
	'O': makeSpec([]edgePair{
		{id("O", LC, B), id("O", L, UB)},
		{id("O", L, UB), id("O", L, PT)},
		{id("O", L, PT), id("O", LC, T)},
		{id("O", LC, T), id("O", RC, T)},
		{id("O", RC, T), id("O", R, PT)},
		{id("O", R, PT), id("O", R, UB)},
		{id("O", R, UB), id("O", RC, B)},
		{id("O", RC, B), id("O", LC, B)},
	}, nil),

	// P - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1111.
	// 1...1
	// 1...1
	// 1111.
	// 1....
	// 1....
	// 1....
	'P': makeSpec([]edgePair{
		{id("P", L, B), id("P", L, M)},
		{id("P", L, M), id("P", L, T)},
		{id("P", L, T), id("P", RC, T)},
		{id("P", RC, T), id("P", R, PT)},
		{id("P", R, PT), id("P", R, UM)},
		{id("P", R, UM), id("P", RC, M)},
		{id("P", RC, M), id("P", L, M)},
	}, nil),

	// Q - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// 1..1.
	// .11.1
	'Q': makeSpec([]edgePair{
		{id("Q", C, B), id("Q", LC, B)},
		{id("Q", LC, B), id("Q", L, UB)},
		{id("Q", L, UB), id("Q", L, PT)},
		{id("Q", L, PT), id("Q", LC, T)},
		{id("Q", LC, T), id("Q", RC, T)},
		{id("Q", RC, T), id("Q", R, PT)},
		{id("Q", R, PT), id("Q", R, PM)},
		{id("Q", R, PM), id("Q", RC, UB)},
		{id("Q", RC, UB), id("Q", R, B)},
		{id("Q", RC, UB), id("Q", C, B)},
	}, nil),

	// R - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1111.
	// 1...1
	// 1...1
	// 1111.
	// 1...1
	// 1...1
	// 1...1
	'R': makeSpec([]edgePair{
		{id("R", L, B), id("R", L, M)},
		{id("R", L, M), id("R", L, T)},
		{id("R", L, T), id("R", RC, T)},
		{id("R", RC, T), id("R", R, PT)},
		{id("R", R, PT), id("R", R, UM)},
		{id("R", R, UM), id("R", RC, M)},
		{id("R", RC, M), id("R", L, M)},
		{id("R", RC, M), id("R", R, PM)},
		{id("R", R, PM), id("R", R, B)},
	}, nil),

	// S - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1....
	// .111.
	// ....1
	// 1...1
	// .111.
	'S': makeSpec([]edgePair{
		{id("S", R, PT), id("S", RC, T)},
		{id("S", RC, T), id("S", LC, T)},
		{id("S", LC, T), id("S", L, PT)},
		{id("S", L, PT), id("S", L, UM)},
		{id("S", L, UM), id("S", LC, M)},
		{id("S", LC, M), id("S", RC, M)},
		{id("S", RC, M), id("S", R, PM)},
		{id("S", R, PM), id("S", R, UB)},
		{id("S", R, UB), id("S", RC, B)},
		{id("S", RC, B), id("S", LC, B)},
		{id("S", LC, B), id("S", L, UB)},
	}, nil),

	// T - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 11111
	// ..1..
	// ..1..
	// ..1..
	// ..1..
	// ..1..
	// ..1..
	'T': makeSpec([]edgePair{
		{id("T", L, T), id("T", M, T)},
		{id("T", M, T), id("T", R, T)},
		{id("T", M, T), id("T", M, B)},
	}, nil),

	// U - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// .111.
	'U': makeSpec([]edgePair{
		{id("U", L, T), id("U", L, UB)},
		{id("U", L, UB), id("U", LC, B)},
		{id("U", LC, B), id("U", RC, B)},
		{id("U", RC, B), id("U", R, UB)},
		{id("U", R, UB), id("U", R, T)},
	}, nil),

	// V - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// .1.1.
	// .1.1.
	// ..1..
	'V': makeSpec([]edgePair{
		{id("V", L, T), id("V", C, B)},
		{id("V", C, B), id("V", R, T)},
	}, nil),

	// W - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// 1.1.1
	// 1.1.1
	// .1.1.
	'W': makeSpec([]edgePair{
		{id("W", L, T), id("W", LC, B)},
		{id("W", LC, B), id("W", C, M)},
		{id("W", C, M), id("W", RC, B)},
		{id("W", RC, B), id("W", R, T)},
	}, nil),

	// X - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 1...1
	// .1.1.
	// ..1..
	// .1.1.
	// 1...1
	// 1...1
	'X': makeSpec([]edgePair{
		{id("X", L, T), id("X", C, M)},
		{id("X", C, M), id("X", R, B)},
		{id("X", R, T), id("X", C, M)},
		{id("X", C, M), id("X", L, B)},
	}, nil),

	// Y - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1...1
	// 1...1
	// .1.1.
	// ..1..
	// ..1..
	// ..1..
	// ..1..
	'Y': makeSpec([]edgePair{
		{id("Y", L, T), id("Y", C, M)},
		{id("Y", C, M), id("Y", R, T)},
		{id("Y", C, M), id("Y", C, B)},
	}, nil),

	// Z - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 11111
	// ....1
	// ...1.
	// ..1..
	// .1...
	// 1....
	// 11111
	'Z': makeSpec([]edgePair{
		{id("Z", L, T), id("Z", R, T)},
		{id("Z", R, T), id("Z", L, B)},
		{id("Z", L, B), id("Z", R, B)},
	}, nil),

	// ===== LOWERCASE (a..z) =====
	//-----------------------------

	// a - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// .111.
	// ....1
	// .1111
	// 1...1
	// .1111
	'a': makeSpec([]edgePair{
		{id("a", LC, UM), id("a", RC, UM)},
		{id("a", RC, UM), id("a", R, M)},
		{id("a", R, M), id("a", R, UB)},
		{id("a", R, UB), id("a", RC, PM)},
		{id("a", RC, PM), id("a", LC, PM)},
		{id("a", LC, PM), id("a", L, UB)},
		{id("a", L, UB), id("a", LC, B)},
		{id("a", LC, B), id("a", RC, B)},
		{id("a", RC, B), id("a", R, UB)},
		{id("a", R, UB), id("a", R, B)},
	}, nil),

	// b - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1....
	// 1....
	// 1....
	// 1111.
	// 1...1
	// 1...1
	// 1111.
	'b': makeSpec([]edgePair{
		{id("b", L, T), id("b", L, M)},
		{id("b", L, M), id("b", L, B)},
		{id("b", L, B), id("b", RC, B)},
		{id("b", RC, B), id("b", R, UB)},
		{id("b", R, UB), id("b", R, PM)},
		{id("b", R, PM), id("b", RC, M)},
		{id("b", RC, M), id("b", L, M)},
	}, nil),

	// c - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// .111.
	// 1...1
	// 1....
	// 1...1
	// .111.
	'c': makeSpec([]edgePair{
		{id("c", R, M), id("c", RC, UM)},
		{id("c", RC, UM), id("c", LC, UM)},
		{id("c", LC, UM), id("c", L, M)},
		{id("c", L, M), id("c", L, UB)},
		{id("c", L, UB), id("c", LC, B)},
		{id("c", LC, B), id("c", RC, B)},
		{id("c", RC, B), id("c", R, UB)},
	}, nil),

	// d - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// ....1
	// ....1
	// ....1
	// .1111
	// 1...1
	// 1...1
	// .1111
	'd': makeSpec([]edgePair{
		{id("d", R, T), id("d", R, M)},
		{id("d", R, M), id("d", R, B)},
		{id("d", R, B), id("d", LC, B)},
		{id("d", LC, B), id("d", L, UB)},
		{id("d", L, UB), id("d", L, PM)},
		{id("d", L, PM), id("d", LC, M)},
		{id("d", LC, M), id("d", R, M)},
	}, nil),

	// e - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// .111.
	// 1...1
	// 11111
	// 1....
	// .1111
	'e': makeSpec([]edgePair{
		{id("e", L, PM), id("e", R, PM)},
		{id("e", R, PM), id("e", R, M)},
		{id("e", R, M), id("e", RC, UM)},
		{id("e", RC, UM), id("e", LC, UM)},
		{id("e", LC, UM), id("e", L, M)},
		{id("e", L, M), id("e", L, PM)},
		{id("e", L, PM), id("e", L, UB)},
		{id("e", L, UB), id("e", LC, B)},
		{id("e", LC, B), id("e", R, B)},
	}, nil),

	// f - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1....
	// 111..
	// 1....
	// 1....
	// 1....
	'f': makeSpec([]edgePair{
		{id("f", R, PT), id("f", RC, T)},
		{id("f", RC, T), id("f", LC, T)},
		{id("f", LC, T), id("f", L, PT)},
		{id("f", L, PT), id("f", L, M)},
		{id("f", L, M), id("f", L, B)},
		{id("f", L, M), id("f", C, M)},
	}, nil),

	// g - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1...1
	// .1111
	// ....1
	// ....1
	// .111.
	'g': makeSpec([]edgePair{
		{id("g", R, UM), id("g", R, PT)},
		{id("g", R, PT), id("g", RC, T)},
		{id("g", RC, T), id("g", LC, T)},
		{id("g", LC, T), id("g", L, PT)},
		{id("g", L, PT), id("g", L, UM)},
		{id("g", L, UM), id("g", LC, M)},
		{id("g", LC, M), id("g", RC, M)},
		{id("g", RC, M), id("g", R, UM)},
		{id("g", R, UM), id("g", R, UB)},
		{id("g", R, UB), id("g", RC, B)},
		{id("g", RC, B), id("g", LC, B)},
	}, nil),

	// h - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1....
	// 1....
	// 1....
	// 1111.
	// 1...1
	// 1...1
	// 1...1
	'h': makeSpec([]edgePair{
		{id("h", L, T), id("h", L, M)},
		{id("h", L, M), id("h", L, B)},
		{id("h", L, M), id("h", RC, M)},
		{id("h", RC, M), id("h", R, PM)},
		{id("h", R, PM), id("h", R, B)},
	}, nil),

	// i - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// ..1..
	// .....
	// ..1..
	// ..1..
	// ..1..
	// .111.
	'i': makeSpec([]edgePair{
		{id("i", C, PT), id("i", C, PT)},
		{id("i", C, M), id("i", C, B)},
		{id("i", LC, B), id("i", C, B)},
		{id("i", C, B), id("i", RC, B)},
	}, nil),

	// j - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// ..1..
	// .....
	// ..1..
	// ..1..
	// ..1..
	// 1.1..
	// .1...
	'j': makeSpec([]edgePair{
		{id("j", C, T), id("j", C, T)},
		{id("j", C, UM), id("j", C, UB)},
		{id("j", C, UB), id("j", LC, B)},
		{id("j", LC, B), id("j", L, UB)},
	}, nil),

	// k - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1....
	// 1....
	// 1..1.
	// 1.1..
	// 111..
	// 1..1.
	// 1...1
	'k': makeSpec([]edgePair{
		{id("k", L, T), id("k", L, PM)},
		{id("k", L, PM), id("k", L, B)},
		{id("k", L, PM), id("k", RC, UM)},
		{id("k", L, PM), id("k", R, B)},
	}, nil),

	// l - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 1....
	// 1....
	// 1....
	// 1....
	// 1....
	// 1....
	// .1...
	'l': makeSpec([]edgePair{
		{id("l", L, T), id("l", L, UB)},
		{id("l", L, UB), id("l", LC, B)},
	}, nil),

	// m - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 1...1.
	// 11.11
	// 1.1.1
	// 1...1
	// 1...1
	'm': makeSpec([]edgePair{
		{id("m", L, B), id("m", L, UM)},
		{id("m", L, UM), id("m", C, PM)},
		{id("m", C, PM), id("m", R, UM)},
		{id("m", R, UM), id("m", R, B)},
	}, nil),

	// n - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 1111.
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	'n': makeSpec([]edgePair{
		{id("n", L, B), id("n", L, UM)},
		{id("n", L, UM), id("n", RC, UM)},
		{id("n", RC, UM), id("n", R, M)},
		{id("n", R, M), id("n", R, B)},
	}, nil),

	// o - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// .111.
	// 1...1
	// 1...1
	// 1...1
	// .111.
	'o': makeSpec([]edgePair{
		{id("o", RC, UM), id("o", LC, UM)},
		{id("o", LC, UM), id("o", L, M)},
		{id("o", L, M), id("o", L, UB)},
		{id("o", L, UB), id("o", LC, B)},
		{id("o", LC, B), id("o", RC, B)},
		{id("o", RC, B), id("o", R, UB)},
		{id("o", R, UB), id("o", R, M)},
		{id("o", R, M), id("o", RC, UM)},
	}, nil),

	// p - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .111.
	// 1...1
	// 1...1
	// 1111.
	// 1....
	// 1....
	'p': makeSpec([]edgePair{
		{id("p", L, B), id("p", L, M)},
		{id("p", L, M), id("p", L, UM)},
		{id("p", L, UM), id("p", LC, PT)},
		{id("p", LC, PT), id("p", RC, PT)},
		{id("p", RC, PT), id("p", R, UM)},
		{id("p", R, UM), id("p", R, M)},
		{id("p", R, M), id("p", RC, PM)},
		{id("p", RC, PM), id("p", LC, PM)},
		{id("p", LC, PM), id("p", L, M)},
	}, nil),

	// q - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .111.
	// 1...1
	// 1...1
	// .1111
	// ....1
	// ....1
	'q': makeSpec([]edgePair{
		{id("q", R, B), id("q", R, M)},
		{id("q", R, M), id("q", R, UM)},
		{id("q", R, UM), id("q", RC, PT)},
		{id("q", RC, PT), id("q", LC, PT)},
		{id("q", LC, PT), id("q", L, UM)},
		{id("q", L, UM), id("q", L, M)},
		{id("q", L, M), id("q", LC, PM)},
		{id("q", LC, PM), id("q", RC, PM)},
		{id("q", RC, PM), id("q", R, M)},
	}, nil),

	// r - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 1.111
	// 11...
	// 1....
	// 1....
	// 1....
	'r': makeSpec([]edgePair{
		{id("r", L, UM), id("r", L, M)},
		{id("r", L, M), id("r", L, B)},
		{id("r", L, M), id("r", R, UM)},
	}, nil),

	// s - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// .1111
	// 1....
	// .111.
	// ....1
	// 1111.
	's': makeSpec([]edgePair{
		{id("s", R, UM), id("s", LC, UM)},
		{id("s", LC, UM), id("s", L, M)},
		{id("s", L, M), id("s", LC, PM)},
		{id("s", LC, PM), id("s", RC, PM)},
		{id("s", RC, PM), id("s", R, UB)},
		{id("s", R, UB), id("s", RC, B)},
		{id("s", RC, B), id("s", L, B)},
	}, nil),

	// t - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// ..1..
	// 11111
	// ..1..
	// ..1..
	// ..1..
	// ...1.
	't': makeSpec([]edgePair{
		{id("t", C, T), id("t", C, UM)},
		{id("t", C, UM), id("t", C, UB)},
		{id("t", C, UB), id("t", RC, B)},
		{id("t", L, UM), id("t", C, UM)},
		{id("t", C, UM), id("t", R, UM)},
	}, nil),

	// u - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 1...1
	// 1...1
	// 1...1
	// 1...1
	// .111.
	'u': makeSpec([]edgePair{
		{id("u", L, UM), id("u", L, UB)},
		{id("u", L, UB), id("u", LC, B)},
		{id("u", LC, B), id("u", RC, B)},
		{id("u", RC, B), id("u", R, UB)},
		{id("u", R, UB), id("u", R, UM)},
	}, nil),

	// v - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 1...1
	// 1...1
	// 1...1
	// .1.1.
	// ..1..
	'v': makeSpec([]edgePair{
		{id("v", L, UM), id("v", C, B)},
		{id("v", C, B), id("v", R, PM)},
	}, nil),

	// w - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 1...1
	// 1...1
	// 1...1
	// 1.1.1
	// .1.1.
	'w': makeSpec([]edgePair{
		{id("w", L, UM), id("w", LC, B)},
		{id("w", LC, B), id("w", C, UB)},
		{id("w", LC, B), id("w", RC, B)},
		{id("w", RC, B), id("w", R, UM)},
	}, nil),

	// x - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 1...1
	// .1.1.
	// ..1..
	// .1.1.
	// 1...1
	'x': makeSpec([]edgePair{
		{id("x", L, UM), id("x", C, PM)},
		{id("x", C, PM), id("x", R, B)},
		{id("x", R, UM), id("x", C, PM)},
		{id("x", C, PM), id("x", L, B)},
	}, nil),

	// y - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 1...1
	// .1.1.
	// ..1..
	// .1...
	// 1....
	'y': makeSpec([]edgePair{
		{id("y", L, UM), id("y", C, PM)},
		{id("y", C, PM), id("y", R, UM)},
		{id("y", C, PM), id("y", L, B)},
	}, nil),

	// z - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .....
	// .....
	// 11111
	// ...1.
	// ..1..
	// .1...
	// 11111
	'z': makeSpec([]edgePair{
		{id("z", L, UM), id("z", R, UM)},
		{id("z", R, UM), id("z", L, B)},
		{id("z", L, B), id("z", R, B)},
	}, nil),
}

var numberSpec = map[rune]letterSpec{
	// ===== NUMBERS (0..9) =====
	//---------------------------

	// 0 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1...1
	// 1.1.1
	// 1...1
	// 1...1
	// .111.
	'0': makeSpec([]edgePair{
		{id("0", RC, T), id("0", LC, T)},
		{id("0", LC, T), id("0", L, PT)},
		{id("0", L, PT), id("0", L, UB)},
		{id("0", L, UB), id("0", LC, B)},
		{id("0", LC, B), id("0", RC, B)},
		{id("0", RC, B), id("0", R, UB)},
		{id("0", R, UB), id("0", R, PT)},
		{id("0", R, PT), id("0", RC, T)},
		// +
		{id("0", C, M), id("0", C, M)},
	}, nil),

	// 1 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// ..1..
	// .11..
	// ..1..
	// ..1..
	// ..1..
	// ..1..
	// .111.
	'1': makeSpec([]edgePair{
		{id("1", LC, PT), id("1", C, T)},
		{id("1", C, T), id("1", C, B)},
		{id("1", LC, B), id("1", C, B)},
		{id("1", C, B), id("1", RC, B)},
	}, nil),

	// 2 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// ....1
	// ...1.
	// ..1..
	// .1...
	// 11111
	'2': makeSpec([]edgePair{
		{id("2", L, PT), id("2", LC, T)},
		{id("2", LC, T), id("2", RC, T)},
		{id("2", RC, T), id("2", R, PT)},
		{id("2", R, PT), id("2", R, UM)},
		{id("2", R, UM), id("2", L, B)},
		{id("2", L, B), id("2", R, B)},
	}, nil),

	// 3 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// ....1
	// .111.
	// ....1
	// 1...1
	// .111.
	'3': makeSpec([]edgePair{
		{id("3", L, PT), id("3", LC, T)},
		{id("3", LC, T), id("3", RC, T)},
		{id("3", RC, T), id("3", R, PT)},
		{id("3", R, PT), id("3", R, UM)},
		{id("3", R, UM), id("3", RC, M)},
		{id("3", RC, M), id("3", LC, M)},
		{id("3", RC, M), id("3", R, PM)},
		{id("3", R, PM), id("3", R, UB)},
		{id("3", R, UB), id("3", RC, B)},
		{id("3", RC, B), id("3", LC, B)},
		{id("3", LC, B), id("3", L, UB)},
	}, nil),

	// 4 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// ...11
	// ..1.1
	// .1..1
	// 11111
	// ....1
	// ....1
	// ....1
	'4': makeSpec([]edgePair{
		{id("4", R, T), id("4", L, M)},
		{id("4", L, M), id("4", R, M)},
		{id("4", R, T), id("4", R, M)},
		{id("4", R, M), id("4", R, B)},
	}, nil),

	// 5 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 11111
	// 1....
	// 1....
	// 1111.
	// ....1
	// ....1
	// 1111.
	'5': makeSpec([]edgePair{
		{id("5", L, T), id("5", L, M)},
		{id("5", L, M), id("5", RC, M)},
		{id("5", RC, M), id("5", R, PM)},
		{id("5", R, PM), id("5", R, UB)},
		{id("5", RC, B), id("5", L, B)},
		{id("5", L, T), id("5", R, T)},
	}, nil),

	// 6 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1....
	// 1111.
	// 1...1
	// 1...1
	// .111.
	'6': makeSpec([]edgePair{
		{id("6", R, PT), id("6", RC, T)},
		{id("6", RC, T), id("6", LC, T)},
		{id("6", LC, T), id("6", L, PT)},
		{id("6", L, PT), id("6", L, PM)},
		{id("6", L, PM), id("6", L, UB)},
		{id("6", L, UB), id("6", LC, B)},
		{id("6", LC, B), id("6", RC, B)},
		{id("6", RC, B), id("6", R, UB)},
		{id("6", R, UB), id("6", R, PM)},
		{id("6", R, PM), id("6", RC, M)},
		{id("6", RC, M), id("6", LC, M)},
		{id("6", LC, M), id("6", L, PM)},
	}, nil),

	// 7 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// 11111
	// ....1
	// ...1.
	// ..1..
	// .1...
	// 1....
	// 1....
	'7': makeSpec([]edgePair{
		{id("7", L, T), id("7", R, T)},
		{id("7", R, T), id("7", R, PT)},
		{id("7", R, PT), id("7", L, UB)},
		{id("7", L, UB), id("7", L, B)},
	}, nil),

	// 8 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1...1
	// .111.
	// 1...1
	// 1...1
	// .111.
	'8': makeSpec([]edgePair{
		{id("8", LC, M), id("8", RC, M)},
		{id("8", RC, M), id("8", R, UM)},
		{id("8", R, UM), id("8", R, PT)},
		{id("8", R, PT), id("8", RC, T)},
		{id("8", RC, T), id("8", LC, T)},
		{id("8", LC, T), id("8", L, PT)},
		{id("8", L, PT), id("8", L, UM)},
		{id("8", L, UM), id("8", LC, M)},
		{id("8", RC, M), id("8", R, PM)},
		{id("8", R, PM), id("8", R, UB)},
		{id("8", R, UB), id("8", RC, B)},
		{id("8", RC, B), id("8", LC, B)},
		{id("8", LC, B), id("8", L, UB)},
		{id("8", L, UB), id("8", L, PM)},
		{id("8", L, PM), id("8", LC, M)},
	}, nil),

	// 9 - canonical minimal 5×7 skeleton; edges listed in drawing order.
	// .111.
	// 1...1
	// 1...1
	// .1111
	// ....1
	// ....1
	// ..1..
	'9': makeSpec([]edgePair{
		{id("9", R, UM), id("9", R, PT)},
		{id("9", R, PT), id("9", RC, T)},
		{id("9", RC, T), id("9", LC, T)},
		{id("9", LC, T), id("9", L, PT)},
		{id("9", L, PT), id("9", L, UM)},
		{id("9", L, UM), id("9", LC, M)},
		{id("9", LC, M), id("9", RC, M)},
		{id("9", RC, M), id("9", R, UM)},
		{id("9", R, UM), id("9", R, UB)},
		{id("9", R, UB), id("9", RC, B)},
	}, nil),
}
