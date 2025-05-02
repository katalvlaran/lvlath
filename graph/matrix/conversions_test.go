package matrix

import (
	"reflect"
	"testing"

	"lvlath/graph/core"
)

func TestToEdgeList(t *testing.T) {
	g := core.NewGraph(true, true)
	g.AddEdge("U", "V", 7)
	list := ToEdgeList(g)
	want := []EdgeListItem{{FromID: "U", ToID: "V", Weight: 7}}
	if !reflect.DeepEqual(list, want) {
		t.Errorf("expected %v, got %v", want, list)
	}
}

func TestToMatrix(t *testing.T) {
	g := core.NewGraph(false, true)
	g.AddEdge("A", "B", 1)
	m := ToMatrix(g)
	iA := m.Index["A"]
	iB := m.Index["B"]

	if m.Data[iA][iB] != 1 {
		t.Errorf("expected weight 1 at [%d][%d], got %d", iA, iB, m.Data[iA][iB])
	}
	if m.Data[iB][iA] != 1 {
		t.Errorf("expected mirror weight at [%d][%d], got %d", iB, iA, m.Data[iB][iA])
	}
	// all other entries must be zero
	for r := range m.Data {
		for c := range m.Data {
			if (r != iA || c != iB) && (r != iB || c != iA) {
				if m.Data[r][c] != 0 {
					t.Errorf("expected zero at [%d][%d], got %d", r, c, m.Data[r][c])
				}
			}
		}
	}
}
