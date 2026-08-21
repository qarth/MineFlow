package mineflow

import (
	"fmt"
	"math"
)

// pattern.go — port of PrecedencePattern, the pattern factories (OneFive,
// OneNine, KnightsMove, Naive, LessNaive, MinSearch), NaiveSearch,
// PrintPattern, and the pattern accuracy measurement from
// mineflow.cpp:1297-1741.

// PrecedencePattern is a set of offsets from a base block.
type PrecedencePattern struct {
	Offsets []Vector3I
}

// NewPrecedencePattern creates a pattern from the given offsets (copied).
func NewPrecedencePattern(offsets []Vector3I) PrecedencePattern {
	out := PrecedencePattern{Offsets: make([]Vector3I, len(offsets))}
	copy(out.Offsets, offsets)
	return out
}

// Size returns the number of offsets in the pattern.
func (p PrecedencePattern) Size() int {
	return len(p.Offsets)
}

// PatternOneFive returns the 1:5 pattern (5 offsets, one bench up).
func PatternOneFive() PrecedencePattern {
	return PrecedencePattern{Offsets: []Vector3I{
		{0, -1, 1}, {-1, 0, 1}, {0, 0, 1}, {1, 0, 1}, {0, 1, 1},
	}}
}

// PatternOneNine returns the 1:9 pattern (9 offsets, one bench up).
func PatternOneNine() PrecedencePattern {
	offsets := make([]Vector3I, 0, 9)
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			offsets = append(offsets, Vector3I{X: i, Y: j, Z: 1})
		}
	}
	return PrecedencePattern{Offsets: offsets}
}

// PatternKnightsMove returns the knight's move pattern.
func PatternKnightsMove() PrecedencePattern {
	return PrecedencePattern{Offsets: []Vector3I{
		{0, -1, 1}, {-1, 0, 1}, {0, 0, 1}, {1, 0, 1}, {0, 1, 1},
		{-1, -2, 2}, {1, -2, 2}, {-2, -1, 2}, {2, -1, 2},
		{-2, 1, 2}, {2, 1, 2}, {-1, 2, 2}, {1, 2, 2},
	}}
}

// PatternNaive returns the naive pattern for the given slope over numZ benches.
func PatternNaive(blockDef BlockDefinition, slopeDef SlopeDefinition, numZ int) PrecedencePattern {
	var ptrn PrecedencePattern
	NaiveSearch(blockDef, slopeDef, numZ, func(off Vector3I) {
		ptrn.Offsets = append(ptrn.Offsets, off)
	})
	return ptrn
}

// PatternLessNaive returns the "less naive" pattern for the given slope: it
// keeps track of x/y and doesn't add offsets at an already-seen x/y location
// (instead relying on transitivity through the original x/y).
func PatternLessNaive(blockDef BlockDefinition, slopeDef SlopeDefinition, numZ int) PrecedencePattern {
	var ptrn PrecedencePattern
	seen := make(map[int]map[int]bool)

	NaiveSearch(blockDef, slopeDef, numZ, func(off Vector3I) {
		ys, ok := seen[off.X]
		add := false
		if !ok {
			add = true
		} else if !ys[off.Y] {
			add = true
		}
		if add {
			ptrn.Offsets = append(ptrn.Offsets, off)
			if seen[off.X] == nil {
				seen[off.X] = make(map[int]bool)
			}
			seen[off.X][off.Y] = true
		}
	})

	return ptrn
}

