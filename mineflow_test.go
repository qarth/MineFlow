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

func TestPatternHelpers(t *testing.T) {
	pattern := NewPrecedencePattern(nil).OneFive()
	if pattern.Size() != 5 {
		t.Fatalf("OneFive size mismatch: got %d want 5", pattern.Size())
	}

	if pattern.Offsets[0].Z != 1 {
		t.Fatalf("expected all offsets to be one bench ahead, got %+v", pattern.Offsets[0])
	}
}

func TestRegular3DBlockModelPatternPrecedence(t *testing.T) {
	blockDef := BlockDefinition{NumX: 2, NumY: 2, NumZ: 2}
	pattern := NewPrecedencePattern([]Vector3I{{0, 0, 1}})
	precedence := NewRegular3DBlockModelPatternPrecedence(blockDef, pattern)

	if precedence.NumBlocks() != 8 {
		t.Fatalf("unexpected block count: got %d want 8", precedence.NumBlocks())
	}

	ants := precedence.Antecedents(blockDef.GridIndex(0, 0, 0))
	if len(ants) != 1 {
		t.Fatalf("expected one antecedent for the first layer, got %d", len(ants))
	}
	if ants[0] != blockDef.GridIndex(0, 0, 1) {
		t.Fatalf("expected antecedent at the next layer, got %d", ants[0])
	}

	if got := precedence.Antecedents(blockDef.GridIndex(0, 0, 1)); len(got) != 0 {
		t.Fatalf("expected no antecedents on the last layer, got %v", got)
	}
}
