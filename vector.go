package mineflow

import "math"

// This file ports the vector helpers, angle conversion, and linspace utilities
// from mineflow.h (VectorBase templates, ToDegrees/ToRadians, Linspace).

// TAU is 2*pi, matching the constant used throughout the C++ code.
const TAU = 6.283185307179586476925286766559

// ToDegrees converts radians to degrees.
func ToDegrees(radians float64) float64 {
	return radians * 360.0 / TAU
}

// ToRadians converts degrees to radians.
func ToRadians(degrees float64) float64 {
	return degrees * TAU / 360.0
}

// Vector3I is a 3D integer vector, used for precedence pattern offsets
// (Vector3IT in the C++ code).
type Vector3I struct {
	X int
	Y int
	Z int
}

// Vector2D is a 2D double vector (VectorBase<double, 2> in C++).
type Vector2D struct {
	X float64
	Y float64
}

// Vector3D is a 3D double vector (VectorBase<double, 3> in C++).
type Vector3D struct {
	X float64
	Y float64
	Z float64
}

func (v Vector3D) Add(o Vector3D) Vector3D  { return Vector3D{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vector3D) Sub(o Vector3D) Vector3D  { return Vector3D{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vector3D) Scale(s float64) Vector3D { return Vector3D{v.X * s, v.Y * s, v.Z * s} }
func (v Vector3D) Neg() Vector3D            { return Vector3D{-v.X, -v.Y, -v.Z} }

// Dot returns the dot product of two 3D vectors.
func Dot(a, b Vector3D) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// MagnitudeSquared returns the squared length of the vector.
func MagnitudeSquared(v Vector3D) float64 { return Dot(v, v) }

// Magnitude returns the length of the vector.
func Magnitude(v Vector3D) float64 { return math.Sqrt(MagnitudeSquared(v)) }

// Distance returns the distance between two points.
func Distance(a, b Vector3D) float64 { return Magnitude(b.Sub(a)) }

// Theta returns the angle between two vectors in radians.
func Theta(a, b Vector3D) float64 {
	return math.Acos(Dot(a, b) / (Magnitude(a) * Magnitude(b)))
}

// Normalized returns the unit vector in the same direction.
func Normalized(v Vector3D) Vector3D { return v.Scale(1.0 / Magnitude(v)) }

// Cross returns the cross product of two 3D vectors.
func Cross(a, b Vector3D) Vector3D {
	return Vector3D{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

// TriArea2 returns twice the signed area of the triangle formed by a, b, c.
func TriArea2(a, b, c Vector2D) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (c.X-a.X)*(b.Y-a.Y)
}

// TriArea returns the signed area of the triangle formed by a, b, c.
func TriArea(a, b, c Vector2D) float64 { return TriArea2(a, b, c) / 2.0 }

// IsLeft reports whether c is to the left of the directed line a -> b.
func IsLeft(a, b, c Vector2D) bool { return TriArea2(a, b, c) > 0.0 }

// IsRight reports whether c is to the right of the directed line a -> b.
func IsRight(a, b, c Vector2D) bool { return TriArea2(a, b, c) < 0.0 }

// IsCollinear reports whether a, b, c are collinear.
func IsCollinear(a, b, c Vector2D) bool { return TriArea2(a, b, c) == 0.0 }

// Linspace returns n evenly spaced values from start to stop, inclusive
// (InplaceLinspace in the C++ code).
func Linspace(start, stop float64, n int) []float64 {
	if n <= 0 {
		return nil
	}
	out := make([]float64, n)
	step := (stop - start) / float64(n-1)
	for i := range out {
		out[i] = start + float64(i)*step
	}
	return out
}
