# lvlath Universal Engineering Standard (UES)
```text
lvlath Universal Engineering Standard (UES)
Synchronizing engineering standard for designing, documenting, implementing,
testing, and evolving lvlath packages.

Goal:
- identical quality bar across all packages,
- stable, immutable contracts,
- absolute deterministic behavior,
- auditable and regression-free evolution,
- strong mathematical correctness,
- predictable documentation, memory safety, and API governance.

This standard is not “style only”.
It is a contract discipline, architecture discipline, documentation discipline,
and verification discipline. It establishes not "how it works somehow", but
"how we build it correctly".

===============================================================================
0. Golden Principles of lvlath
===============================================================================

0.1 Contract > implementation
- Public behavior, sentinel errors, determinism, numeric policy, aliasing policy,
  ownership rules, and partial-result semantics are strictly part of the API contract.
- Internal optimization must never silently redefine these rules.

0.2 Determinism by default
- Same input, same explicit options, same package version, same graph/data state
  => same observable result, same structural topology, same ordering, same sentinel 
  classification, and same wrapped failure category.
- If a package is intentionally nondeterministic, that must be explicit, local,
  documented, and test-governed.

0.3 Sentinel-first error model
- `errors.Is` is the primary and canonical classification mechanism.
- Error strings are human-readable diagnostics (UX), not machine protocol. 
  Classifying errors via `strings.Contains` is strictly forbidden.

0.4 One source of truth
- A canonical kernel/algorithm exists in exactly one place.
- Facades validate, apply policy, and delegate.
- Examples demonstrate. Docs explain. Tests protect.
- None of these are allowed to invent alternative semantics or alternative mathematics.

0.5 Explicit policy only
- No hidden heuristics. No hidden fallback modes.
- No “smart fallback” that silently changes mathematics, traversal, shape rules,
  or output semantics.
- Contract-changing behavior must be opt-in via explicit `Option`, `WithXxx`, 
  explicit builder modes, or explicit type selection.

0.6 Tests defend mathematics
- Tests must validate correct math, formalized structures, and the contract.
- Tests must never dictate or coerce the implementation into mathematically wrong behavior.
- A bad test must be fixed, not satisfied by crippling the algorithm.

0.7 Result structures are contracts
- If a package returns a Result-type, its fields, invariants, ownership rules,
  and interpretations under failure are just as much a part of the API as the 
  function signature itself.
- For non-trivial algorithm packages, a named `XxxResult` structure is mandatory.

0.8 Documentation is executable truth, not marketing
- Documentation must describe the real API, real guarantees, real complexity,
  real error protocol, and real limitations.
- `doc.go`, exported GoDoc, `docs/*.md`, examples, and tests must not describe 
  different laws.
- “Convenient lies”, “future methods”, and “rough intuition” are strictly forbidden.

0.9 No hidden operational degradation
- An algorithm might be mathematically correct but engineerically flawed.
- Retention leaks, flaky examples, benchmark setup drift, hidden allocations in 
  hot loops, and silent shape mutations are direct violations of this standard.

0.10 Evolution > rewriting
- No "updating for the sake of updating".
- Any change must either fix a real bug, strengthen the contract/safety/determinism,
  or improve clarity without altering the underlying semantic meaning.

===============================================================================
1. Package Charter (mandatory for every package)
===============================================================================

Each package must begin with a package charter in `doc.go`.
This is not introductory text; it is the package-level constitution.

1.1 Domain Scope (What it does)
Must explicitly fix:
- what domain entities and relations the package introduces,
- what types and structures are the public carriers of the contract,
- what operations/algorithms are mathematically guaranteed,
- what invariants are considered fundamental,
- explicit guarantees on: nil-policy, numeric policy, cancellation, and concurrency.

1.2 Non-goals (What it explicitly does NOT do)
Must explicitly list:
- tasks the package refuses to solve,
- adjacent tasks that belong in neighboring packages,
- "similar but different" algorithms that do not apply here,
- guarantees intentionally omitted (e.g., snapshot-isolation, all-shortest-paths 
  instead of single, strong connectivity instead of weak).

1.3 Contract Laws
The charter must explicitly declare:
- Shape / Dimensions: valid sizes, behavior on zero-values and degenerate cases.
- Numeric / Domain policy: NaN/Inf, +Inf, 0, no edge, no path, loops,
  multi-edges, arithmetic overflow, and explicit separation of storage,
  algorithm-input, policy, result, and dense-representation domains
  according to Section 2.8.
- Determinism law: traversal order, tie-break rules, sorting semantics.
- Error law: which sentinel-errors are returned at which stages.
- Ownership / Aliasing law: who owns memory, where shallow/deep copy occurs.
- Partial Result law: interpretation of results upon cancellation/interruption.

1.4 Explicit Modes
- Any mode that changes the contract must be enabled explicitly via `Option` 
  or builder configuration. No implicit contract switches.

1.5 Result Artifact Law
For every non-trivial algorithm package, the public result must be represented by
an explicit named `XxxResult` structure.

1.5.1 Mandatory rules:
- Typed-value public algorithm returns are forbidden for the canonical algorithm surface.
- Returning raw parallel maps/slices as the primary result contract is forbidden.
- The result type must expose semantically meaningful helper methods whenever the
  package domain needs structured result interaction.
- The result type must document:
  - field invariants,
  - ownership,
  - nil-receiver behavior,
  - interpretation under failure and partial-result policy.
- If the package uses the `core.Nilable` convention, the result type must implement
  its nil-state behavior explicitly and safely.

1.5.2 Examples of acceptable shape:
- BFS `Result`
- DFS `Result`
- `CycleDetectionResult`
- Dijkstra `Result`

1.5.3 Examples of forbidden canonical shape:
- `(map[string]float64, map[string]string, error)`
- `([]int, error)` when the result actually carries multiple domain semantics
- anonymous structs returned from public APIs

1.6 Documentation Layer Separation
A strict boundary must exist:
- `doc.go`: package charter, public laws, invariants.
- Exported GoDoc: per-symbol contract.
- `docs/*.md`: repository-level tutorials, formulas, diagrams, architecture.
`docs/*.md` must not replace `doc.go`, and `doc.go` must not bloat into a tutorial.

===============================================================================
2. Universal Layered Architecture
===============================================================================

Every lvlath package is reasoned about through strict layers. 
Not every package needs every file, but the architecture must remain explicit.

2.1 Storage / Data Layer
Responsible for:
- concrete data containers and memory layout,
- ownership and nil-safety,
- bounds/shape/topology invariants,
- access cost guarantees (O(1), O(n), amortized).

2.2 Policy Layer
Contains:
- numeric policies,
- graph/domain policies (directed vs undirected, weights),
- explicit contract switches.
*Law:* Policy is a part of the contract, not an "optimization".

2.3 Validators / Sanitizers
Responsible for:
- fail-fast validation,
- structural and numeric validation,
- predictable normalization (only if it does not change contract meaning),
- centralized return of sentinel errors.

2.4 Kernels / Algorithms
Contains:
- canonical implementations and real mathematics,
- fixed loop orders,
- deterministic policy applications.
*Law:* The kernel is the single source of truth.

2.5 Facades / API Surface
Contains only:
- input validation,
- options application and finalization,
- delegation to the kernel,
- adaptation of error/result surfaces.
*Law:* Facades must never contain alternative mathematics.

2.5.1 Canonical Facade and Wrapper Law
Every non-trivial lvlath package must expose:

1) a canonical public facade in `api.go`,
2) optionally, a set of wrapper helpers built strictly on top of the canonical facade.

Canonical facade requirements:
- validates required public inputs,
- applies and finalizes options,
- delegates to exactly one kernel,
- publishes the canonical result artifact.

Wrapper requirements:
- wrappers must not introduce alternative mathematics,
- wrappers must not weaken sentinel classification,
- wrappers must not invent hidden defaults that change the contract,
- wrappers must document whether they:
  - discard part of the canonical result,
  - force-enable a mode (e.g. path tracking),
  - perform a point query over the canonical result.

Examples:
- `Dijkstra(...)` is canonical.
- `Distances(...)`, `DistanceTo(...)`, `ShortestPathTo(...)` are wrappers.
- wrappers are allowed only when they reduce caller boilerplate without changing
  the contract semantics.

2.6 Adapters / Builders
Allowed to:
- convert external domain structures (core -> algo, graph -> matrix),
- build local representations.
*Law:* Adapters must explicitly fix policy translation/loss, preserve the 
error protocol, and document the determinism source.

2.7 Executable Documentation Layer
- `example_test.go`, `docs/*.md`, and focused helpers.
- Explains and demonstrates the real surface only. Must never outrun the source.

2.8 Numeric Domain Separation Law

The same float64 value may have different validity and meaning in different
library domains. Numeric semantics must never be transferred implicitly from
one domain into another.

2.8.1 Storage-Domain Values

Storage types must define the exact numeric set they can persist.

Current lvlath/core law:
- core.Graph edge weights must be finite.
- NaN, +Inf, and -Inf are rejected before topology publication.
- Absence of an edge is represented by absence of topology, not by an infinite
  stored edge weight.

2.8.2 Algorithm-Input Values

Every algorithm must define its accepted subset of the storage domain.

Examples:
- Dijkstra accepts finite non-negative costs.
- Flow accepts finite non-negative capacities under its tolerance policy.
- MST accepts finite weights according to its ordering contract.
- Algorithms that support negative weights must declare that support explicitly.

An algorithm may impose a stricter numeric domain than its storage layer.
It must not silently widen the storage domain.

2.8.3 Arithmetic-Domain Values

Individually valid finite operands may produce an unrepresentable intermediate
or final value.

Mandatory rules:
- arithmetic overflow must be detected when it can affect the public result,
- overflow must receive a distinct sentinel when callers need to branch on it,
- an overflowed value must not be silently converted into an unrelated domain
  state such as “unreachable”,
- partial-result publication must follow the package's declared failure law.

Example:
- finite current distance + finite edge weight producing +Inf is arithmetic
  overflow, not an unreachable shortest-path result.

2.8.4 Result-Domain Sentinels

Algorithm results may use numeric sentinels when their meaning is explicit,
documented, and test-governed.

Examples:
- +Inf may represent a known but unreachable shortest-path destination.
- A missing or unknown entity must remain a separate protocol state and must not
  be collapsed into the same numeric sentinel without an explicit contract.

2.8.5 Policy-Domain Sentinels

Options may use numeric sentinels to express unbounded policy.

Examples:
- MaxDistance=+Inf may mean “no distance cutoff”.
- InfEdgeThreshold=+Inf may mean “no finite edge is blocked”.

A policy-domain +Inf value must not authorize +Inf in storage-domain input.

2.8.6 Dense-Representation Sentinels

Dense structures may require a stored value for every index pair.

In such representations:
- +Inf may represent no direct edge,
- +Inf may represent an unreachable pair,
- the exact interpretation must be declared by the matrix or adapter contract.

When converting between dense and graph storage:
- +Inf absence cells must become absent edges,
- they must not become core.Graph edges with +Inf weight.

2.8.7 Hard Prohibitions

The following are forbidden:
- silently reinterpreting result-domain +Inf as an edge weight,
- silently reinterpreting policy-domain +Inf as storage permission,
- translating arithmetic overflow into ordinary unreachable state,
- losing numeric-domain meaning during graph/matrix/adapter conversion,
- documenting one numeric domain while implementing another.

2.8.8 Verification Law

Every package using numeric sentinels must test:
- valid boundary values,
- NaN,
- +Inf,
- -Inf,
- finite negative values where relevant,
- arithmetic overflow where relevant,
- domain conversion and sentinel preservation where adapters exist.

Documentation, tests, examples, and implementation must describe the same
numeric-domain boundaries.

Hard Prohibitions:
- Algorithm kernels depending on external domains without an explicit adapter layer.
- Fast-paths changing behavior, ordering, or error classes.
- Hidden nondeterminism (map iteration, races, randomization) leaking into outputs.
- Having two different kernels for the exact same mathematics.

===============================================================================
3. Public Surface Synchronization Law
===============================================================================

This is a hard law.

3.1 Public surface must be synchronized across:
- source code,
- `doc.go`,
- repository docs (`docs/*.md`),
- examples,
- tests,
- benchmarks.

3.2 If an exported function/type/method is documented, it must exist.

3.3 If a public function/type/method exists, its contract must be documented.

3.4 Repository docs must never mention:
- future APIs,
- hypothetical helpers,
- old signatures,
- removed behaviors,
- stronger guarantees than the current source provides.
- primary usage patterns that bypass the canonical result surface when such a
  result surface exists.

3.5 Compatibility Aliases
If a backward-compatible alias exists, the docs must explicitly state:
- the canonical name,
- the compatibility alias,
- whether the alias is legacy-only or maintained as first-class.

===============================================================================
4. Determinism Governance
===============================================================================

Determinism is a package-level law, not an accidental property.

4.1 Source-of-Order Law
Every package returning ordered results must document the exact order source:
- `NeighborIDs(u)` scan order,
- `Edges()` or `Vertices()` lexical order,
- matrix row-major order,
- stable sort by ID,
- insertion/input scan order.
It is insufficient to say "deterministic". The exact tie-break source must be named.

4.2 Traversal Relation Law
For algorithmic packages over graphs, the package charter must explicitly state
which relation the algorithm consumes (e.g., neighbor IDs, directed outgoing edges,
weak relations). This is load-bearing and must not be inferred loosely.

4.3 Conflict Policy
For ambiguous cases, every package must fix one explicit rule:
- multi-edge: reject / first-wins / last-wins / merge,
- duplicates: reject / merge / ignore,
- tie-break: stable input order / stable sorted order,
- forest root selection: explicit source and order,
- path parent selection: first discovered / lowest ID / first scanned.

4.4 No Magical Accelerations
A fast-path is valid only if it yields:
- bit-equivalent or contract-equivalent output,
- identical sentinel classification,
- identical numeric/domain policy,
- identical tie-break behavior.
If a fast-path changes the contract surface, it is forbidden.

4.5 Deterministic Forest / Multi-Phase Traversal Law
If an algorithm executes in multiple phases (start component + full traversal, 
build + scan, snapshot + export), each phase must explicitly fix:
- source of order and root selection,
- the exact meaning of the intermediate state,
- deterministic composition of the final output.

4.6 Stable output vs invariant-only output
Packages must distinguish exact deterministic output from invariant-only correctness.
Tests, examples, docs, and helpers must follow that distinction exactly.

===============================================================================
5. Error Governance (sentinel-first)
===============================================================================

5.1 Basic Rules
- Every failure class gets its own sentinel: `var ErrXxx = errors.New(...)`.
- Callers and tests classify through `errors.Is`.
- Strings are never protocol. Classification via `strings.Contains` is forbidden.
- Wrapping must preserve sentinel identity.

5.2 Wrap Law
Allowed forms:
- `fmt.Errorf("op: %w", ErrXxx)`
- `fmt.Errorf("Type.Method(%q): %w", id, ErrXxx)`

5.3 Multi-layer Error Preservation Law
If an algorithm/adapter introduces its own sentinel for a lower-level failure, 
it must preserve both:
1. its own algorithmic level of classification,
2. the original underlying root cause.
Mandatory mechanisms: double-wrap (`%w + %w`), `errors.Join`, or an equivalent 
strategy that guarantees `errors.Is` resolves both levels.

5.4 Minimum Universal Set
Common packages should define their required subset of:
- `ErrNil*`
- `ErrOutOfRange`
- `ErrInvalidDimensions` / `ErrBadShape`
- `ErrDimensionMismatch`
- `ErrNaNInf`

5.5 Domain-specific Errors
Must be explicitly defined if needed (e.g., `ErrSingular`, `ErrNotSymmetric`, 
`ErrInvalidWeight`, `ErrUnknownVertex`, `ErrNoPath`, `ErrMixedEdgesNotAllowed`).

===============================================================================
6. Options Pattern Governance
===============================================================================

6.1 Principal Law
Options change the contract only explicitly. No hidden defaults that change math.

6.2 Canonical Signature and Assembly
Recommended signature: `type Option func(*options) error`.
Assembly must be centralized via a helper like `applyOptions(opts ...Option) (options, error)` or `finalizeOptions`.
This helper must:
- return an error on a `nil` option,
- return an error on a `nil` callback (if mandatory),
- compute derived invariants centrally,
- validate all options *before* any heavy allocations occur.
- enforce "last-writer-wins" unless explicitly documented otherwise.
- If Option is publicly constructible, or if the options fields are publicly
  mutable, built-in option constructors are not a sufficient validation barrier.
  Canonical assembly must revalidate the resulting complete configuration after
  every option before the state reaches a kernel.
- Test-only production exports that expose private option assembly are forbidden.
  External tests must use the public facade or directly test public option values.

6.2.1 No-Panic Options Law
Public option setters and public option assembly must never use panic for
ordinary contract validation.

Mandatory rule:
- invalid option input must return an error,
- invalid callback configuration must return an error,
- nil option values must return an error,
- panic is reserved only for unrecoverable internal invariants that cannot be
  triggered by valid public use.

In lvlath, option validation is part of the public contract surface and must
participate in the sentinel-first error protocol.

6.3 Two-Level Option Model
Separate numeric policy options (tolerances, thresholds) from domain policy options 
(directed, weighted, loops, metric closure).

6.4 Observer Hook Law
If an option registers a hook or callback:
- The hook is an observer by default. It must not mutate structures unless declared.
- Hook errors must fit into the general error protocol.
- Hook semantics must be tightly bound to an explicit stage (enqueue, dequeue, visit).

===============================================================================
7. Naming & File Layout Standard
===============================================================================

7.1 Naming
Exported:
- Functions: `VerbNoun`, `NewXxx`, `NormalizeXxx`
- Types: `Noun`
- Errors: `ErrNoun`
- Options: `WithXxx`, `WithAllowXxx`, `WithNoXxx`
Unexported:
- Helpers: short, but domain-accurate.
- Stage names (`stage1Validate`, `enqueueChild`) are allowed only if they genuinely 
  structure an objectively thick method.

7.2 File Layout
Preferred isolation (one major concern per file), with mandatory files for
non-trivial packages:

Mandatory for algorithm / domain packages:
- `api.go`               (canonical public facades and wrappers)
- `bench_test.go`        (benchmark governance)
- `doc.go`               (package charter)
- `errors.go`            (single source of truth for sentinels)
- `example_test.go`      (stable package examples)
- `impl_*.go`            (kernels / algorithms)
- `imp_*_test.go`        (contract tests)
- `test_helpers_test.go` (explicit testing helpers)
- `types.go`             (results, interfaces, package-level types)
- `docs/*.md`            (repository tutorial/specification)

Strongly recommended where applicable:
- `helpers.go`           (internal shared logic)
- `options.go`           (options, defaults, application logic)
- `validators.go`        (structural/numeric validation)
- `*_test.go`            (others contract tests)

Law:
- `api.go` is mandatory for every non-trivial package.
- Public algorithm entry points must not be scattered arbitrarily across impl files.
- If a package is truly tiny and has no meaningful facade/kernel separation,
  omission must be explicitly justified and remain exceptional.

7.3 Constants Law
No magic unnamed values if the value:
- affects the contract or interpretation,
- dictates algorithm stages,
- participates in capacity/preallocation/threshold policies.

===============================================================================
8. GoDoc Contract Template
===============================================================================

Every exported function, method, type, and behavior-bearing helper must use the 
following structured blocks. Do not write filler, but leave zero ambiguity.

Required block model:
- What (What it does)
- Why (Practical purpose)
- Implementation (Real code stages, not fake theory)
- Behavior highlights (Key traits)
- Inputs (Validation rules)
- Returns (Ownership, aliasing, partial result status)
- Errors (Sentinel classes and origins)
- Determinism (Exact order source and tie-breaks)
- Complexity (Honest, current algorithmic complexity)
- Notes (Edge cases)
- Nilability (if receiver or result can be nil)
- AI-Hints (Targeted warnings for IDEs/LLMs to prevent misuse)

===============================================================================
9. Concurrency Governance
===============================================================================

9.1 Lock Taxonomy
Any package with internal mutable state must explicitly document in `doc.go`:
- what data each lock protects,
- canonical lock order,
- which methods hold which locks.

9.2 Linearizability Categories
Public methods relevant to concurrency must be classifiable as:
- `StrictSnapshot`
- `BestEffortSnapshot`
- `Mutation / Transactional`
- `Unsupported under concurrent mutation`

9.3 Mutator Atomicity Law
If a package claims thread-safe mutation, operations that alter linked invariants 
must be topologically and structurally atomic. If atomicity cannot be guaranteed, 
concurrent mutation must be explicitly declared as unsupported.

9.4 Algorithm-over-Mutable-Data Law
Algorithmic packages over structures like `core.Graph` must clearly state:
- whether they read progressively or via snapshot,
- whether concurrent mutation is forbidden,
- whether partial results are possible during topology drift.

9.5 Forbidden Testing Patterns
Never call `t.Fatal`, `t.FailNow`, or equivalent from a non-owner worker goroutine. 
Use `errCh`, `doneCh`, or deterministic aggregation.

===============================================================================
10. Aliasing & Ownership Governance
===============================================================================

10.1 Ownership Law
If a public structure exposes slices, maps, pointers, or view metadata, docs must state:
- shallow vs deep copy (never hide shallow copies behind the generic word "copy"),
- who owns the returned memory,
- whether mutation is safe or forbidden.

10.2 Result Ownership Law
For algorithm `Result` objects, docs must explicitly state:
- whether the caller owns the returned slice/map after return,
- whether package-side retention exists,
- whether the result is snapshot-like (detached) or live-like (linked).

10.3 View Law
If a package exposes views, it must fix:
- copy vs alias,
- mutable vs read-only by convention,
- what is preserved and what is transformed,
- parent lifetime dependency.

10.4 Nilable Result Law
If a public result type can meaningfully appear as a nil pointer receiver,
its methods must remain safe and classify nil access explicitly.

Mandatory rules:
- nil receiver methods must not panic,
- nil receiver behavior must be documented,
- nil result access must return a distinct sentinel when the package defines one,
- tests must explicitly anchor nil-result behavior.

This law is especially important for algorithm result types with helper methods
such as:
- `DistanceTo`
- `HasPathTo`
- `PathTo`
- `Clone`

===============================================================================
11. Public Result & Wrapper Governance
===============================================================================

11.1 Canonical Result Surface
Every non-trivial algorithm package must expose one canonical result artifact
(`XxxResult`) as the primary interaction surface.

11.2 Wrapper Honesty Law
Wrapper helpers are allowed only if they are semantically honest projections of
the canonical result surface.

They may:
- reduce boilerplate,
- force-enable a necessary mode explicitly,
- discard unused parts of the canonical result.

They must not:
- introduce alternative mathematics,
- silently change determinism,
- weaken error classification,
- invent fallback modes.

11.3 Result Helper Methods
Result helper methods should be introduced when they materially improve
correctness, ergonomics, or clarity.

Typical examples:
- `DistanceTo`
- `HasPathTo`
- `PathTo`
- `Clone`

11.4 Nil-Safe Result Methods
If result methods can be called on nil receivers in real code paths, nil-safe
classification must be explicit and tested.

11.5 Typed-Value Prohibition for Canonical Algorithms
The canonical public algorithm facade must not return typed-value bundles such as:
- multiple parallel maps,
- loosely related slices,
- anonymous structs.

Those are allowed only internally or in narrow wrappers, never as the primary
contract for a non-trivial package.

===============================================================================
12. Deterministic ID Governance
===============================================================================

For identity-heavy packages, the following must be documented:
1) ID format
2) Monotonicity law
3) Collision policy (reject, override, merge)
4) Sorting semantics (numeric vs lexicographic)
5) Round-trip preservation policy (serialization parity)
6) Counter bump policy on user-supplied IDs

===============================================================================
13. Thick Methods Standard
===============================================================================

Every thick algorithmic or pipeline method must follow an auditable stage model.

13.1 Stage 1: Validate & Normalize
- Early-return on nil/invalid.
- Validate shape, mode, options.
- Normalize only if normalization is contract-safe.

13.2 Stage 2: Assemble Policy & Allocate Once
- Resolve options and freeze runtime policy.
- Prepare reusable buffers once.
- Preallocate capacities honestly (`make(..., cap)`).
- No hidden allocations in hot loops.

13.3 Stage 3: Core Loops
- Fixed loop order based on the determinism law.
- Deterministic policy application.
- No hidden sorting in hot loops.
- Fast-path execution only if semantics are 100% equivalent.

13.4 Stage 4: Finalize & Post-checks
- Enforce final invariants.
- Execute final sorts, reversals, or canonicalizations.
- Attach metadata, counts, flags, and anchoring properties.

13.5 Stage 5: Publish Result
- Return public result type.
- Preserve or clear partial results exactly as the contract requires.

13.6 No-Magic Local State Law
Thresholds, root depths, capacity hints, and sentinel constants must be named variables.

===============================================================================
14. Partial Result Governance
===============================================================================

If a package returns a partial result + error, it must document this strictly.

14.1 Partial-Result Law
Must specify:
- whether partial result is returned on cancellation,
- whether partial result is returned on runtime error,
- which fields remain authoritatively valid,
- which fields are explicitly cleared or invalidated.

14.2 Testing Partial Results
Tests must anchor partial-result semantics. Every retained or cleared field must 
be governed by explicit assertions.

14.3 Documentation Strictness
Docs and examples must never imply full success semantics on a partial result.

===============================================================================
15. Pseudocode and Visualization Honesty Law
===============================================================================

Pseudocode, formulas, and diagrams in docs are allowed only if faithful to the contract.

15.1 Pseudocode Restrictions
Pseudocode must not:
- invent non-existent APIs,
- move parent/metadata assignment to the wrong semantic stage,
- omit critical policy gates,
- imply exhaustive enumeration where only witness discovery exists.

15.2 Formula Law
Formulas are included only if they genuinely describe the implemented semantics and 
help understand the contract. "Formulas for the sake of quantity" are forbidden.

15.3 Visualization Law
ASCII, Mermaid, or diagram sections must genuinely clarify the architecture or contract. 
Decorative garbage is strictly forbidden.

===============================================================================
16. Test Governance
===============================================================================

16.1 Minimum Contract Grid
Every contract must be covered by:
- Validation group: nil, bad shape, missing input, policy violations.
- Medium group: real math on non-trivial data, correct structure, deterministic output.
- Special group: edge cases, forest/mixed modes, cancellation, multi-layer error 
  preservation, partial results.

16.2 Fast-path / Fallback Coverage
If a fast-path exists, tests must hit both the fast-path and the fallback, explicitly 
proving contract equivalence.

16.3 Math-First Rule
Tests protect correct math. Tests must never coerce the implementation into incorrect 
semantics to satisfy a poorly written assertion.

16.4 Error Testing Law
Classification must use `errors.Is`. Protocol checking via `strings.Contains` is banned.

16.5 Testing Helpers Law
Every non-trivial package must provide a dedicated `test_helpers_test.go`.

Purpose:
- eliminate any need for reflection-heavy or assertion-framework-heavy testing,
- provide precise, contract-oriented failure messages,
- centralize repeated protocol checks for result types, sentinels, nil-state,
  path/order/result comparisons, and numeric assertions.

Mandatory baseline helper set (adapt and extend per package):
- `mustErrorIs(t *testing.T, err error, target error)`
- `mustNilState(t *testing.T, value any, wantNil bool, op string)`
- `isNilLike(value any) bool`
- `mustEqualBool(t *testing.T, got, want bool, op string)`
- `mustEqualInt(t *testing.T, got, want int, op string)`
- `mustEqualString(t *testing.T, got, want string, op string)`

Additional helpers must be introduced where the domain needs them, for example:
- numeric comparisons,
- stable path/order checks,
- structured result checks,
- witness/cycle comparison,
- map-domain assertions.

Hard law:
- `reflect.DeepEqual` must not be used where explicit structural helpers provide
  stronger contract defense and clearer failure messages.
- external assertion frameworks such as `testify` are forbidden in UES-grade packages.

16.6 Regression Anchors
Every fixed bug must receive a explicitly named regression test anchor.

===============================================================================
17. Benchmark Governance
===============================================================================

17.1 Setup Integrity Law
All graph/data building, parsing, and setup must complete before `b.ResetTimer()`.

16.2 Shape Integrity Law
The benchmark must measure exactly what is claimed. If topology policies (e.g., 
duplicate rejection, multi-edge constraints) silently shrink the input shape, the 
benchmark must either regenerate valid input or honestly document the reduced payload.

17.2.1 Algorithmic Regime Law
Every benchmark must answer a concrete question:

"What algorithmic regime am I loading?"

Examples of valid regimes:
- sparse chain traversal,
- dense local competition,
- mixed-edge endpoint resolution,
- cutoff policy overhead,
- path-tracking overhead,
- forest traversal restart logic,
- cycle witness explosion control.

Mandatory rules:
- each benchmark name must correspond to a meaningful topology or policy regime,
- builder helpers are allowed only when they encode reusable topology semantics,
  not when they merely hide arbitrary setup noise,
- a benchmark that does not reveal a concrete algorithmic regime should be removed.

Benchmark suites for algorithmic packages should prefer a small number of
high-value, shape-driven regimes over a large number of decorative or redundant cases.

17.3 Hot Loop Purity Law
Inside the `b.N` loop:
- execute only the measured call,
- perform minimal error checking,
- no `fmt`, no hidden conversions, no benchmark scaffolding allocations.
- `b.ReportAllocs()` is mandatory for core and algorithmic packages.

17.3.1 Setup Failure Integrity Law
Benchmark setup must fail fast and explicitly on topology-construction errors.

Mandatory rules:
- setup errors must never be ignored,
- benchmark fixtures must not silently degrade shape because of rejected inserts,
- if graph policy rejects part of the generated payload, the benchmark must either:
  - regenerate until the intended shape is achieved, or
  - document the reduced shape honestly and explicitly.

17.4 Seed and Reproducibility
If random generation is used, a fixed seed is mandatory. The input shape must be 
100% reproducible across runs.

17.5 Error Branch Isolation
Benchmarks must guarantee they do not accidentally measure an erroneous branch 
(e.g., failing fast on setup data).

===============================================================================
18. Examples Governance
===============================================================================

18.1 Examples must compile, run against the real API, and demonstrate pipeline usage 
(setup -> invoke -> consume). Fake "future methods" are forbidden.

18.1.1 Mandatory Heavy Examples Law for Algorithm Packages
Every non-trivial algorithm package must provide a core package-example suite in
`example_test.go` built around practical, scenario-driven pipelines.

For algorithm packages, the default expectation is:
- at least five meaningful package examples,
- each example demonstrates a distinct contract regime or practical domain story,
- each example follows:
  `build -> algorithm -> consume`

Strong default rules:
- examples should use at least 12 edges / relations unless the example is a
  deliberately minimal law-demo,
- examples must have a real operational story, not a toy triangle unless the
  purpose is a narrowly scoped law demonstration,
- examples must use the public API meaningfully, not merely call functions “for show”,
- printed output must be fully deterministic,
- examples must not imply stronger guarantees than the package contract provides.

Minimal-law examples are allowed, but they do not replace the mandatory heavy scenario set.

18.2 Stability Law
Examples must not be flaky.
- `// Output:` must be perfectly deterministic. If order varies by contract, the 
  example must print invariant-only output.
- Time-based flaky cancellation (timeouts) that risk output drift is forbidden. 
  Use deterministic logical cancellation instead.

18.3 Package Examples vs Scenario Examples
Strict separation:
- `example_test.go`: stable, package-level contract and pipeline examples.
- `examples/*.go`: powerful real-world scenarios, rich domain processes, and 
  composition patterns demonstrating business value.

===============================================================================
19. Repository Documentation Governance (`docs/*.md`)
===============================================================================

Repository docs are the public teaching contract. They must be stronger than a README 
but must not blindly duplicate `doc.go`.

19.1 Synchronization Law
`doc.go`, GoDoc, tests, examples, and `docs/*.md` must describe identical semantics. 
No fictional APIs or speculative helpers.

19.2 Repository Markdown Header Law
Every fundamental `docs/*.md` file must begin with a repository-comment header fixing:
- file name,
- package reference,
- purpose,
- contract status,
- scope,
- license.

19.3 Complexity Honesty Law
Complexity claims in repository docs must reflect the current real algorithm shape. 
Outdated quadratic claims on now-linear code are treated as documentation bugs.

19.4 Example Bridge Law
Docs examples must either be compact but complete, or explicitly point to 
`example_test.go` as the canonical executable source. No ambiguous half-teasers.

===============================================================================
20. Contract Conflict Resolution Procedure
===============================================================================

When a conflict is discovered between behavior, docs, tests, and math:

20.1 Detect the conflict explicitly.
20.2 Determine the true law. Resolution priority:
     Mathematical correctness > Determinism > Contract stability > Honest operational model.
     Do not pick the "easiest patch" if it violates mathematics.
20.3 Lock the law in three places:
     - `doc.go`
     - Per-symbol GoDoc
     - Test regression anchor
20.4 Propagate the fix to all repository docs and examples.
20.5 Record compatibility impact explicitly.

===============================================================================
21. Change Admission Criteria
===============================================================================

A change is explicitly acceptable ONLY if it brings:
- a real bug fix,
- stronger contract clarity,
- better determinism or safety,
- more truthful documentation,
- honest benchmark improvements,
- cleaner ownership boundaries.

A change is strictly REJECTED if it:
- weakens determinism,
- hides errors or alters sentinel classes silently,
- rewrites tests just to pass a broken implementation,
- adds undocumented mode shifts,
- invents API in docs/examples,
- increases ambiguity or introduces retention leaks.

===============================================================================
22. Algorithm Package Charter Addendum
===============================================================================

For algorithms operating over `core.Graph` or equivalent domain structures:

22.1 Graph Policy Declaration
The algorithm must explicitly declare:
- weighted vs unweighted acceptance,
- directed / undirected / mixed policy,
- which relation is consumed (NeighborIDs, Edges, local adjacency),
- whether partial results are supported.

22.2 Deterministic Traversal Law
If returning an `Order`, it must name: root order source, neighbor relation source, 
tie-break rule, and any final canonicalization sort.

22.3 Visit Definition Law
If distinguishing discovery / enqueue / dequeue / process / finalize, the package 
must explicitly choose and define what counts as a "visit".

21.4 Filter Semantics Law
Any applied filter must be strictly classified as: relation-level, edge-level, 
node-level, or component-level.

22.5 Path Reconstruction Law
For `PathTo` or predecessor maps, explicitly define:
- whether it reconstructs one path or all,
- what counts as reachable,
- semantics under forest/multi-root modes.
- Path reconstruction over caller-owned predecessor state must always terminate.
- Broken, cyclic, repeated, empty-before-root, and out-of-domain predecessor
  relations must produce a documented deterministic error.
- No partial witness may be published when predecessor validation fails.
- A path method must not assume that an exported predecessor map remains
  kernel-valid after publication.

22.5.1 Public Path Query Law
If a package supports path reconstruction, public path interaction must be exposed
through the result artifact, not by leaking raw predecessor storage as the primary interface.

Mandatory rules:
- path queries should be performed via methods such as `PathTo`,
- the result type must explicitly classify:
  - unknown target,
  - known unreachable target,
  - tracking-disabled mode,
  - nil-result access,
- public docs and examples must not teach internal predecessor maps as the primary
  user-facing contract when a higher-level result surface exists.

22.6 FullTraversal / Forest Law
Forest modes must define: secondary root ordering, the semantics of Depth and Parent, 
and what remains anchored to the original source versus what becomes per-component.

22.7 Queue / Frontier Memory Law (Retention Prevention)
If the algorithm uses a worklist, queue, or frontier, using slices in the format 
`queue = queue[1:]` is STRICTLY FORBIDDEN as it pins the underlying array and creates 
an operational retention leak during long graph traversals. 
BFS-like packages must use a head-index cursor, a ring buffer, or an equivalent 
safe frontier discipline.

22.8 Witness vs Exhaustive Law
Returned cycles, paths, or proofs must be classified as: exhaustive, representative, 
witness-only, or best-effort.

===============================================================================
23. AI-Hints Governance
===============================================================================

AI-Hints are an operational safety subsystem, not marketing or noise.

23.1 Mandatory Zones
AI-Hints must be present in:
- `doc.go` (package level),
- Exported GoDoc for complex symbols,
- Test/bench/example file headers where misuse risk is high.

23.2.1 Threat-Model Law
Every AI-Hint must defend against a concrete, historically plausible failure mode.

Examples of valid threat models:
- replacing `errors.Is` with string matching,
- collapsing unknown-target and unreachable-target states,
- simplifying mixed-edge traversal to `edge.To`,
- removing heap tie-breaks as a “micro-optimization”,
- reintroducing panic-based option validation,
- publishing partial results where the contract forbids them,
- using `queue = queue[1:]` in long-running frontier algorithms.

Hints without a concrete threat model are noise and must not be added.

23.2 Purpose
AI-Hints must target real failure patterns: wrong edge semantics, string-based 
error parsing, aliasing mistakes, fake optimization refactors by LLMs, and stale pseudocode.

23.3 Prohibition on Noise
AI-Hints are NOT needed where they:
- repeat the obvious,
- do not prevent a real error,
- add noise and dilute meaning.
They must NEVER contradict the source code.

22.3 AI-Hints Audit Law
After any contract or implementation change, an audit is mandatory:
- delete obsolete AI-Hints,
- update changed hints,
- do not spawn useless ones.

===============================================================================
Appendix A - Quality Passport
===============================================================================

A lvlath package is considered fully "UES-grade" when it demonstrates ALL of the following:

- Uncompromising package charter (`doc.go`).
- Perfectly synchronized public surface (Code = Docs = Tests = Examples).
- Exact, named determinism law.
- Strict sentinel-first error protocol.
- Explicit ownership, aliasing, and concurrency laws.
- Centralized options assembly.
- Single Source of Truth kernel implementation.
- Honest, shape-aware benchmarks.
- Contract-first, mathematics-defending tests.
- Repository docs that teach the real package, not a fictional one.

High-Risk Defect Zones (to be relentlessly hunted down):
- Hidden ambiguity in Result types.
- Doc/Code drift.
- Flakiness in examples or tests.
- Silent shape drift in benchmarks.
- Hidden allocations in hot loops.
- Retention memory leaks in queue frontiers.
- Adapter-layers that silently drop contract semantics.

===============================================================================
Appendix B - Anti-Dogma (What NOT to standardize by force)
===============================================================================

UES demands maximum engineering awareness, not maximum blind formalism.
The following actions are strictly forbidden under the guise of "following standard":

- Inserting empty GoDoc sections just for visual symmetry.
- Spawning AI-Hints without a real threat model.
- Introducing a new sentinel error if there is no distinct branching in the protocol.
- Introducing `Option` paradigms if the mode does not actually alter the mathematical contract.
- Splitting code arbitrarily across multiple `impl_*.go` files if the package is tiny 
  and it yields no architectural clarity.

UES exists to eradicate ambiguity and bugs, not to generate bureaucracy.
```