// PatternMinSearch returns the Caccetta-Giannini minimum search pattern for
// the given block/slope definition over numZ benches: the "optimal" pattern
// for a specific definition.
func PatternMinSearch(blockDef BlockDefinition, slopeDef SlopeDefinition, n int) PrecedencePattern {
	var ptrn PrecedencePattern
	if slopeDef.Empty() {
		return ptrn
	}

	minSlope := slopeDef.MinSlope()
	maxHeight := blockDef.SizeZ * float64(n)
	maxThrow := maxHeight / math.Tan(minSlope)

	cx := int(math.Ceil(maxThrow))
	nx := cx*2 + 1
	nz := n + 1

	total := nx * nx * nz

	flagDef := blockDef
	flagDef.NumX = nx
	flagDef.NumY = nx
	flagDef.NumZ = nz

	const (
		notFlagged = 0
		noArcs     = 1
		someArcs   = 2
	)
	flag := make([]int8, total)

	// Now construct the minimum search pattern
	originIndex := flagDef.GridIndex(cx, cx, 0)
	flag[originIndex] = noArcs
	flagged := []int{originIndex}
	for z := 1; z < nz; z++ {
		thisHeight := float64(z) * flagDef.SizeZ
		thisThrow := thisHeight / math.Tan(minSlope)
		thisMaxOff := int(math.Ceil(thisThrow))

		// flag the violating blocks
		var newOffsets []Vector3I
		for x := -thisMaxOff; x <= thisMaxOff; x++ {
			for y := -thisMaxOff; y <= thisMaxOff; y++ {
				fi := flagDef.OffsetIndex(originIndex, x, y, z)

				if flag[fi] == notFlagged && slopeDef.Within(float64(x)*flagDef.SizeX,
					float64(y)*flagDef.SizeY,
					float64(z)*flagDef.SizeZ) {
					flag[fi] = noArcs

					flagged = append(flagged, fi)
					ptrn.Offsets = append(ptrn.Offsets, Vector3I{x, y, z})
					newOffsets = append(newOffsets, Vector3I{x, y, z})
				}
			}
		}

		// Extend flagged blocks
		var extra []int
		for _, fi := range flagged {
			fz := flagDef.ZIndex(fi)

			var offsets []Vector3I
			if flag[fi] == noArcs {
				offsets = ptrn.Offsets
				flag[fi] = someArcs
			} else {
				offsets = newOffsets
			}

			for _, arc := range offsets {
				if fz+arc.Z >= flagDef.NumZ {
					break
				}

				idx := flagDef.OffsetIndex(fi, arc.X, arc.Y, arc.Z)
				if flag[idx] == notFlagged {
					flag[idx] = noArcs
					extra = append(extra, idx)
				}
			}
		}

		flagged = append(flagged, extra...)
	}

	return ptrn
}

// PatternMinSearchSlope returns the minimum search pattern for a constant
// slope (radians) on a unit block model over numZ benches.
func PatternMinSearchSlope(slopeRad float64, numZ int) PrecedencePattern {
	blockDef := UnitModel(1, 1, 1)
	slopeDef := ConstantSlope(slopeRad)
	return PatternMinSearch(blockDef, slopeDef, numZ)
}

// NaiveSearch enumerates all offsets inside the slope cone up to numZ benches,
// invoking cb for each.
func NaiveSearch(blockDef BlockDefinition, slopeDef SlopeDefinition, nz int, cb func(Vector3I)) {
	if slopeDef.Empty() {
		return
	}

	minSlope := slopeDef.MinSlope()
	if minSlope <= 0 {
		return
	}

	for z := 1; z <= nz; z++ {
		thisHeight := blockDef.SizeZ * float64(z)
		thisThrow := thisHeight / math.Tan(minSlope)
		thisMaxOff := int(math.Ceil(thisThrow))

		for x := -thisMaxOff; x <= thisMaxOff; x++ {
			for y := -thisMaxOff; y <= thisMaxOff; y++ {
				if slopeDef.Within(float64(x)*blockDef.SizeX,
					float64(y)*blockDef.SizeY,
					float64(z)*blockDef.SizeZ) {
					cb(Vector3I{x, y, z})
				}
			}
		}
	}
}

// PatternAccuracy holds confusion-matrix statistics comparing a pattern
// against the naive slope cone.
type PatternAccuracy struct {
	TruePositive  int
	TrueNegative  int
	FalsePositive int
	FalseNegative int

	Accuracy            float64
	TruePositiveRate    float64
	FalseNegativeRate   float64
	MatthewsCorrelation float64
}

const (
	flagTrueNegative  = 0
	flagFalsePositive = 1
	flagTruePositive  = 2
	flagFalseNegative = 3
)

// getAccuracyFlag flags each block of the definition as a true/false
// positive/negative for the pattern vs the naive slope cone
// (mineflow.cpp:1549-1595).
func getAccuracyFlag(blockDef BlockDefinition, slopeDef SlopeDefinition, ptrn PrecedencePattern) []int8 {
	flag := make([]int8, blockDef.NumBlocks()) // TRUE_NEGATIVE

	mx := blockDef.NumX / 2
	my := blockDef.NumY / 2

	start := blockDef.GridIndex(mx, my, 0)
	NaiveSearch(blockDef, slopeDef, blockDef.NumZ, func(off Vector3I) {
		if blockDef.OffsetInDef(mx, my, 0, off.X, off.Y, off.Z) {
			idx := blockDef.OffsetIndex(start, off.X, off.Y, off.Z)
			flag[idx] = flagFalsePositive
		}
	})

	// Now apply the arc template, the flag will keep track of duplicates, and
	// avoid being inefficient
	stack := []int{start}
	for len(stack) > 0 {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for _, offset := range ptrn.Offsets {
			if blockDef.IndexOffsetInDef(t, offset.X, offset.Y, offset.Z) {
				idx := blockDef.OffsetIndex(t, offset.X, offset.Y, offset.Z)

				if flag[idx] > 1 {
					continue
				}

				if flag[idx] == flagFalsePositive {
					flag[idx] = flagTruePositive
				} else if flag[idx] == flagTrueNegative {
					flag[idx] = flagFalseNegative
				}

				stack = append(stack, idx)
			}
		}
	}

	return flag
}

