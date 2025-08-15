// Package builder provides internal helper functions and types
// for configuring ID schemes in graph constructors.
package builder

import (
	"fmt"
	"strconv"
)

// IDFn generates a vertex identifier from its zero‐based index.
// It must be a pure, deterministic function: given the same idx, it always returns the same string.
// Panics in implementations indicate programmer error in configuration.
type IDFn func(idx int) string

// DefaultIDFn returns the decimal string of idx, e.g. 0→"0", 42→"42".
// Complexity: O(d) time where d = number of digits in idx, O(1) extra space.
// Never panics.
func DefaultIDFn(idx int) string {
	return strconv.Itoa(idx)
}

// SymbolIDFn returns the uppercase Latin letter for idx in [0..25], e.g. 0→"A", 25→"Z".
// Complexity: O(1) time, O(1) space.
// Panics if idx < 0 or idx > 25.
func SymbolIDFn(idx int) string {
	if idx < 0 || idx > 25 {
		panic(fmt.Sprintf("SymbolIDFn: idx must be in [0,25], got %d", idx))
	}
	// convert the computed letter‐code to a rune, then to string
	return string('A' + rune(idx))
}

// AlphanumericIDFn returns a base-36 string for idx, e.g. 0→"0", 10→"a", 35→"z", 36→"10".
// Complexity: O(d) time where d = base-36 digit count, O(1) extra space.
// Panics if idx < 0.
func AlphanumericIDFn(idx int) string {
	if idx < 0 {
		panic(fmt.Sprintf("AlphanumericIDFn: idx must be ≥ 0, got %d", idx))
	}

	return strconv.FormatInt(int64(idx), 36)
}

// ExcelColumnIDFn returns the “Excel‐style” column name for idx, e.g. 0→"A", 25→"Z", 26→"AA".
// Complexity: O(k) time where k ≈ log₍₂₆₎(idx), O(1) extra space.
// Panics if idx < 0.
func ExcelColumnIDFn(idx int) string {
	if idx < 0 {
		panic(fmt.Sprintf("ExcelColumnIDFn: idx must be ≥ 0, got %d", idx))
	}
	// build letters in reverse order
	var runes []rune
	var i, j int
	for i = idx; i >= 0; i = i/26 - 1 { // 26 alphabet size
		// each step produces one letter
		runes = append(runes, rune('A'+(i%26)))
	}
	// reverse in-place to correct order
	for i, j = 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// HexIDFn returns the lowercase hexadecimal representation of idx,
// e.g. 0→"0", 10→"a", 255→"ff".
// Complexity: O(d) time where d = hex digit count, O(1) space.
// Panics if idx < 0.
func HexIDFn(idx int) string {
	if idx < 0 {
		panic(fmt.Sprintf("HexIDFn: idx must be ≥ 0, got %d", idx))
	}

	return strconv.FormatInt(int64(idx), 16)
}

// SymbolNumberIDFn returns prefix + decimal index, e.g. "v0", "v1", ...
// Complexity: O(d) where d is the number of decimal digits in idx.
// Panics if idx < 0.
func SymbolNumberIDFn(prefix string) IDFn {
	return func(idx int) string {
		if idx < 0 {
			panic(fmt.Sprintf("SymbolNumberIDFn: idx must be ≥ 0, got %d", idx))
		}
		return prefix + strconv.Itoa(idx)
	}
}

// WithSymbNumb sets the ID scheme to SymbolNumberIDFn(prefix).
// Example: WithSymbNumb("v") → "v0","v1",...
// Complexity: O(1).
func WithSymbNumb(prefix string) BuilderOption {
	return WithIDScheme(SymbolNumberIDFn(prefix))
}

// WithDefaultIDs resets the ID scheme to DefaultIDFn.
// Complexity: O(1).
func WithDefaultIDs() BuilderOption {
	return WithIDScheme(DefaultIDFn)
}

// WithSymbolIDs sets the ID scheme to SymbolIDFn.
// Complexity: O(1).
func WithSymbolIDs() BuilderOption {
	return WithIDScheme(SymbolIDFn)
}

// WithExcelColumnIDs sets the ID scheme to ExcelColumnIDFn.
// Complexity: O(1).
func WithExcelColumnIDs() BuilderOption {
	return WithIDScheme(ExcelColumnIDFn)
}

// WithHexIDs sets the ID scheme to HexIDFn.
// Complexity: O(1).
func WithHexIDs() BuilderOption {
	return WithIDScheme(HexIDFn)
}

// WithAlphanumericIDs sets the ID scheme to AlphanumericIDFn.
// Complexity: O(1).
func WithAlphanumericIDs() BuilderOption {
	return WithIDScheme(AlphanumericIDFn)
}
