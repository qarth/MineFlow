package mineflow

import "testing"

// precedence_test.go — port of the C++ precedence tests
// (mineflow.cpp:3680-3772), plus coverage of the 3D pattern classes.

func TestRegular2DGrid45DegreePrecedenceBase(t *testing.T) {
	pre := NewRegular2DGrid45DegreePrecedence(10, 6)
	if pre.NumBlocks() != 60 {
		t.Fatalf("NumBlocks: got %d want 60", pre.NumBlocks())
	}

	ne := 8*5*3 + 1*5*2 + 1*5*2
	if got := NumPrecedenceConstraints(pre); got != ne {
		t.Fatalf("NumPrecedenceConstraints: got %d want %d", got, ne)
	}

	to := AntecedentsSlice(pre, 5)
	if len(to) != 3 || to[0] != 14 || to[1] != 15 || to[2] != 16 {
		t.Fatalf("Antecedents(5): got %v want [14 15 16]", to)
	}

	if !ConsistentPrecedenceConstraints(pre) {
		t.Fatal("expected consistent precedence constraints")
	}
}

func TestRegular2DGrid45DegreePrecedenceOneWide(t *testing.T) {
	pre := NewRegular2DGrid45DegreePrecedence(1, 6)
	if pre.NumBlocks() != 6 {
		t.Fatalf("NumBlocks: got %d want 6", pre.NumBlocks())
	}

	if got := NumPrecedenceConstraints(pre); got != 5 {
		t.Fatalf("NumPrecedenceConstraints: got %d want 5", got)
	}

	to := AntecedentsSlice(pre, 0)
	if len(to) != 1 || to[0] != 1 {
		t.Fatalf("Antecedents(0): got %v want [1]", to)
	}
}

func TestRegular2DGrid45DegreePrecedenceReachableAntecedents(t *testing.T) {
	pre := NewRegular2DGrid45DegreePrecedence(10, 6)

	expected := []int{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 1, 1, 1, 0, 0, 0,
		0, 0, 0, 1, 1, 1, 1, 1, 0, 0,
		0, 0, 1, 1, 1, 1, 1, 1, 1, 0,
		0, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	} // flipped

	actual := make([]int, 60)
	buffer := NewReachableSearchBuffer(pre.NumBlocks())
	for v := range ReachableAntecedents(pre, 5, buffer) {
		actual[v] = 1
	}

	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("index %d: got %d want %d", i, actual[i], expected[i])
		}
	}
}

func TestRegular2DGrid45DegreePrecedenceReachableSuccessors(t *testing.T) {
	pre := NewRegular2DGrid45DegreePrecedence(10, 6)

	expected := []int{
		0, 1, 1, 1, 2, 2, 1, 0, 0, 0,
		0, 0, 1, 1, 1, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	} // flipped

	actual := make([]int, 60)
	buffer := NewReachableSearchBuffer(pre.NumBlocks())
	for v := range ReachableSuccessors(pre, 15, buffer) {
		actual[v]++
	}
	for v := range ReachableSuccessors(pre, 23, buffer) {
		actual[v]++
	}

	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("index %d: got %d want %d", i, actual[i], expected[i])
		}
	}
}

func TestRegular2DGrid45DegreePrecedenceAllConstraints(t *testing.T) {
	pre := NewRegular2DGrid45DegreePrecedence(10, 6)

	count := 0
	for c := range AllConstraints(pre) {
		if c.From >= c.To {
			t.Fatalf("expected From < To, got %d -> %d", c.From, c.To)
		}
		count++
	}
	if count != 140 {
		t.Fatalf("constraint count: got %d want 140", count)
	}
}

