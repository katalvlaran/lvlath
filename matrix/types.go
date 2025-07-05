// Package matrix defines configuration options and sentinel errors
// for adjacency and incidence matrix operations.
package matrix

import "errors"

// Sentinel errors for matrix package operations.
var (
	// ErrUnknownVertex indicates a referenced vertex is not present in the matrix.
	ErrUnknownVertex = errors.New("matrix: unknown vertex")

	// ErrDimensionMismatch indicates two matrices have incompatible dimensions for the operation.
	ErrDimensionMismatch = errors.New("matrix: dimension mismatch")

	// ErrNonBinaryIncidence indicates a non-binary entry in an unweighted incidence matrix.
	ErrNonBinaryIncidence = errors.New("matrix: non-binary incidence in unweighted matrix")

	// ErrEigenFailed indicates that eigen decomposition did not converge.
	ErrEigenFailed = errors.New("matrix: eigen decomposition failed")

	// ErrNilGraph indicates that a nil *core.Graph was passed to a matrix constructor.
	ErrNilGraph = errors.New("matrix: graph is nil")

	// ErrNotImplemented signals a placeholder routine.
	ErrNotImplemented = errors.New("matrix: not yet implemented")
)

// MatrixOptions configures how adjacency and incidence matrices are built.
//   - Directed:       treat edges as directed (true) or undirected (false).
//   - Weighted:       preserve edge weights when true; otherwise treat all edges as weight 1.
//   - AllowMulti:     include parallel edges when true; otherwise collapse duplicates.
//   - AllowLoops:     include self-loops when true; otherwise skip them.
//   - MetricClosure:  enables “fill missing edges → Inf + APSP metric closure”.
//
// Use NewMatrixOptions to create with default values and overrides.
type MatrixOptions struct {
	Directed      bool // directed edges
	Weighted      bool // weight preservation
	AllowMulti    bool // parallel edges
	AllowLoops    bool // self-loops
	MetricClosure bool // non-edges → Inf + APSP closure
}

// Option configures a MatrixOptions instance.
type Option func(*MatrixOptions)

// WithDirected returns an Option that sets the Directed field.
func WithDirected(d bool) Option {
	return func(o *MatrixOptions) { o.Directed = d }
}

// WithWeighted returns an Option that sets the Weighted field.
func WithWeighted(w bool) Option {
	return func(o *MatrixOptions) { o.Weighted = w }
}

// WithAllowMulti returns an Option that sets the AllowMulti field.
func WithAllowMulti(m bool) Option {
	return func(o *MatrixOptions) { o.AllowMulti = m }
}

// WithAllowLoops returns an Option that sets the AllowLoops field.
func WithAllowLoops(l bool) Option {
	return func(o *MatrixOptions) { o.AllowLoops = l }
}

// WithMetricClosure returns an Option that sets the MetricClosure field.
func WithMetricClosure(mc bool) Option {
	return func(o *MatrixOptions) { o.MetricClosure = mc }
}

// NewMatrixOptions constructs a MatrixOptions with given Option functions applied.
// Defaults: Directed=false, Weighted=false, AllowMulti=true, AllowLoops=true, MetricClosure=false.
func NewMatrixOptions(opts ...Option) MatrixOptions {
	mo := MatrixOptions{
		Directed:      false,
		Weighted:      false,
		AllowMulti:    true,
		AllowLoops:    true,
		MetricClosure: false,
	}
	for _, opt := range opts {
		opt(&mo)
	}

	return mo
}
