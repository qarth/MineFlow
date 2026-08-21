package mineflow

import (
	"math"
	"sort"
)

// slope.go — port of AzmSlopePair and SlopeDefinition from
// mineflow.cpp:415-672. All angles are in radians.

// AzmSlopePair is a single component of a full slope definition.
// Both Azimuth and Slope are in radians.
type AzmSlopePair struct {
	Azimuth float64
	Slope   float64
}

// less orders by azimuth, then slope (AzmSlopePair::operator< in C++).
func (a AzmSlopePair) less(other AzmSlopePair) bool {
	if a.Azimuth == other.Azimuth {
		return a.Slope < other.Slope
	}
	return a.Azimuth < other.Azimuth
}

// SlopeDefinition is a sorted list of azimuth slope pairs. It linearly
// interpolates for any requested azimuth; other interpolation techniques are
// supported by creating a very "full" slope definition (say 512 pairs) and
// then linearly interpolating that.
type SlopeDefinition struct {
	pairs []AzmSlopePair // sorted by azimuth
}

// NewSlopeDefinition creates a SlopeDefinition from the given pairs
// (they are sorted internally, and the slice is copied).
func NewSlopeDefinition(pairs []AzmSlopePair) SlopeDefinition {
	out := make([]AzmSlopePair, len(pairs))
	copy(out, pairs)
	sort.SliceStable(out, func(i, j int) bool { return out[i].less(out[j]) })
	return SlopeDefinition{pairs: out}
}

// ConstantSlope returns a SlopeDefinition with a constant slope (radians).
func ConstantSlope(slope float64) SlopeDefinition {
	return SlopeDefinition{pairs: []AzmSlopePair{{Azimuth: 0, Slope: slope}}}
}

// getLeftRight finds the pair indices bracketing the given azimuth,
// wrapping around TAU (GetLeftRight in the C++ code, mineflow.cpp:452-463).
// pairs must be non-empty.
func getLeftRight(pairs []AzmSlopePair, azimuth float64) (left, right int) {
	for azimuth >= TAU {
		azimuth -= TAU
	}
	for azimuth < 0 {
		azimuth += TAU
	}

	// lower_bound: first pair with Azimuth >= azimuth
	right = sort.Search(len(pairs), func(i int) bool { return pairs[i].Azimuth >= azimuth })
	if right == len(pairs) {
		right = 0
	}
	if right == 0 {
		left = len(pairs) - 1
	} else {
		left = right - 1
	}
	return left, right
}

// getXval returns the interpolation parameter between the left and right
// pairs at the given azimuth (GetXval in the C++ code, mineflow.cpp:464-479).
func getXval(pairs []AzmSlopePair, left, right int, azimuth float64) float64 {
	var toLeft, toRight float64
	if pairs[left].Azimuth > azimuth {
		toLeft = TAU - pairs[left].Azimuth + azimuth
	} else {
		toLeft = azimuth - pairs[left].Azimuth
	}
	if pairs[right].Azimuth < azimuth {
		toRight = TAU - azimuth + pairs[right].Azimuth
	} else {
		toRight = pairs[right].Azimuth - azimuth
	}
	return toLeft / (toLeft + toRight)
}

// Get computes the slope (radians) at the given azimuth (radians).
func (d SlopeDefinition) Get(azimuth float64) float64 {
	if len(d.pairs) == 0 {
		return 0.0
	}
	if len(d.pairs) == 1 {
		return d.pairs[0].Slope
	}

	left, right := getLeftRight(d.pairs, azimuth)
	xval := getXval(d.pairs, left, right, azimuth)

	return d.pairs[left].Slope + (d.pairs[right].Slope-d.pairs[left].Slope)*xval
}

// Within computes whether the given vector (dx, dy, dz) is within the slope
// definition.
func (d SlopeDefinition) Within(dx, dy, dz float64) bool {
	if dx == 0 && dy == 0 {
		return true
	}
	dt := math.Sqrt(dx*dx + dy*dy)
	theta := math.Atan(math.Abs(dz) / dt)
	azm := math.Pi/2 - math.Atan2(dy, dx)
	return theta >= d.Get(azm)
}

