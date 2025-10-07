// Package matrix provides graph‐aware wrappers over the core Matrix API,
// exposing high‐level methods for incidence‐matrix representations of graphs.
package matrix

//// srcMark is the value placed at the source vertex row in a directed incidence column.
//const srcMark = -1.0
//
//// dstMark is the value placed at the target vertex row in a directed incidence column.
//const dstMark = +1.0
//
//// loopMark is the value placed for a self-loop in undirected incidence.
//const loopMark = +1.0
//
//// IncidenceMatrix wraps a Matrix as a graph incidence representation.
//// VertexIndex maps VertexID → row index in Mat.
//// Edges holds the ordered list of *core.Edge corresponding to columns.
//// Mat holds -1, 0, +1 entries indicating incidence.
//// opts preserves original construction options for round-trip fidelity.
//type IncidenceMatrix struct {
//	Mat         Matrix         // underlying incidence matrix
//	VertexIndex map[string]int // mapping of VertexID to row
//	Edges       []*core.Edge   // slice of edges for each column
//	opts        MatrixOptions  // original construction options
//}

//// NewIncidenceMatrix constructs an IncidenceMatrix from g.
//// Stage 1 (Validate): ensure g is non-nil.
//// Stage 2 (Prepare): extract vertex list and edge list.
//// Stage 3 (Execute): call BuildDenseIncidence.
//// Stage 4 (Finalize): wrap result and return.
//// Returns ErrMatrixNilGraph or any BuildDenseIncidence error.
//func NewIncidenceMatrix(g *core.Graph, opts MatrixOptions) (*IncidenceMatrix, error) {
//	// Stage 1: Validate input graph
//	if g == nil {
//		return nil, ErrMatrixNilGraph
//	}
//
//	// Stage 2: Prepare ordered slices of vertices and edges
//	vertices := g.Vertices() // get vertex IDs in iteration order
//	edges := g.Edges()       // get all graph edges
//
//	// Stage 3: Delegate to low-level builder
//	var (
//		idx  map[string]int // vertex → row index
//		cols []*core.Edge   // filtered, ordered edges
//		mat  *Dense         // resulting incidence matrix
//		err  error          // error placeholder
//	)
//	idx, cols, mat, err = BuildDenseIncidence(vertices, edges, opts)
//	if err != nil {
//		return nil, fmt.Errorf("NewIncidenceMatrix: %w", err)
//	}
//
//	// Stage 4: Wrap and return high-level IncidenceMatrix
//	return &IncidenceMatrix{
//		Mat:         mat,
//		VertexIndex: idx,
//		Edges:       cols,
//		opts:        opts,
//	}, nil
//}

//// VertexCount returns the number of vertices (rows) in the incidence matrix.
//// Complexity: O(1).
//func (im *IncidenceMatrix) VertexCount() int {
//	return im.Mat.Rows()
//}
//
//// EdgeCount returns the number of edges (columns) in the incidence matrix.
//// Complexity: O(1).
//func (im *IncidenceMatrix) EdgeCount() int {
//	return im.Mat.Cols()
//}

//// VertexIncidence returns the incidence row for vertexID.
//// Stage 1 (Validate): lookup vertex index.
//// Stage 2 (Prepare): allocate result slice.
//// Stage 3 (Execute): copy each entry via Mat.At.
//// Stage 4 (Finalize): return slice or error.
//// Returns ErrUnknownVertex if vertexID not found.
//func (im *IncidenceMatrix) VertexIncidence(vertexID string) ([]float64, error) {
//	// Stage 1: Lookup row index
//	row, ok := im.VertexIndex[vertexID]
//	if !ok {
//		return nil, fmt.Errorf("VertexIncidence: unknown vertex %q: %w", vertexID, ErrMatrixUnknownVertex)
//	}
//
//	// Stage 2: Allocate output slice
//	cols := im.Mat.Cols()        // total columns
//	out := make([]float64, cols) // one entry per column
//
//	// Stage 3: Copy entries from underlying matrix
//	var (
//		j   int
//		val float64
//		err error
//	)
//	for j = 0; j < cols; j++ {
//		val, err = im.Mat.At(row, j) // may return ErrIndexOutOfBounds
//		if err != nil {
//			return nil, fmt.Errorf("VertexIncidence: At(%d,%d): %w", row, j, err)
//		}
//		out[j] = val // assign to output
//	}
//
//	// Stage 4: Return result
//	return out, nil
//}

//// EdgeEndpoints returns the source and target IDs for edge column j.
//// Stage 1 (Validate): ensure column index in [0,EdgeCount).
//// Stage 2 (Execute): fetch core.Edge and extract endpoints.
//// Stage 3 (Finalize): return or wrap ErrMatrixDimensionMismatch.
//func (im *IncidenceMatrix) EdgeEndpoints(j int) (fromID, toID string, err error) {
//	// Stage 1: Validate column index
//	if j < 0 || j >= im.EdgeCount() {
//		return "", "",
//			fmt.Errorf("EdgeEndpoints: column %d out of range [0,%d): %w",
//				j, im.EdgeCount(), ErrMatrixDimensionMismatch)
//	}
//
//	// Stage 2: Lookup edge and extract endpoints
//	e := im.Edges[j] // *core.Edge for this column
//
//	// Stage 3: Return from/to IDs
//	return e.From, e.To, nil
//}
