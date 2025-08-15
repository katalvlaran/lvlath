// SPDX-License-Identifier: Apache-2.0
// Package matrix_test contains unit tests for the matrix validators.
package matrix_test

import (
	"errors"
	"testing"

	"github.com/katalvlaran/lvlath/matrix"
	"github.com/stretchr/testify/require"
)

// TestValidateSameShape covers nil inputs, matching and mismatched dimensions.
func TestValidateSameShape(t *testing.T) {
	t.Parallel()

	// helper matrix implementation
	identity := func(r, c int) matrix.Matrix {
		m, err := matrix.NewDense(r, c)
		require.NoError(t, err)
		return m
	}

	tests := []struct {
		name    string
		a, b    matrix.Matrix
		wantErr error
	}{
		{"both nil", nil, nil, matrix.ErrNilMatrix},
		{"first nil", nil, identity(2, 2), matrix.ErrNilMatrix},
		{"second nil", identity(2, 2), nil, matrix.ErrNilMatrix},
		{"equal 2x3", identity(2, 3), identity(2, 3), nil},
		{"row mismatch", identity(2, 3), identity(3, 3), matrix.ErrMatrixDimensionMismatch},
		{"col mismatch", identity(2, 3), identity(2, 4), matrix.ErrMatrixDimensionMismatch},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := matrix.ValidateSameShape(tc.a, tc.b)
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Truef(t, errors.Is(err, tc.wantErr),
					"expected errors.Is(%v, %v)", err, tc.wantErr)
			}
		})
	}
}

// TestValidateSquare covers nil inputs, square and non-square cases.
func TestValidateSquare(t *testing.T) {
	t.Parallel()

	identity := func(n int) matrix.Matrix {
		m, err := matrix.NewDense(n, n)
		require.NoError(t, err)
		return m
	}

	tests := []struct {
		name string
		m    matrix.Matrix
		want error
	}{
		{"nil", nil, matrix.ErrNilMatrix},
		{"1x1", identity(1), nil},
		{"3x3", identity(3), nil},
		{"2x3", func() matrix.Matrix { m, _ := matrix.NewDense(2, 3); return m }(), matrix.ErrMatrixDimensionMismatch},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := matrix.ValidateSquare(tc.m)
			if tc.want == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Truef(t, errors.Is(err, tc.want),
					"expected errors.Is(%v, %v)", err, tc.want)
			}
		})
	}
}
