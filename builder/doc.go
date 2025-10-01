// SPDX-License-Identifier: MIT

// Package builder provides deterministic, option-driven “builders” for graphs
// (via lvlath/core) and numeric datasets (slices; optionally consumable by
// lvlath/matrix). The package standardizes configuration, ID schemes, weight
// policies, validation and error signaling so higher-level algorithms stay DRY,
// predictable, and easy to test.
//
// -----------------------------------------------------------------------------
// Design contract (strict)
// -----------------------------------------------------------------------------
//
//   - One public orchestrator:
//     BuildGraph(gopts, bopts, cons...)
//     It creates *core.Graph with gopts, resolves builderConfig from bopts, and
//     applies constructors in order. If any constructor returns an error, the
//     orchestration stops immediately and returns the wrapped error.
//
//   - Constructors are first-class:
//     type Constructor func(g *core.Graph, cfg builderConfig) error
//     Each closure performs a deterministic mutation on g with the resolved cfg.
//     Constructors neither panic nor mutate global state; they return sentinel
//     errors on invalid input or incompatible graph modes.
//
//   - Implementations live in impl_*.go, data registries in *_spec.go, and thin
//     public wrappers (if any) in api.go. This separation keeps responsibilities
//     focused and makes docs easy to locate.
//
//   - Determinism:
//     Identical inputs (options, seed, constructor order, data registries) yield
//     identical outputs. Graph builders document a stable vertex/edge emission order.
//     Sequence builders use rngFrom(cfg, seed) (see sequence_primitives.go) so that
//     a shared cfg.rng (via WithSeed(...)) yields globally consistent randomness;
//     otherwise a local rand.New(rand.NewSource(seed)) is used.
//
//   - Idempotency (graphs):
//     Builders check HasVertex/HasEdge (and for undirected graphs also reversed
//     orientation) before insertion. Re-running the same builder does not create
//     duplicates.
//
//   - Safety & errors:
//     No panics at runtime. Invalid public options should be rejected in option
//     constructors (fail fast). Constructors return wrapped sentinels such as
//     ErrConstructFailed, ErrOptionViolation, ErrBadSize (names per your codebase).
//
// -----------------------------------------------------------------------------
// Configuration (functional options)
// -----------------------------------------------------------------------------
//
//   - BuilderOption mutates an internal builderConfig snapshot (immutable to impls).
//     Typical fields (kept stable across implementations):
//     – rng / seed               • for stochastic builders (rngFrom honors cfg.rng first)
//     – idFn                     • vertex ID scheme for graph builders
//     – leftPrefix/rightPrefix   • bipartite label roots
//     – weightFn                 • edge-weight policy for Weighted() graphs
//     – partition/namespace      • optional scoping utilities for composite IDs
//
//   - ID schemes (examples):
//     DefaultIDFn (0,1,2,…), SymbolIDFn (A,B,…), ExcelColumnIDFn (A..Z,AA..),
//     AlphanumericIDFn (base-36), HexIDFn (hex). Choose per readability needs.
//
//   - Weight policies (examples):
//     DefaultWeightFn (constant DefaultEdgeWeight), ConstantWeightFn(v),
//     UniformWeightFn(min,max), NormalWeightFn(mean,stddev), ExponentialWeightFn(rate).
//
//   - Validation helpers:
//     validateMin, validateProbability, validatePartition, …
//     Prefer helpers over ad-hoc checks for consistent error surfaces.
//
// -----------------------------------------------------------------------------
// Graph builders (declared in api.go, implemented in impl_*.go)
// -----------------------------------------------------------------------------
//
//   - Composition entry-point:
//     g, err := builder.BuildGraph(gopts, bopts, cons...)
//     where cons is a sequence of Constructor closures (e.g., Complete(5), Wheel(8)).
//
//   - Standard families (examples; see api.go for the current list):
//     Cycle(n), Path(n), Star(n), Wheel(n), Complete(n),
//     CompleteBipartite(n1,n2), Grid(r,c),
//     RandomSparse(n,p), RandomRegular(n,d), PlatonicSolid(name,withCenter),
//     Hexagram(variant) — chord overlays over a base ring (Cycle/Wheel).
//
//   - Complexity (typical):
//     Cycle/Path/Star/Wheel/Grid       → O(V+E), memory O(V)
//     Complete/K_{n1,n2}               → O(n^2) edges in undirected mode
//     RandomSparse/Regular             → per docs in api.go; deterministic per seed/options
//
//   - Error model:
//     – Nil constructor passed to BuildGraph → ErrConstructFailed
//     – Incompatible graph modes (e.g., trying to add self-loops when Looped()==false)
//     are handled explicitly by the implementation or delegated to core.AddEdge.
//
// -----------------------------------------------------------------------------
// Glyph/word/number builders (letters_spec.go + impl_letters.go)
// -----------------------------------------------------------------------------
//
//   - Data registry:
//     letters_spec.go is the single source of truth for glyph geometry on a 5×7 grid.
//     Each glyph has canonical IDs "<Glyph>_<Horiz>_<Vert>" and an ordered edge list.
//     JOINER = "_" is the only delimiter; changing it is a breaking change for fixtures.
//
//   - Building logic (impl_letters.go):
//     – Letters(text, scope) Constructor
//     Adds one connected component per glyph with strict idempotency.
//     scope=="" → uses pure canonical IDs; repeated glyphs would collide → ErrOptionViolation.
//     scope!="" → each glyph is namespaced "<scope>::<pos>::<CanonicalID>" for collision-free reuse.
//     – Word(word, scope) is a thin alias over Letters.
//     – Digit(d, scope) and Number(number, decimal, scope) produce numeric glyphs
//     from numberSpec; non-digit runes are ignored unless added to the registry.
//
//   - Directed/Weighted policy:
//     If g.Directed(): undirected semantic edges are mirrored explicitly.
//     If g.Weighted(): default ConstantWeightFn(1) is used; otherwise weight 0.
//
//   - Complexity:
//     Per glyph O(V+E). For k glyphs, O(k·(V+E)). V,E are tiny for 5×7 skeletons.
//
// -----------------------------------------------------------------------------
// Sequence datasets (impl_pulse.go / impl_chirp.go / impl_ohlc.go)
// -----------------------------------------------------------------------------
//
//   - BuildPulse(n, seed, opts...)
//     Rectangular/triangular pulse with optional linear Trend and Gaussian Noise.
//     Deterministic per (n, seed, options). Complexity O(n). Invalid n→nil.
//
//   - BuildAudioChirp(n, seed, opts...)
//     Linear frequency sweep from f0 to f1 with optional Trend and Noise.
//     Deterministic per (n, seed, options). Complexity O(n). Invalid n→nil.
//
//   - BuildOHLCSeries(days, seed, opts...)
//     GBM-like daily OHLC with a fixed number of intraday steps (open=first, close=last,
//     high/low over the intraday path). Invariants by construction:
//     low ≤ min(open,close) ≤ max(open,close) ≤ high
//     Deterministic per (days, seed, options). Complexity O(days*steps). Invalid days→nil slices.
//
//   - Matrix integration (optional):
//     Implementations return slices (zero-allocation ergonomics).
//     If needed, wrap outputs with lvlath/matrix (e.g., DenseFrom(...)) for windowing,
//     resampling, normalization, or feature extraction.
//
// -----------------------------------------------------------------------------
// Determinism, seeds, and golden tests
// -----------------------------------------------------------------------------
//   - All stochastic builders use rngFrom(cfg, seed):
//     if cfg.rng != nil → shared stream (set via WithSeed(...));
//     else → local rand.New(rand.NewSource(seed)).
//     This keeps replayability both for isolated calls and for composed pipelines.
//   - Golden testing advice:
//     – Fix seed=1 and assert first K values of Pulse/Chirp (don’t overfit on entire series).
//     – For OHLC, assert invariants, positivity, and stable length.
//
// -----------------------------------------------------------------------------
// Error signaling & invariants
// -----------------------------------------------------------------------------
//   - Constructors wrap errors with a context tag (e.g., "BuildLetters", "Path") and
//     return sentinels (ErrOptionViolation, ErrConstructFailed, ErrBadSize, …).
//   - Sequence helpers return nil (or nils) for invalid sizes/parameters,
//     keeping dataset call sites simple and panic-free.
//   - Graph idempotency: a builder never inserts an already present vertex/edge.
//   - Directed mirroring: if g.Directed()==true, every semantic undirected edge (u,v)
//     is emitted as (u→v) and (v→u) unless already present.
//
// -----------------------------------------------------------------------------
// Extensibility & file layout
// -----------------------------------------------------------------------------
// • To add a new topology:
//  1. Declare the factory in api.go (docstring + complexity).
//  2. Implement Constructor in a dedicated impl_<topic>.go (no public symbols).
//  3. Put any immutable datasets in a *_spec.go file (stable IDs, stable order).
//  4. Add thin BuildX wrappers in api.go only if you need a convenience call
//     that runs a single constructor against an existing graph.
//
// • To add a new sequence:
//  1. Implement it in its own impl_<name>.go and reuse shared defaults/helpers
//     from sequence_primitives.go (rngFrom, shared constants).
//  2. Keep determinism and validation rules identical to the existing trio.
//  3. If options are required, route BuilderOption → builderConfig → extract*Params.
//
// -----------------------------------------------------------------------------
// Compatibility guarantees
// -----------------------------------------------------------------------------
//   - Canonical glyph IDs and edge emission order are considered fixtures. Do not
//     rename existing IDs or reorder edges without a migration plan and golden updates.
//   - JOINER ("_") and namespace separators ("::") are stable tokens across releases.
//   - Public function signatures in api.go and sequence builders are semver-stable.
//     Internal builderConfig may evolve as long as options remain backward compatible.
//
// -----------------------------------------------------------------------------
// AI-Hints (practical usage)
// -----------------------------------------------------------------------------
//   - Compose deterministically:
//     g, _ := builder.BuildGraph(gopts, bopts,
//     builder.Complete(5),
//     builder.Letters("AB", "logo"),
//     builder.Wheel(8),
//     )
//   - Lock randomness for tests: WithSeed(42) in bopts for graph RNGs; for sequences
//     you can either pass explicit seed or rely on shared cfg.rng the same WithSeed(...).
//   - Human-readable IDs: WithIDScheme(SymbolIDFn) or WithPartitionPrefix("L","R").
//   - Namespacing glyphs: use non-empty scope to reuse the same glyph multiple times
//     without ID collisions: "<scope>::<pos>::<CanonicalID>".
//   - Matrix post-processing: convert sequences to lvlath/matrix.Dense for windowing,
//     FFT, filtering and normalization; keep builder outputs minimal and predictable.
//
// See api.go for the public factory declarations and impl_* files for contracts
// and emission order details per constructor. letters_spec.go documents the glyph
// dataset format and invariants. sequence_primitives.go hosts shared RNG/defaults
// for dataset builders.
package builder
