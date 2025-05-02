package graph

import (
	"testing"
)

func TestToEdgeList(t *testing.T) {
	g := NewGraph(true, true)
	g.AddEdge("U", "V", 7)
	list := g.ToEdgeList()
	want := EdgeListItem{FromID: "U", ToID: "V", Weight: 7}
	if len(list) != 1 || list[0] != want {
		t.Errorf("expected [%v], got %v", want, list)
	}
}

func TestToMatrix(t *testing.T) {
	g := NewGraph(false, true)
	g.AddEdge("A", "B", 1)
	m := g.ToMatrix()
	idxA := m.Index["A"]
	idxB := m.Index["B"]
	if m.Data[idxA][idxB] != 1 {
		t.Errorf("expected weight 1 at [%d][%d], got %d", idxA, idxB, m.Data[idxA][idxB])
	}
	if m.Data[idxB][idxA] != 1 {
		t.Errorf("undirected mirror should set [%d][%d]", idxB, idxA)
	}
	// несуществующие пары должны быть 0
	for i := range m.Data {
		for j := range m.Data {
			if (i != idxA || j != idxB) && (i != idxB || j != idxA) {
				if m.Data[i][j] != 0 {
					t.Errorf("expected zero at [%d][%d], got %d", i, j, m.Data[i][j])
				}
			}
		}
	}
}
