// Package builder contains unit tests for the configuration primitives
// (builderConfig and BuilderOption) to ensure correct application and override behavior.
package builder

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// assertPanics runs f and asserts that it panics with a message containing wantSubstr.
func assertPanics(t *testing.T, f func(), wantSubstr string) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got none", wantSubstr)
			return
		}
		got := fmt.Sprint(r)
		if wantSubstr != "" && !strings.Contains(got, wantSubstr) {
			t.Fatalf("panic mismatch: want substring %q, got %q", wantSubstr, got)
		}
	}()
	f()
}

// TestIDSchemeOptions verifies that ID scheme options are applied in order
// and that nil schemes are ignored (no-op).
func TestIDSchemeOptions(t *testing.T) {
	t.Parallel() // allow this test to run in parallel

	// 1. Default configuration: IDFn should be DefaultIDFn
	cfgDefault := newBuilderConfig()
	// call idFn on a sample index
	if got := cfgDefault.idFn(7); got != "7" {
		t.Errorf("default idFn: expected \"7\", got %q", got)
	}

	// 2. WithSymbolIDs should override to SymbolIDFn
	cfgSymbol := newBuilderConfig(WithSymbolIDs())
	if got := cfgSymbol.idFn(0); got != "A" {
		t.Errorf("WithSymbolIDs: expected \"A\", got %q", got)
	}

	// 3. WithExcelColumnIDs should override to ExcelColumnIDFn
	cfgExcel := newBuilderConfig(WithExcelColumnIDs())
	if got := cfgExcel.idFn(27); got != "AB" {
		t.Errorf("WithExcelColumnIDs: expected \"AB\", got %q", got)
	}

	// 4. WithAlphanumericIDs should override to AlphanumericIDFn
	cfgAlpha := newBuilderConfig(WithAlphanumericIDs())
	if got := cfgAlpha.idFn(35); got != "z" {
		t.Errorf("WithAlphanumericIDs: expected \"z\", got %q", got)
	}

	// 5. WithDefaultIDs after another option should reset to DefaultIDFn
	cfgReset := newBuilderConfig(WithSymbolIDs(), WithDefaultIDs())
	if got := cfgReset.idFn(3); got != "3" {
		t.Errorf("WithDefaultIDs override: expected \"3\", got %q", got)
	}

	// 6. WithIDScheme(nil) MUST panic (fail-fast), not no-op
	assertPanics(t, func() { _ = newBuilderConfig(WithIDScheme(nil)) }, "WithIDScheme(nil)")
}

// TestRNGOptions verifies that RNG options configure the rng field correctly,
// including reproducibility with WithSeed and ignoring nil in WithRand.
func TestRNGOptions(t *testing.T) {
	t.Parallel() // allow parallel execution

	// 1. By default, rng should be nil (deterministic behavior)
	cfgDefault := newBuilderConfig()
	if cfgDefault.rng != nil {
		t.Errorf("default rng: expected nil, got %v", cfgDefault.rng)
	}

	// 2. WithRand should set rng when non-nil
	expRNG := rand.New(rand.NewSource(123))
	cfgWithRand := newBuilderConfig(WithRand(expRNG))
	if cfgWithRand.rng != expRNG {
		t.Errorf("WithRand: expected rng %v, got %v", expRNG, cfgWithRand.rng)
	}

	// 3. WithRand(nil) MUST panic (fail-fast), not no-op
	assertPanics(t, func() { _ = newBuilderConfig(WithRand(nil)) }, "WithRand(nil)")

	// 4. WithSeed should produce reproducible RNG
	cfgSeed1 := newBuilderConfig(WithSeed(42))
	a1 := cfgSeed1.rng.Int63()
	b1 := cfgSeed1.rng.Int63()
	cfgSeed2 := newBuilderConfig(WithSeed(42))
	a2 := cfgSeed2.rng.Int63()
	b2 := cfgSeed2.rng.Int63()
	if a1 != a2 || b1 != b2 {
		t.Errorf("WithSeed reproducibility: got (%d,%d) vs (%d,%d)", a1, b1, a2, b2)
	}
}

// TestWeightFnOptions verifies that weight function options apply correctly,
// override in order, and ignore nil inputs.
func TestWeightFnOptions(t *testing.T) {
	t.Parallel() // allow parallel execution

	const constVal = 9.0
	const min, max = 2.0, 4.0
	rng := rand.New(rand.NewSource(1))

	// 1. Default configuration: weightFn should be DefaultWeightFn
	cfgDefault := newBuilderConfig()
	if w := cfgDefault.weightFn(nil); w != DefaultEdgeWeight {
		t.Errorf("default weightFn(nil): expected %g, got %g", DefaultEdgeWeight, w)
	}

	// 2. WithConstantWeight should override to constant value
	cfgConst := newBuilderConfig(WithConstantWeight(constVal))
	if w := cfgConst.weightFn(nil); w != constVal {
		t.Errorf("WithConstantWeight(nil): expected %g, got %g", constVal, w)
	}
	if w := cfgConst.weightFn(rng); w != constVal {
		t.Errorf("WithConstantWeight(rng): expected %g, got %g", constVal, w)
	}

	// 3. WithUniformWeight should override to uniform sampler
	cfgUni := newBuilderConfig(WithUniformWeight(min, max))
	// nil rng yields default
	if w := cfgUni.weightFn(nil); w != DefaultEdgeWeight {
		t.Errorf("WithUniformWeight(nil rng): expected default %g, got %g", DefaultEdgeWeight, w)
	}
	// seeded rng yields value in [min,max]
	val := cfgUni.weightFn(rng)
	if val < min || val > max {
		t.Errorf("WithUniformWeight(rng): expected in [%g,%g], got %g", min, max, val)
	}

	// 4. Override order: last option wins
	cfgOverride := newBuilderConfig(WithConstantWeight(1), WithUniformWeight(min, max))
	val2 := cfgOverride.weightFn(rng)
	if val2 < min || val2 > max {
		t.Errorf("override order: expected uniform in [%g,%g], got %g", min, max, val2)
	}

	// 5. WithWeightFn(nil) MUST panic (fail-fast)
	assertPanics(t, func() { _ = newBuilderConfig(WithWeightFn(nil)) }, "WithWeightFn(nil)")
}
