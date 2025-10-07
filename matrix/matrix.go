// Package matrix defines the core Matrix interface for linear algebra operations.
//
// What & Why:
//
//	The Matrix interface provides a uniform abstraction over two-dimensional mutable
//	arrays of float64 values, enabling a wide range of mathematical and graph-based
//	algorithms to operate generically on any implementation (e.g., Dense matrices).
//	This design ensures safety through bounds checking and supports deep cloning
//	for immutability guarantees in algorithm pipelines.
//
// Complexity:
//
//	Rows() and Cols() run in O(1) time.
//	At() and Set() perform bounds checking in O(1) time, returning an error on invalid indices.
//	Clone() performs a deep copy in O(rows*cols) time, allocating new storage.
package matrix

//// Matrix represents a two-dimensional mutable array of float64 values.
//// Each method enforces bounds checking and returns clear errors on misuse.
//// Users can implement this interface to provide custom storage layouts.
//type Matrix interface {
//	// Rows returns the number of rows in the matrix.
//	// Complexity: O(1).
//	Rows() int
//
//	// Cols returns the number of columns in the matrix.
//	// Complexity: O(1).
//	Cols() int
//
//	// At retrieves the element at position (i, j).
//	// Returns ErrIndexOutOfBounds if i<0, i>=Rows(), j<0 or j>=Cols().
//	// Complexity: O(1).
//	At(i, j int) (float64, error)
//
//	// Set assigns the value v at position (i, j).
//	// Returns ErrIndexOutOfBounds if indices are invalid.
//	// Complexity: O(1).
//	Set(i, j int, v float64) error
//
//	// Clone returns a deep copy of the matrix.
//	// The returned Matrix is independent of the original.
//	// Complexity: O(rows*cols).
//	Clone() Matrix
//}
