package mineflow

import "testing"

// solver_test.go — port of the C++ TEST(MFlow, LargestMinCut*) suites
// (mineflow.cpp:3955-3998). The data-file tests live in golden_data_test.go.

func TestLargestMinCutTiny(t *testing.T) {
	values := SliceBlockValues{7, 2, -2, -2, -2}

	pre := NewExplicitPrecedence(values.NumBlocks())
	for _, pair := range [][2]int{{0, 2}, {0, 3}, {1, 3}, {1, 4}} {
		if err := pre.AddConstraint(pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}

	solver, err := NewPseudoSolver(pre, values)
	if err != nil {
		t.Fatal(err)
	}
	info, err := solver.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if info.NumContainedNodes != 3 {
		t.Fatalf("NumContainedNodes: got %d want 3", info.NumContainedNodes)
	}

	values2 := NewSolveLargestValuesAdapter(values)
	solver2, err := NewPseudoSolver(pre, values2)
	if err != nil {
		t.Fatal(err)
	}
	info2, err := solver2.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if info2.NumContainedNodes != 5 {
		t.Fatalf("NumContainedNodes (largest): got %d want 5", info2.NumContainedNodes)
	}
}

func TestLargestMinCutMMW(t *testing.T) {
	values := SliceBlockValues{3, 8, 1, -2, -2, -2, -2, 0, 0, 0, 0, 0}

	pre := NewExplicitPrecedence(values.NumBlocks())
	for _, pair := range [][2]int{
		{0, 3}, {0, 4}, {1, 4}, {1, 5}, {2, 5}, {2, 6},
		{7, 0}, {7, 1}, {8, 1}, {8, 2}, {9, 3}, {9, 4},
		{10, 4}, {10, 5}, {11, 5}, {11, 6},
	} {
		if err := pre.AddConstraint(pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}

	solver, err := NewPseudoSolver(pre, values)
	if err != nil {
		t.Fatal(err)
	}
	info, err := solver.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if info.NumContainedNodes != 5 {
		t.Fatalf("NumContainedNodes: got %d want 5", info.NumContainedNodes)
	}

	values2 := NewSolveLargestValuesAdapter(values)
	solver2, err := NewPseudoSolver(pre, values2)
	if err != nil {
		t.Fatal(err)
	}
	info2, err := solver2.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if info2.NumContainedNodes != 8 {
		t.Fatalf("NumContainedNodes (largest): got %d want 8", info2.NumContainedNodes)
	}
}

func TestPseudoSolverUpdateValues(t *testing.T) {
	pre := NewExplicitPrecedence(2)
	if err := pre.AddConstraint(0, 1); err != nil {
		t.Fatal(err)
	}

	solver, err := NewPseudoSolver(pre, SliceBlockValues{5, -3})
	if err != nil {
		t.Fatal(err)
	}
	info, err := solver.Solve()
	if err != nil {
		t.Fatal(err)
	}
	// Mining block 0 requires block 1: 5 - 3 = 2 > 0, so both are mined.
	if info.NumContainedNodes != 2 || info.ContainedValue != 2 {
		t.Fatalf("got %d nodes / value %d, want 2 nodes / value 2",
			info.NumContainedNodes, info.ContainedValue)
	}

	// After updating the values, block 0 is no longer worth mining.
	if err := solver.UpdateValues(SliceBlockValues{1, -3}); err != nil {
		t.Fatal(err)
	}
	info, err = solver.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if info.NumContainedNodes != 0 || info.ContainedValue != 0 {
		t.Fatalf("got %d nodes / value %d, want 0 nodes / value 0",
			info.NumContainedNodes, info.ContainedValue)
	}
}
