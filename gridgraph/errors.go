package gridgraph

import "errors"

var (
	// ErrEmptyGrid indicates the input 2D slice is empty.
	ErrEmptyGrid = errors.New("gridgraph: input grid must have at least one row and one column")
	// ErrNonRectangular indicates rows of differing lengths.
	ErrNonRectangular = errors.New("gridgraph: all rows must have the same length")
	// ErrComponentIndex indicates a requested component index is invalid.
	ErrComponentIndex = errors.New("gridgraph: component index out of range")
	// ErrNoPath indicates no conversion path exists between two components.
	ErrNoPath = errors.New("gridgraph: no path between specified components")
)
