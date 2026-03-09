// SPDX-License-Identifier: AGPL-3.0-only
// Copyright (C) 2025-2026 katalvlaran

package bfs_test

import (
	"errors"
	"fmt"
	"testing"
)

// AI-HINTS (file):
//   - Explicit comparisons give better error messages and avoid hiding mismatches.
//   - Use errors.Is for error protocol checks; never compare error strings.
//   - Keep helpers small and deterministic; they must not allocate in hot loops (tests are OK, but stay disciplined).

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustErrorIs(t *testing.T, err error, sentinel error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %v, got nil", sentinel)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected errors.Is(err, %v)=true, got err=%v", sentinel, err)
	}
}

func mustEqualSlice(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("slice length mismatch: got=%d want=%d got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice mismatch at index %d: got=%q want=%q got=%v want=%v", i, got[i], want[i], got, want)
		}
	}
}

func mustEqualIntMap(t *testing.T, got, want map[string]int) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("map length mismatch: got=%d want=%d got=%v want=%v", len(got), len(want), got, want)
	}

	for k, wantV := range want {
		gotV, ok := got[k]
		if !ok {
			t.Fatalf("map missing key %q: got=%v want=%v", k, got, want)
		}
		if gotV != wantV {
			t.Fatalf("map value mismatch for key %q: got=%d want=%d got=%v want=%v", k, gotV, wantV, got, want)
		}
	}

	for k := range got {
		if _, ok := want[k]; !ok {
			t.Fatalf("map has unexpected key %q: got=%v want=%v", k, got, want)
		}
	}
}

func mustEqualBoolMap(t *testing.T, got, want map[string]bool) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("map length mismatch: got=%d want=%d got=%v want=%v", len(got), len(want), got, want)
	}

	for k, wantV := range want {
		gotV, ok := got[k]
		if !ok {
			t.Fatalf("map missing key %q: got=%v want=%v", k, got, want)
		}
		if gotV != wantV {
			t.Fatalf("map value mismatch for key %q: got=%t want=%t got=%v want=%v", k, gotV, wantV, got, want)
		}
	}

	for k := range got {
		if _, ok := want[k]; !ok {
			t.Fatalf("map has unexpected key %q: got=%v want=%v", k, got, want)
		}
	}
}

func mustEqualStringMap(t *testing.T, got, want map[string]string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("map length mismatch: got=%d want=%d got=%v want=%v", len(got), len(want), got, want)
	}

	for k, wantV := range want {
		gotV, ok := got[k]
		if !ok {
			t.Fatalf("map missing key %q: got=%v want=%v", k, got, want)
		}
		if gotV != wantV {
			t.Fatalf("map value mismatch for key %q: got=%q want=%q got=%v want=%v", k, gotV, wantV, got, want)
		}
	}

	for k := range got {
		if _, ok := want[k]; !ok {
			t.Fatalf("map has unexpected key %q: got=%v want=%v", k, got, want)
		}
	}
}

func mustEqualBool(t *testing.T, got, want bool, format string, args ...any) {
	t.Helper()
	if got != want {
		t.Fatalf(format, args...)
	}
}

func mustPanicFree(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	fn()
}

func mustFmt(t *testing.T, format string, args ...any) string {
	t.Helper()
	return fmt.Sprintf(format, args...)
}
