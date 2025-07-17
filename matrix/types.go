// Package matrix defines configuration options and sentinel errors
// for adjacency and incidence matrix operations.
package matrix

import "errors"

// VertexID uniquely identifies a graph vertex.
type VertexID string

// Weight represents an edge weight; must be finite and non-NaN.
type Weight float64

// Sentinel errors for matrix operations.
var (
	// ErrMatrixUnknownVertex indicates that a referenced VertexID is not present.
	ErrMatrixUnknownVertex = errors.New("matrix: unknown vertex")

	// ErrMatrixDimensionMismatch indicates incompatible dimensions.
	ErrMatrixDimensionMismatch = errors.New("matrix: dimension mismatch")

	// ErrMatrixNonBinaryIncidence indicates non-±1 entries in unweighted incidence.
	ErrMatrixNonBinaryIncidence = errors.New("matrix: non-binary incidence")

	// ErrMatrixNilGraph indicates a nil *core.Graph was passed.
	ErrMatrixNilGraph = errors.New("matrix: nil graph")

	// ErrMatrixNotImplemented is a placeholder for future methods.
	ErrMatrixNotImplemented = errors.New("matrix: not implemented")
)

// Default configuration values for MatrixOptions.
const (
	DefaultDirected      = false
	DefaultWeighted      = false
	DefaultAllowMulti    = true
	DefaultAllowLoops    = true
	DefaultMetricClosure = false
)

// MatrixOptions configures graph→matrix transformation.
// Zero value means:
//
//	Directed=false, Weighted=false, AllowMulti=true, AllowLoops=true, MetricClosure=false.
type MatrixOptions struct {
	Directed      bool // directed edges
	Weighted      bool // preserve actual weights
	AllowMulti    bool // include parallel edges
	AllowLoops    bool // include self-loops
	MetricClosure bool // fill missing edges with +Inf and run APSP
}

// MatrixOption configures how adjacency/incidence matrices are built.
// Panics on invalid use (programmer error).
type MatrixOption func(*MatrixOptions)

// WithDirected sets Directed mode.
func WithDirected(d bool) MatrixOption {
	return func(o *MatrixOptions) { o.Directed = d }
}

// WithWeighted sets Weighted mode.
func WithWeighted(w bool) MatrixOption {
	return func(o *MatrixOptions) { o.Weighted = w }
}

// WithAllowMulti sets AllowMulti mode.
func WithAllowMulti(m bool) MatrixOption {
	return func(o *MatrixOptions) { o.AllowMulti = m }
}

// WithAllowLoops sets AllowLoops mode.
func WithAllowLoops(l bool) MatrixOption {
	return func(o *MatrixOptions) { o.AllowLoops = l }
}

// WithMetricClosure sets MetricClosure mode.
func WithMetricClosure(mc bool) MatrixOption {
	return func(o *MatrixOptions) { o.MetricClosure = mc }
}

// NewMatrixOptions returns a MatrixOptions populated with defaults
// and then modified by any provided MatrixOption functions.
func NewMatrixOptions(opts ...MatrixOption) MatrixOptions {
	mo := MatrixOptions{
		Directed:      DefaultDirected,
		Weighted:      DefaultWeighted,
		AllowMulti:    DefaultAllowMulti,
		AllowLoops:    DefaultAllowLoops,
		MetricClosure: DefaultMetricClosure,
	}
	for _, opt := range opts {
		opt(&mo)
	}

	return mo
}