func TestRegular3DBlockModelPatternPrecedence(t *testing.T) {
	blockDef := UnitModel(2, 2, 2)
	pre := NewRegular3DBlockModelPatternPrecedence(blockDef, NewPrecedencePattern([]Vector3I{{0, 0, 1}}))

	if pre.NumBlocks() != 8 {
		t.Fatalf("unexpected block count: got %d want 8", pre.NumBlocks())
	}

	ants := AntecedentsSlice(pre, blockDef.GridIndex(0, 0, 0))
	if len(ants) != 1 {
		t.Fatalf("expected one antecedent for the first layer, got %d", len(ants))
	}
	if ants[0] != blockDef.GridIndex(0, 0, 1) {
		t.Fatalf("expected antecedent at the next layer, got %d", ants[0])
	}

	if got := AntecedentsSlice(pre, blockDef.GridIndex(0, 0, 1)); len(got) != 0 {
		t.Fatalf("expected no antecedents on the last layer, got %v", got)
	}
}

// The OneFive pattern over a small 3D model: compare inner-region fast path
// and boundary path against the naive per-block application of the pattern.
func TestRegular3DBlockModelPatternPrecedenceMatchesNaive(t *testing.T) {
	blockDef := UnitModel(6, 5, 4)
	pattern := PatternOneFive()
	pre := NewRegular3DBlockModelPatternPrecedence(blockDef, pattern)

	for from := 0; from < pre.NumBlocks(); from++ {
		x, y, z := blockDef.XYZIndices(from)
		var want []int
		if z < blockDef.NumZ-1 {
			for _, off := range pattern.Offsets {
				cx, cy, cz := x+off.X, y+off.Y, z+off.Z
				if blockDef.InDef(cx, cy, cz) {
					want = append(want, blockDef.GridIndex(cx, cy, cz))
				}
			}
		}

		got := AntecedentsSlice(pre, from)
		if len(got) != len(want) {
			t.Fatalf("Antecedents(%d): got %v want %v", from, got, want)
		}
		gotSet := make(map[int]bool, len(got))
		for _, v := range got {
			gotSet[v] = true
		}
		for _, v := range want {
			if !gotSet[v] {
				t.Fatalf("Antecedents(%d): missing %d (got %v want %v)", from, v, got, want)
			}
		}
	}
}

func TestRegular3DBlockModelKeyedPatternsPrecedence(t *testing.T) {
	blockDef := UnitModel(3, 3, 3)
	patterns := []PrecedencePattern{PatternOneFive(), PatternOneNine()}
	patternIndices := make([]int, blockDef.NumBlocks())
	// Bottom layer uses OneFive, the rest use OneNine.
	for i := blockDef.NumX * blockDef.NumY; i < blockDef.NumBlocks(); i++ {
		patternIndices[i] = 1
	}

	pre := NewRegular3DBlockModelKeyedPatternsPrecedence(blockDef, patterns, patternIndices)
	if pre.NumBlocks() != 27 {
		t.Fatalf("unexpected block count: got %d want 27", pre.NumBlocks())
	}

	// Center block of the bottom layer (OneFive): 5 antecedents.
	center := blockDef.GridIndex(1, 1, 0)
	if got := NumAntecedents(pre, center); got != 5 {
		t.Fatalf("NumAntecedents(center): got %d want 5", got)
	}
	// Center block of the middle layer (OneNine): 9 antecedents.
	mid := blockDef.GridIndex(1, 1, 1)
	if got := NumAntecedents(pre, mid); got != 9 {
		t.Fatalf("NumAntecedents(mid): got %d want 9", got)
	}
}

func TestExplicitPrecedenceBasics(t *testing.T) {
	pre := NewExplicitPrecedence(5)
	for _, pair := range [][2]int{{0, 2}, {0, 3}, {1, 3}, {1, 4}} {
		if err := pre.AddConstraint(pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}

	if got := NumPrecedenceConstraints(pre); got != 4 {
		t.Fatalf("NumPrecedenceConstraints: got %d want 4", got)
	}

	// Successors falls back to a full scan (ExplicitPrecedence does not
	// implement SuccessorsProvider).
	succ := SuccessorsSlice(pre, 3)
	if len(succ) != 2 {
		t.Fatalf("Successors(3): got %v want 2 entries", succ)
	}
}
