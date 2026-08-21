package mineflow

import "testing"

// pattern_test.go — port of the C++ pattern tests (mineflow.cpp:3893-3939).

func TestPatternOneFive(t *testing.T) {
	ptrn := PatternOneFive()
	if ptrn.Size() != 5 {
		t.Fatalf("size: got %d want 5", ptrn.Size())
	}

	want := []Vector3I{
		{0, -1, 1}, {-1, 0, 1}, {0, 0, 1}, {1, 0, 1}, {0, 1, 1},
	}
	for i, w := range want {
		if ptrn.Offsets[i] != w {
			t.Fatalf("offset %d: got %+v want %+v", i, ptrn.Offsets[i], w)
		}
	}
}

func TestPatternOneNine(t *testing.T) {
	ptrn := PatternOneNine()
	if ptrn.Size() != 9 {
		t.Fatalf("size: got %d want 9", ptrn.Size())
	}

	k := 0
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			if ptrn.Offsets[k] != (Vector3I{i, j, 1}) {
				t.Fatalf("offset %d: got %+v want %+v", k, ptrn.Offsets[k], Vector3I{i, j, 1})
			}
			k++
		}
	}
}

func TestPatternMinSearch(t *testing.T) {
	ptrn := PatternMinSearchSlope(deg(45), 10)
	if ptrn.Size() != 25 {
		t.Fatalf("size: got %d want 25", ptrn.Size())
	}
}
