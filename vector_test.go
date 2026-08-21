package mineflow

import (
	"math"
	"testing"
)

// vector_test.go — port of the C++ TEST(Vector, *), TEST(Angles, *), and
// TEST(Linspace, *) suites (mineflow.cpp:3220-3574).

func assertNear(t *testing.T, want, got, tol float64) {
	t.Helper()
	if math.Abs(want-got) > tol {
		t.Fatalf("got %v, want %v (tol %v)", got, want, tol)
	}
}

func TestVectorBasicConstructor(t *testing.T) {
	a := Vector3D{1.2, -13.4, 5.41}
	if a.X != 1.2 || a.Y != -13.4 || a.Z != 5.41 {
		t.Fatalf("unexpected vector: %+v", a)
	}
}

func TestVectorAddition(t *testing.T) {
	a := Vector3D{1, 2, 3}
	b := Vector3D{-1, 7.2, 0}

	c := a.Add(b)
	if c.X != 0 || c.Y != 9.2 || c.Z != 3 {
		t.Fatalf("unexpected sum: %+v", c)
	}

	a = a.Add(a)
	if a.X != 2 || a.Y != 4 || a.Z != 6 {
		t.Fatalf("unexpected doubling: %+v", a)
	}
}

func TestVectorSubtraction(t *testing.T) {
	a := Vector3D{12, 4, 2}
	b := Vector3D{3, 8, 1}

	c := a.Sub(b)
	if c.X != 9 || c.Y != -4 || c.Z != 1 {
		t.Fatalf("unexpected difference: %+v", c)
	}

	a = a.Sub(a)
	if a.X != 0 || a.Y != 0 || a.Z != 0 {
		t.Fatalf("unexpected self-difference: %+v", a)
	}
}

func TestVectorMultiplication(t *testing.T) {
	a := Vector3D{1.2, 2.4, 3.6}

	b := a.Scale(4.0)
	if b != (Vector3D{4.8, 9.6, 14.4}) {
		t.Fatalf("unexpected product: %+v", b)
	}

	b = b.Scale(1.0 / 4.0)
	if b != a {
		t.Fatalf("unexpected round trip: %+v", b)
	}
}

func TestVectorDotProduct(t *testing.T) {
	if got := Dot(Vector3D{1, 3, -5}, Vector3D{4, -2, -1}); got != 3 {
		t.Fatalf("unexpected dot product: %v", got)
	}
	if got := Dot(Vector3D{1, 0, 0}, Vector3D{0, 1, 0}); got != 0 {
		t.Fatalf("unexpected dot product: %v", got)
	}
}

func TestVectorMagnitude(t *testing.T) {
	if got := Magnitude(Vector3D{1, 1, 0}); got != math.Sqrt(2) {
		t.Fatalf("unexpected magnitude: %v", got)
	}
	b := Vector3D{3, 4, 0}
	if got := Magnitude(b); got != 5 {
		t.Fatalf("unexpected magnitude: %v", got)
	}
	if got := MagnitudeSquared(b); got != 25 {
		t.Fatalf("unexpected magnitude squared: %v", got)
	}
	c := Vector3D{12, 16, 25}
	assertNear(t, 32.0156, Magnitude(c), 0.001)
	if got := MagnitudeSquared(c); got != 1025 {
		t.Fatalf("unexpected magnitude squared: %v", got)
	}
}

func TestVectorTheta(t *testing.T) {
	a := Vector3D{1, 0, 0}
	b := Vector3D{0, 1, 0}

	assertNear(t, 1.5708, Theta(a, b), 0.0001)

	c := Vector3D{1, 1, 0}
	assertNear(t, 0.7854, Theta(a, c), 0.0001)
	assertNear(t, 0.7854, Theta(c, b), 0.0001)
}