// WithinVector computes whether the given vector is within the slope
// definition.
func (d SlopeDefinition) WithinVector(v Vector3D) bool {
	return d.Within(v.X, v.Y, v.Z)
}

// MinSlope returns the minimum slope over all azimuths (radians).
func (d SlopeDefinition) MinSlope() float64 {
	if len(d.pairs) == 0 {
		return 0.0
	}
	minSlope := d.pairs[0].Slope
	for _, pair := range d.pairs {
		if pair.Slope < minSlope {
			minSlope = pair.Slope
		}
	}
	return minSlope
}

// NumPairs returns the number of azimuth/slope pairs.
func (d SlopeDefinition) NumPairs() int {
	return len(d.pairs)
}

// Pairs returns the sorted azimuth/slope pairs.
func (d SlopeDefinition) Pairs() []AzmSlopePair {
	return d.pairs
}

// Empty reports whether the definition has no pairs.
func (d SlopeDefinition) Empty() bool {
	return len(d.pairs) == 0
}

// CubicInterpolate returns a densified SlopeDefinition using cubic
// interpolation (cnt points; the C++ default is 512). Panics if the
// definition has fewer than 4 pairs.
func CubicInterpolate(def SlopeDefinition, cnt int) SlopeDefinition {
	n := def.NumPairs()
	if n < 4 {
		panic("must be at least 4 pairs for cubic")
	}
	pairs := def.pairs

	y0 := n - 1
	y1 := 0
	y2 := 1
	y3 := 2

	outPairs := make([]AzmSlopePair, cnt+1)
	for i, v := range Linspace(0, TAU, cnt+1) {
		if v >= pairs[y2].Azimuth && pairs[y2].Azimuth != 0 {
			y0 = y1
			y1 = y2
			y2 = y3
			y3++
			if y3 == n {
				y3 = 0
			}
		}

		mu := getXval(pairs, y1, y2, v)

		yp0 := pairs[y0].Slope
		yp1 := pairs[y1].Slope
		yp2 := pairs[y2].Slope
		yp3 := pairs[y3].Slope

		mu2 := mu * mu
		a0 := yp3 - yp2 - yp0 + yp1
		a1 := yp0 - yp1 - a0
		a2 := yp2 - yp0
		a3 := yp1

		outPairs[i].Azimuth = v
		outPairs[i].Slope = a0*mu*mu2 + a1*mu2 + a2*mu + a3
	}
	outPairs = outPairs[:cnt]

	return NewSlopeDefinition(outPairs)
}

// CosineInterpolate returns a densified SlopeDefinition using cosine
// interpolation (cnt points; the C++ default is 512). Panics if the
// definition has fewer than 2 pairs.
func CosineInterpolate(def SlopeDefinition, cnt int) SlopeDefinition {
	n := def.NumPairs()
	if n < 2 {
		panic("must be at least 2 pairs for cosine")
	}
	pairs := def.pairs

	y1 := 0
	y2 := 1

	outPairs := make([]AzmSlopePair, cnt+1)
	for i, v := range Linspace(0, TAU, cnt+1) {
		if v >= pairs[y2].Azimuth && pairs[y2].Azimuth != 0 {
			y1 = y2
			y2++
			if y2 == n {
				y2 = 0
			}
		}

		mu := getXval(pairs, y1, y2, v)
		yp1 := pairs[y1].Slope
		yp2 := pairs[y2].Slope

		mu2 := (1 - math.Cos(mu*math.Pi)) / 2.0

		outPairs[i].Azimuth = v
		outPairs[i].Slope = yp1*(1-mu2) + yp2*mu2
	}
	outPairs = outPairs[:cnt]

	return NewSlopeDefinition(outPairs)
}