func (a *PatternAccuracy) calcAccuracyMeasure() {
	tp := float64(a.TruePositive)
	fp := float64(a.FalsePositive)
	tn := float64(a.TrueNegative)
	fn := float64(a.FalseNegative)

	a.Accuracy = (tp + tn) / (tp + fp + tn + fn)
	a.TruePositiveRate = tp / (tp + fp)
	a.FalseNegativeRate = fn / (tp + fp)

	numer := tp*tn - fp*fn
	denom := math.Sqrt((tp + fp) * (tp + fn) * (tn + fp) * (tn + fn))
	if denom == 0 {
		denom = 1.0
	}
	a.MatthewsCorrelation = numer / denom
}

// MeasureAccuracy measures the accuracy of a pattern against the slope cone.
func MeasureAccuracy(blockDef BlockDefinition, slopeDef SlopeDefinition, ptrn PrecedencePattern) PatternAccuracy {
	var accuracy PatternAccuracy
	flag := getAccuracyFlag(blockDef, slopeDef, ptrn)

	for _, v := range flag {
		switch v {
		case flagTrueNegative:
			accuracy.TrueNegative++
		case flagTruePositive:
			accuracy.TruePositive++
		case flagFalseNegative:
			accuracy.FalseNegative++
		case flagFalsePositive:
			accuracy.FalsePositive++
		}
	}

	accuracy.calcAccuracyMeasure()
	return accuracy
}

// MultiMeasureAccuracy measures the accuracy of a pattern against the slope
// cone, one PatternAccuracy per bench level.
func MultiMeasureAccuracy(blockDef BlockDefinition, slopeDef SlopeDefinition, ptrn PrecedencePattern) []PatternAccuracy {
	flag := getAccuracyFlag(blockDef, slopeDef, ptrn)

	nz := blockDef.NumZ
	nxy := blockDef.NumX * blockDef.NumY

	accuracies := make([]PatternAccuracy, nz)

	accuracies[0].TruePositive = 1
	accuracies[0].TrueNegative = nxy - 1

	k := nxy
	for z := 1; z < blockDef.NumZ; z++ {
		accuracies[z] = accuracies[z-1]
		for yx := 0; yx < blockDef.NumX*blockDef.NumY; yx++ {
			v := flag[k]
			k++

			switch v {
			case flagTrueNegative:
				accuracies[z].TrueNegative++
			case flagTruePositive:
				accuracies[z].TruePositive++
			case flagFalseNegative:
				accuracies[z].FalseNegative++
			case flagFalsePositive:
				accuracies[z].FalsePositive++
			}
		}
	}

	for i := range accuracies {
		accuracies[i].calcAccuracyMeasure()
	}
	return accuracies
}

func (a PatternAccuracy) String() string {
	return fmt.Sprintf("tp %d\ntn %d\nfp %d\nfn %d\nac %v\ntr %v\nfr %v\nmc %v",
		a.TruePositive, a.TrueNegative, a.FalsePositive, a.FalseNegative,
		a.Accuracy, a.TruePositiveRate, a.FalseNegativeRate, a.MatthewsCorrelation)
}

// PrintPattern prints an ASCII picture of the pattern to stdout.
func PrintPattern(p PrecedencePattern) {
	if len(p.Offsets) == 0 {
		return
	}

	xlo, ylo := math.MaxInt, math.MaxInt
	xhi, yhi := math.MinInt, math.MinInt

	for _, off := range p.Offsets {
		if off.X < xlo {
			xlo = off.X
		}
		if off.Y < ylo {
			ylo = off.Y
		}
		if off.X > xhi {
			xhi = off.X
		}
		if off.Y > yhi {
			yhi = off.Y
		}
	}
	pnx := xhi - xlo + 1
	pny := yhi - ylo + 1
	img := make([]int, pnx*pny)
	for i := range img {
		img[i] = -1
	}

	for _, off := range p.Offsets {
		i := (off.Y-ylo)*pnx + (off.X - xlo)
		if img[i] == -1 {
			img[i] = off.Z
		}
		if off.X == 0 && off.Y == 0 {
			img[i] = 0
		}
	}

	i := 0
	for y := 0; y < pny; y++ {
		for x := 0; x < pnx; x++ {
			if img[i] == -1 {
				fmt.Print("  ")
			} else {
				fmt.Printf("%2d", img[i])
			}
			i++
		}
		fmt.Println()
	}
}