func TestVectorCross(t *testing.T) {
	if got := Cross(Vector3D{1, 0, 0}, Vector3D{0, 1, 0}); got != (Vector3D{0, 0, 1}) {
		t.Fatalf("unexpected cross product: %+v", got)
	}
	if got := Cross(Vector3D{3, -3, 1}, Vector3D{4, 9, 2}); got != (Vector3D{-15, -2, 39}) {
		t.Fatalf("unexpected cross product: %+v", got)
	}
}

func TestVectorNormalize(t *testing.T) {
	if got := Normalized(Vector3D{1.5, 0, 0}); got != (Vector3D{1, 0, 0}) {
		t.Fatalf("unexpected normalization: %+v", got)
	}
	want := math.Sqrt(2.0) / 2
	got := Normalized(Vector3D{22.6, 22.6, 0})
	assertNear(t, want, got.X, 0.0000001)
	assertNear(t, want, got.Y, 0.0000001)
	assertNear(t, 0, got.Z, 0.0000001)
}

func TestVectorLeftRight(t *testing.T) {
	a := Vector2D{0, 0}
	b := Vector2D{1, 1}

	if !IsLeft(a, b, Vector2D{0, 1}) || IsRight(a, b, Vector2D{0, 1}) || IsCollinear(a, b, Vector2D{0, 1}) {
		t.Fatal("(0,1) should be strictly left of (0,0)->(1,1)")
	}
	if IsLeft(a, b, Vector2D{2, 2}) || IsRight(a, b, Vector2D{2, 2}) || !IsCollinear(a, b, Vector2D{2, 2}) {
		t.Fatal("(2,2) should be collinear with (0,0)->(1,1)")
	}
	if IsLeft(a, b, Vector2D{20, 1}) || !IsRight(a, b, Vector2D{20, 1}) || IsCollinear(a, b, Vector2D{20, 1}) {
		t.Fatal("(20,1) should be strictly right of (0,0)->(1,1)")
	}
}

func TestVectorTriArea(t *testing.T) {
	a := Vector2D{-1, 0}
	b := Vector2D{0, 1}
	c := Vector2D{1, 0}

	if got := TriArea(a, b, c); got != -1 {
		t.Fatalf("unexpected triangle area: %v", got)
	}
	if got := TriArea(c, b, a); got != 1 {
		t.Fatalf("unexpected triangle area: %v", got)
	}
}

func TestAnglesToDegrees(t *testing.T) {
	assertNear(t, 180.0, ToDegrees(3.14159265), 0.00001)
}

func TestAnglesToRadians(t *testing.T) {
	assertNear(t, 3.14159265, ToRadians(180.0), 0.00001)
}

func TestLinspaceBase(t *testing.T) {
	a := Linspace(0, 100, 11)
	if a[0] != 0 || a[1] != 10 || a[2] != 20 || a[3] != 30 || a[10] != 100 {
		t.Fatalf("unexpected linspace: %v", a)
	}
}

func TestLinspaceNonZeroStart(t *testing.T) {
	a := Linspace(20.0, 34.4, 10)
	if a[0] != 20.0 || a[1] != 21.6 || a[8] != 32.8 || a[9] != 34.4 {
		t.Fatalf("unexpected linspace: %v", a)
	}
}

func TestLinspaceNegativeRange(t *testing.T) {
	a := Linspace(83.1, -10.0, 20)
	assertNear(t, 83.1, a[0], 0.0001)
	assertNear(t, 78.2, a[1], 0.0001)
	assertNear(t, -5.1, a[18], 0.0001)
	assertNear(t, -10.0, a[19], 0.0001)
}

func TestLinspaceGeneratorGauss(t *testing.T) {
	sum := 0.0
	for _, v := range Linspace(0, 100, 101) {
		sum += v
	}
	assertNear(t, 5050, sum, 0.00001)
}

func TestLinspaceGeneratorNegative(t *testing.T) {
	sum := 0.0
	for _, v := range Linspace(50, -50, 10) {
		sum += v
	}
	if sum != 0 {
		t.Fatalf("unexpected sum: %v", sum)
	}
}
