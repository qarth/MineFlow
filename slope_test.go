package mineflow

import "testing"

// slope_test.go — port of the C++ slope tests (mineflow.cpp:3774-3891).

func deg(d float64) float64 { return ToRadians(d) }

func TestSlopePairLessThan(t *testing.T) {
	a := AzmSlopePair{deg(90), deg(45)}
	b := AzmSlopePair{deg(0), deg(50)}
	c := AzmSlopePair{deg(180), deg(40)}

	if !a.less(c) || !b.less(c) || !b.less(a) {
		t.Fatal("unexpected azimuth ordering")
	}
}

func TestSlopeGetSimple(t *testing.T) {
	def := NewSlopeDefinition([]AzmSlopePair{
		{deg(0), deg(45)},
		{deg(90), deg(50)},
		{deg(180), deg(40)},
	})

	const tol = 0.0000001
	assertNear(t, deg(45.0), def.Get(deg(0)), tol)
	assertNear(t, deg(47.5), def.Get(deg(45)), tol)
	assertNear(t, deg(50.0), def.Get(deg(90)), tol)
	assertNear(t, deg(45.0), def.Get(deg(135)), tol)
	assertNear(t, deg(40.0), def.Get(deg(180)), tol)
	assertNear(t, deg(42.5), def.Get(deg(270)), tol)
	assertNear(t, deg(45.0), def.Get(deg(360)), tol)
}

func TestSlopeGetSingle(t *testing.T) {
	def := NewSlopeDefinition([]AzmSlopePair{{deg(20), deg(45)}})

	const tol = 0.0000001
	for _, azm := range []float64{0, 45, 270, 730, -30, 999} {
		assertNear(t, deg(45), def.Get(deg(azm)), tol)
	}
}

func TestSlopeGetRound(t *testing.T) {
	def := NewSlopeDefinition([]AzmSlopePair{
		{deg(0), deg(40)},
		{deg(180), deg(50)},
	})

	const tol = 0.0000001
	assertNear(t, deg(40.0), def.Get(deg(0)), tol)
	assertNear(t, deg(42.5), def.Get(deg(45)), tol)
	assertNear(t, deg(45.0), def.Get(deg(90)), tol)
	assertNear(t, deg(47.5), def.Get(deg(135)), tol)
	assertNear(t, deg(50.0), def.Get(deg(180)), tol)
	assertNear(t, deg(45.0), def.Get(deg(270)), tol)
}

func TestSlopeCubic(t *testing.T) {
	def := NewSlopeDefinition([]AzmSlopePair{
		{deg(0), deg(45)},
		{deg(45), deg(45)},
		{deg(90), deg(30)},
		{deg(135), deg(40)},
		{deg(180), deg(45)},
		{deg(270), deg(45)},
	})

	def2 := CubicInterpolate(def, 512)
	if def2.NumPairs() != 512 {
		t.Fatalf("unexpected pair count: got %d want 512", def2.NumPairs())
	}

	assertNear(t, deg(45.0000), def2.Get(deg(0)), 0.000001)
	assertNear(t, deg(43.1476), def2.Get(deg(150)), 0.000001)
}

func TestSlopeCosine(t *testing.T) {
	def := NewSlopeDefinition([]AzmSlopePair{
		{deg(0), deg(45)},
		{deg(45), deg(45)},
		{deg(90), deg(30)},
		{deg(135), deg(40)},
		{deg(180), deg(45)},
		{deg(270), deg(45)},
	})

	def2 := CosineInterpolate(def, 512)
	if def2.NumPairs() != 512 {
		t.Fatalf("unexpected pair count: got %d want 512", def2.NumPairs())
	}

	assertNear(t, deg(45.0000), def2.Get(deg(0)), 0.000001)
	assertNear(t, deg(41.2503), def2.Get(deg(150)), 0.000001)
}

func TestSlopeViolateBase(t *testing.T) {
	def := ConstantSlope(deg(45))

	type tc struct {
		dx, dy, dz float64
		want       bool
	}
	for _, c := range []tc{
		{1, 0, 1, true},
		{-1, 0, 1, true},
		{0, 1, 1, true},
		{0, -1, 1, true},
		{1, 1, 1, false},
		{2, 0, 2, true},
		{4, 0, 4, true},
		{2, 2, 4, true},
	} {
		if got := def.Within(c.dx, c.dy, c.dz); got != c.want {
			t.Fatalf("Within(%v, %v, %v): got %v want %v", c.dx, c.dy, c.dz, got, c.want)
		}
	}
}

func TestSlopeViolateDual(t *testing.T) {
	def := NewSlopeDefinition([]AzmSlopePair{
		{deg(0), deg(40)},
		{deg(180), deg(60)},
	})

	if !def.Within(0, 1, 1) {
		t.Fatal("Within(0, 1, 1): got false want true")
	}
	if def.Within(0, 2, 1) {
		t.Fatal("Within(0, 2, 1): got true want false")
	}
	if def.Within(0, -1, 1) {
		t.Fatal("Within(0, -1, 1): got true want false")
	}
}
