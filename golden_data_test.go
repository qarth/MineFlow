package mineflow

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
)

// golden_data_test.go — port of the C++ TEST(MFlow, *) data-file suites
// (mineflow.cpp:4016-4175). The block counts and contained values are the
// golden answers asserted by the C++ test suite.

func readTestDataValues(t *testing.T, stem string) SliceBlockValues {
	t.Helper()

	f, err := os.Open("data/" + stem + ".dat")
	if err != nil {
		t.Fatalf("opening data file: %v", err)
	}
	defer f.Close()

	var values []int64
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		v, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			t.Fatalf("parsing %q in %s: %v", line, stem, err)
		}
		values = append(values, v)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading data file: %v", err)
	}
	return values
}

// runGolden solves the model and checks the golden answers.
func runGolden(t *testing.T, stem string, blockDef BlockDefinition, pre PrecedenceConstraints, wantNodes, wantValue int64) {
	t.Helper()

	values := readTestDataValues(t, stem)
	if int64(values.NumBlocks()) != int64(blockDef.NumBlocks()) {
		t.Fatalf("%s: value count %d does not match block count %d",
			stem, values.NumBlocks(), blockDef.NumBlocks())
	}
	if pre.NumBlocks() != blockDef.NumBlocks() {
		t.Fatalf("%s: precedence block count %d does not match %d",
			stem, pre.NumBlocks(), blockDef.NumBlocks())
	}

	solver, err := NewPseudoSolver(pre, values)
	if err != nil {
		t.Fatal(err)
	}
	info, err := solver.Solve()
	if err != nil {
		t.Fatal(err)
	}

	if int64(info.NumNodes) != int64(values.NumBlocks()) {
		t.Fatalf("%s: NumNodes: got %d want %d", stem, info.NumNodes, values.NumBlocks())
	}
	if int64(info.NumContainedNodes) != wantNodes {
		t.Fatalf("%s: NumContainedNodes: got %d want %d", stem, info.NumContainedNodes, wantNodes)
	}
	if info.ContainedValue != wantValue {
		t.Fatalf("%s: ContainedValue: got %d want %d", stem, info.ContainedValue, wantValue)
	}
	t.Logf("%s: %d/%d blocks, value %d, %.3fs",
		stem, info.NumContainedNodes, info.NumNodes, info.ContainedValue, info.ElapsedSeconds)
}

func TestMFlowSim2D76(t *testing.T) {
	blockDef := UnitModel(75, 1, 40)
	pre := NewRegular2DGrid45DegreePrecedence(blockDef.NumX, blockDef.NumZ)
	runGolden(t, "sim2d76", blockDef, pre, 945, 295932)
}

func TestMFlowSim2D76Largest(t *testing.T) {
	blockDef := UnitModel(75, 1, 40)
	values := readTestDataValues(t, "sim2d76")
	pre := NewRegular2DGrid45DegreePrecedence(blockDef.NumX, blockDef.NumZ)

	solver, err := NewPseudoSolver(pre, values)
	if err != nil {
		t.Fatal(err)
	}
	info, err := solver.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if info.NumContainedNodes != 945 || info.ContainedValue != 295932 {
		t.Fatalf("got %d nodes / value %d, want 945 / 295932",
			info.NumContainedNodes, info.ContainedValue)
	}

	if err := solver.UpdateValues(NewSolveLargestValuesAdapter(values)); err != nil {
		t.Fatal(err)
	}
	info2, err := solver.Solve()
	if err != nil {
		t.Fatal(err)
	}
	if info2.NumContainedNodes != 946 {
		t.Fatalf("largest: NumContainedNodes: got %d want 946", info2.NumContainedNodes)
	}
}

func TestMFlowBauxiteMed(t *testing.T) {
	blockDef := UnitModel(120, 120, 26)
	pattern := PatternMinSearchSlope(deg(45), 8)
	pre := NewRegular3DBlockModelPatternPrecedence(blockDef, pattern)
	runGolden(t, "bauxitemed", blockDef, pre, 74412, 28416592)
}

func TestMFlowBauxiteMedLargest(t *testing.T) {
	blockDef := UnitModel(120, 120, 26)
	values := readTestDataValues(t, "bauxitemed")
	pattern := PatternMinSearchSlope(deg(45), 8)
	pre := NewRegular3DBlockModelPatternPrecedence(blockDef, pattern)

	solver, err := NewPseudoSolver(pre, values)
	if err != nil {
		t.Fatal(err)
	}
	info, err := solver.SolveLargest()
	if err != nil {
		t.Fatal(err)
	}
	if info.NumContainedNodes != 76813 || info.ContainedValue != 28416592 {
		t.Fatalf("largest: got %d nodes / value %d, want 76813 / 28416592",
			info.NumContainedNodes, info.ContainedValue)
	}
}

func TestMFlowCuCase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large model in -short mode")
	}
	blockDef := UnitModel(170, 215, 50)
	pattern := PatternMinSearchSlope(deg(45), 8)
	pre := NewRegular3DBlockModelPatternPrecedence(blockDef, pattern)
	runGolden(t, "cucase", blockDef, pre, 357304, 19175685)
}

func TestMFlowCuPipe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large model in -short mode")
	}
	blockDef := UnitModel(180, 180, 85)
	pattern := PatternMinSearchSlope(deg(45), 8)
	pre := NewRegular3DBlockModelPatternPrecedence(blockDef, pattern)
	runGolden(t, "cupipe", blockDef, pre, 198078, 102306787)
}

func TestMFlowMcLaughlinGeo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large model in -short mode")
	}
	blockDef := UnitModel(140, 296, 68)
	pattern := PatternMinSearchSlope(deg(45), 8)
	pre := NewRegular3DBlockModelPatternPrecedence(blockDef, pattern)
	runGolden(t, "mclaughlingeo", blockDef, pre, 345936, 1145395060)
}
