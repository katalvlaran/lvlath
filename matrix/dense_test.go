// Package matrix_test contains unit tests for the Dense implementation
// of the Matrix interface in the matrix package.
package matrix_test

import (
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

// TestNewDenseInvalidDimensions ensures that NewDense rejects non-positive dimensions.
func TestNewDenseInvalidDimensions(t *testing.T) {
	_, err := matrix.NewDense(0, 5)                      // attempt to create with zero rows
	require.ErrorIs(t, err, matrix.ErrInvalidDimensions) // expect ErrInvalidDimensions

	_, err = matrix.NewDense(5, 0)                       // attempt to create with zero columns
	require.ErrorIs(t, err, matrix.ErrInvalidDimensions) // expect ErrInvalidDimensions
}

// TestRowsCols verifies that Rows() and Cols() return correct dimension values.
func TestRowsCols(t *testing.T) {
	rows, cols := 3, 4                    // define expected row and column counts
	m, err := matrix.NewDense(rows, cols) // create a Dense matrix of size 3x4
	require.NoError(t, err)               // assert no error on valid dimensions

	require.Equal(t, rows, m.Rows()) // assert Rows() equals expected rows
	require.Equal(t, cols, m.Cols()) // assert Cols() equals expected cols
}

// TestAtSetOutOfBounds ensures At() and Set() return ErrIndexOutOfBounds on invalid access.
func TestAtSetOutOfBounds(t *testing.T) {
	m, err := matrix.NewDense(2, 2) // create a 2x2 Dense matrix
	require.NoError(t, err)         // assert matrix creation succeeded

	_, err = m.At(-1, 0)                                // attempt At() with negative row index
	require.ErrorIs(t, err, matrix.ErrIndexOutOfBounds) // expect ErrIndexOutOfBounds

	_, err = m.At(0, 2)                                 // attempt At() with column index out of range
	require.ErrorIs(t, err, matrix.ErrIndexOutOfBounds) // expect ErrIndexOutOfBounds

	err = m.Set(2, 0, 1.23)                             // attempt Set() with row index out of range
	require.ErrorIs(t, err, matrix.ErrIndexOutOfBounds) // expect ErrIndexOutOfBounds

	err = m.Set(0, -1, 4.56)                            // attempt Set() with negative column index
	require.ErrorIs(t, err, matrix.ErrIndexOutOfBounds) // expect ErrIndexOutOfBounds
}

// TestSetGet validates correct behavior of Set() followed by At() on valid indices.
func TestSetGet(t *testing.T) {
	m, err := matrix.NewDense(2, 3) // create a 2x3 Dense matrix
	require.NoError(t, err)         // ensure valid creation

	err = m.Set(1, 2, 7.89) // set element at row 1, column 2
	require.NoError(t, err) // assert Set() succeeded

	val, err := m.At(1, 2)      // retrieve the set element
	require.NoError(t, err)     // assert At() succeeded
	require.Equal(t, 7.89, val) // assert retrieved value matches set value
}

// TestCloneIndependence ensures Clone() returns a deep copy that does not share storage.
func TestCloneIndependence(t *testing.T) {
	m, err := matrix.NewDense(2, 2) // create a 2x2 Dense matrix
	require.NoError(t, err)         // validate creation

	// initialize matrix elements to distinct values
	_ = m.Set(0, 0, 1.0)
	_ = m.Set(1, 1, 2.0)

	clone := m.Clone() // clone the matrix

	// modify the clone, but not the original
	_ = clone.Set(0, 0, 3.0)

	origVal, err := m.At(0, 0)     // retrieve original matrix element
	require.NoError(t, err)        // assert At() succeeded on original
	require.Equal(t, 1.0, origVal) // expect original remains unchanged

	cloneVal, err := clone.At(0, 0) // retrieve clone's element
	require.NoError(t, err)         // assert At() succeeded on clone
	require.Equal(t, 3.0, cloneVal) // expect clone reflects new value
}

// TestStringOutput checks that String() formats the matrix as expected.
func TestStringOutput(t *testing.T) {
	m, err := matrix.NewDense(2, 2) // create a 2x2 matrix for formatting test
	require.NoError(t, err)         // ensure valid creation

	// populate matrix with sample values
	_ = m.Set(0, 0, 1)
	_ = m.Set(0, 1, 2)
	_ = m.Set(1, 0, 3)
	_ = m.Set(1, 1, 4)

	expected := "[1, 2]\n[3, 4]\n"         // define expected string output
	require.Equal(t, expected, m.String()) // assert String() output matches expected format
}
