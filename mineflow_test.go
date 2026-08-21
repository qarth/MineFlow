package mineflow

import "testing"

func TestSolveUltimatePitExample(t *testing.T) {
	values := []int64{7, 2, -2, -2, -2}
	precedence := [][]int64{{0, 2}, {0, 3}, {1, 3}, {1, 4}}

	got := SolveUltimatePit(values, precedence)
	want := []bool{true, false, true, true, false}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %v want %v", i, got[i], want[i])
		}
	}
}
