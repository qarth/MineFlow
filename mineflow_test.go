package mineflow

import (
	"slices"
	"testing"
)

func TestSolveUltimatePitExample(t *testing.T) {
	values := []int64{7, 2, -2, -2, -2}
	precedence := [][]int64{{0, 2}, {0, 3}, {1, 3}, {1, 4}}

	got, err := SolveUltimatePit(values, precedence)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []bool{true, false, true, true, false}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSolveUltimatePitErrors(t *testing.T) {
	tests := []struct {
		name       string
		values     []int64
		precedence [][]int64
	}{
		{
			name:       "malformed precedence pair",
			values:     []int64{1, -1},
			precedence: [][]int64{{0}},
		},
		{
			name:       "out of range precedence",
			values:     []int64{1, -1},
			precedence: [][]int64{{0, 5}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SolveUltimatePit(tt.values, tt.precedence)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

func TestPatternHelpers(t *testing.T) {
	pattern := OneFivePrecedencePattern()
	if pattern.Size() != 5 {
		t.Fatalf("OneFive size mismatch: got %d want 5", pattern.Size())
	}

	if pattern.Offsets[0].Z != 1 {
		t.Fatalf("expected all offsets to be one bench ahead, got %+v", pattern.Offsets[0])
	}

	ninePattern := OneNinePrecedencePattern()
	if ninePattern.Size() != 9 {
		t.Fatalf("OneNine size mismatch: got %d want 9", ninePattern.Size())
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

func TestInBounds(t *testing.T) {
	b := BlockDefinition{NumX: 3, NumY: 4, NumZ: 5}

	if !b.InBounds(0, 0, 0) {
		t.Fatal("origin should be in bounds")
	}
	if !b.InBounds(2, 3, 4) {
		t.Fatal("max corner should be in bounds")
	}
	if b.InBounds(-1, 0, 0) {
		t.Fatal("negative x should be out of bounds")
	}
	if b.InBounds(3, 0, 0) {
		t.Fatal("x == NumX should be out of bounds")
	}
}

func TestEmptyPit(t *testing.T) {
	// All negative values — nothing should be mined.
	values := []int64{-1, -1, -1}
	got, err := SolveUltimatePit(values, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []bool{false, false, false}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSinglePositiveBlock(t *testing.T) {
	values := []int64{10}
	got, err := SolveUltimatePit(values, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []bool{true}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
