package matrix_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/katalvlaran/lvlath/core"
	"github.com/katalvlaran/lvlath/matrix"
)

func TestToEdgeListAndMatrix(t *testing.T) {
	// Build a directed, weighted graph U→V(7)
	g := core.NewGraph(true, true)
	g.AddEdge("U", "V", 7)

	// 1) ToEdgeList
	elist := matrix.ToEdgeList(g)
	wantList := []matrix.EdgeListItem{{FromID: "U", ToID: "V", Weight: 7}}
	require.Equal(t, wantList, elist)

	// 2) ToMatrix
	m := matrix.ToMatrix(g)
	iU := m.Index["U"]
	iV := m.Index["V"]
	require.Equal(t, int64(7), m.Data[iU][iV])
	// Directed so mirror is zero
	require.Equal(t, int64(0), m.Data[iV][iU])
}

func TestToMatrix_MirrorUndirected(t *testing.T) {
	// Undirected graph A–B(3)
	g := core.NewGraph(false, true)
	g.AddEdge("A", "B", 3)

	m := matrix.ToMatrix(g)
	iA := m.Index["A"]
	iB := m.Index["B"]

	// Mirror entry should also be set
	require.Equal(t, int64(3), m.Data[iA][iB])
	require.Equal(t, int64(3), m.Data[iB][iA])

	// All other cells zero
	for r := range m.Data {
		for c := range m.Data {
			if (r == iA && c == iB) || (r == iB && c == iA) {
				continue
			}
			require.Equal(t, int64(0), m.Data[r][c])
		}
	}
}
