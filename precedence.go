package mineflow

import (
	"iter"
	"sort"
)

// precedence.go — port of the concrete precedence-constraint classes and the
// reachability helpers from mineflow.cpp:830-1293 and 1745-2092.
//
// The C++ input-iterator sources (BlockOffsetSource, BlockOffsetExtentSource,
// BlockVectorSource, ReachableBlockSource, ...) are replaced by iter.Seq
// closures carrying the same state.

// Regular2DGrid45DegreePrecedence implements 45-degree precedence on a 2D
// (x, z) grid (mineflow.cpp:1227-1293).
type Regular2DGrid45DegreePrecedence struct {
	numX int
	numZ int

	antecedentOffsets [3]int
	successorOffsets  [3]int
}

func NewRegular2DGrid45DegreePrecedence(numX, numZ int) *Regular2DGrid45DegreePrecedence {
	if numX <= 0 || numZ <= 0 {
		panic("invalid grid size")
	}
	return &Regular2DGrid45DegreePrecedence{
		numX:              numX,
		numZ:              numZ,
		antecedentOffsets: [3]int{numX - 1, numX, numX + 1},
		successorOffsets:  [3]int{-numX - 1, -numX, -numX + 1},
	}
}

func (p *Regular2DGrid45DegreePrecedence) NumBlocks() int {
	return p.numX * p.numZ
}

// xAdjusted yields blockIndex + offsets[start:start+n], trimming offsets that
// would wrap around the x edges (XAdjustedSource in the C++ code).
func (p *Regular2DGrid45DegreePrecedence) xAdjusted(blockIndex int, offsets [3]int, yield func(int) bool) {
	start := 0
	n := 3
	x := blockIndex % p.numX

	if x == 0 {
		start++
		n--
	}
	if x == p.numX-1 {
		n--
	}

	for i := start; i < start+n; i++ {
		if !yield(blockIndex + offsets[i]) {
			return
		}
	}
}

func (p *Regular2DGrid45DegreePrecedence) Antecedents(fromBlockIndex int) iter.Seq[int] {
	return func(yield func(int) bool) {
		z := fromBlockIndex / p.numX
		if z >= p.numZ-1 {
			return
		}
		p.xAdjusted(fromBlockIndex, p.antecedentOffsets, yield)
	}
}

func (p *Regular2DGrid45DegreePrecedence) Successors(toBlockIndex int) iter.Seq[int] {
	return func(yield func(int) bool) {
		z := toBlockIndex / p.numX
		if z <= 0 {
			return
		}
		p.xAdjusted(toBlockIndex, p.successorOffsets, yield)
	}
}

// Regular3DBlockModelPatternPrecedence applies a PrecedencePattern to every
// block of a regular 3D block model (mineflow.cpp:1812-1936). The workhorse
// precedence class.
type Regular3DBlockModelPatternPrecedence struct {
	numX, numY, numZ int

	offsets              []Vector3I // sorted by (z, y, x)
	precomputed1DOffsets []int
	maxOffsetZ           int
	numOffsetsByZMinus   []int // prefix sums of offsets per z level

	// The inner region: blocks within it need no per-offset bounds checks.
	xLo, xHi int
	yLo, yHi int
}

func NewRegular3DBlockModelPatternPrecedence(blockDef BlockDefinition, pattern PrecedencePattern) *Regular3DBlockModelPatternPrecedence {
	return NewRegular3DBlockModelPatternPrecedenceFromDims(blockDef.NumX, blockDef.NumY, blockDef.NumZ, pattern)
}

func NewRegular3DBlockModelPatternPrecedenceFromDims(numX, numY, numZ int, pattern PrecedencePattern) *Regular3DBlockModelPatternPrecedence {
	if len(pattern.Offsets) == 0 {
		panic("invalid pattern")
	}

	p := &Regular3DBlockModelPatternPrecedence{
		numX: numX,
		numY: numY,
		numZ: numZ,
	}

	p.offsets = make([]Vector3I, len(pattern.Offsets))
	copy(p.offsets, pattern.Offsets)
	sort.Slice(p.offsets, func(i, j int) bool {
		a, b := p.offsets[i], p.offsets[j]
		if a.Z == b.Z {
			if a.Y == b.Y {
				return a.X < b.X
			}
			return a.Y < b.Y
		}
		return a.Z < b.Z
	})

	p.precomputed1DOffsets = make([]int, len(p.offsets))
	for i, off := range p.offsets {
		p.precomputed1DOffsets[i] = off.X + off.Y*numX + off.Z*numX*numY
	}

	p.xLo, p.xHi = 0, numX
	p.yLo, p.yHi = 0, numY
	for _, off := range p.offsets {
		if off.X < 0 && -off.X > p.xLo {
			p.xLo = -off.X
		}
		if off.X > 0 && numX-off.X < p.xHi {
			p.xHi = numX - off.X
		}
		if off.Y < 0 && -off.Y > p.yLo {
			p.yLo = -off.Y
		}
		if off.Y > 0 && numY-off.Y < p.yHi {
			p.yHi = numY - off.Y
		}
	}

	p.maxOffsetZ = p.offsets[len(p.offsets)-1].Z
	p.numOffsetsByZMinus = make([]int, p.maxOffsetZ+1)
	for _, off := range p.offsets {
		p.numOffsetsByZMinus[off.Z]++
	}
	for i := 1; i < len(p.numOffsetsByZMinus); i++ {
		p.numOffsetsByZMinus[i] += p.numOffsetsByZMinus[i-1]
	}

	return p
}

func (p *Regular3DBlockModelPatternPrecedence) NumBlocks() int {
	return p.numX * p.numY * p.numZ
}

func (p *Regular3DBlockModelPatternPrecedence) xyz(k int) (int, int, int) {
	return k % p.numX, (k / p.numX) % p.numY, k / (p.numX * p.numY)
}

func (p *Regular3DBlockModelPatternPrecedence) Antecedents(fromBlockIndex int) iter.Seq[int] {
	x, y, z := p.xyz(fromBlockIndex)

	return func(yield func(int) bool) {
		if z == p.numZ-1 {
			return
		}

		zMinus := p.numZ - z - 1
		n := len(p.offsets)
		if zMinus <= p.maxOffsetZ {
			n = p.numOffsetsByZMinus[zMinus]
		}

		if x >= p.xLo && x < p.xHi && y >= p.yLo && y < p.yHi {
			// Inner region: no bounds checks required (BlockOffsetSource).
			for i := 0; i < n; i++ {
				if !yield(fromBlockIndex + p.precomputed1DOffsets[i]) {
					return
				}
			}
			return
		}

		// Boundary: skip offsets that fall outside the model in x/y
		// (BlockOffsetExtentSource).
		i := 0
		for i < n {
			off := p.offsets[i]
			tx, ty := x+off.X, y+off.Y
			if tx < 0 || tx >= p.numX || ty < 0 || ty >= p.numY {
				i++
			} else {
				break
			}
		}
		for i < n {
			off := p.offsets[i]
			tx, ty, tz := x+off.X, y+off.Y, z+off.Z
			if !yield(tx + ty*p.numX + tz*p.numX*p.numY) {
				return
			}
			i++
			for i < n {
				off := p.offsets[i]
				tx, ty := x+off.X, y+off.Y
				if tx < 0 || tx >= p.numX || ty < 0 || ty >= p.numY {
					i++
				} else {
					break
				}
			}
		}
	}
}

// Successors is not supported by this class in C++ (returns an empty
// iterator); it yields nothing here as well.
func (p *Regular3DBlockModelPatternPrecedence) Successors(toBlockIndex int) iter.Seq[int] {
	return func(func(int) bool) {}
}

func (p *Regular3DBlockModelPatternPrecedence) ApproxNumAntecedents(fromBlockIndex int) int {
	return len(p.offsets)
}

// Regular3DBlockModelKeyedPatternsPrecedence selects a pattern per block via
// patternIndices (mineflow.cpp:1940-1979). Used for locally-varying slopes.
type Regular3DBlockModelKeyedPatternsPrecedence struct {
	patterns       []Regular3DBlockModelPatternPrecedence
	patternIndices []int
}

func NewRegular3DBlockModelKeyedPatternsPrecedence(blockDef BlockDefinition, patterns []PrecedencePattern, patternIndices []int) *Regular3DBlockModelKeyedPatternsPrecedence {
	if len(patterns) == 0 {
		panic("a non zero number of patterns are required")
	}
	if len(patternIndices) != blockDef.NumBlocks() {
		panic("invalid pattern indices count")
	}

	p := &Regular3DBlockModelKeyedPatternsPrecedence{
		patterns:       make([]Regular3DBlockModelPatternPrecedence, len(patterns)),
		patternIndices: patternIndices,
	}
	for i := range patterns {
		p.patterns[i] = *NewRegular3DBlockModelPatternPrecedence(blockDef, patterns[i])
	}
	return p
}

func (p *Regular3DBlockModelKeyedPatternsPrecedence) NumBlocks() int {
	return p.patterns[0].NumBlocks()
}

func (p *Regular3DBlockModelKeyedPatternsPrecedence) Antecedents(fromBlockIndex int) iter.Seq[int] {
	return p.patterns[p.patternIndices[fromBlockIndex]].Antecedents(fromBlockIndex)
}

func (p *Regular3DBlockModelKeyedPatternsPrecedence) Successors(toBlockIndex int) iter.Seq[int] {
	return p.patterns[p.patternIndices[toBlockIndex]].Successors(toBlockIndex)
}

func (p *Regular3DBlockModelKeyedPatternsPrecedence) ApproxNumAntecedents(fromBlockIndex int) int {
	return p.patterns[p.patternIndices[fromBlockIndex]].ApproxNumAntecedents(fromBlockIndex)
}

// ReachableSearchBuffer is a reusable BFS search buffer
// (PrecedenceConstraintsReachableSearchBuffer, mineflow.cpp:1080-1124). The
// seen vector uses a rotating tag to avoid clearing between searches.
type ReachableSearchBuffer struct {
	numBlocks int
	tag       uint8
	queue     []int
	seen      []uint8
}

func NewReachableSearchBuffer(numBlocks int) *ReachableSearchBuffer {
	return &ReachableSearchBuffer{
		numBlocks: numBlocks,
		tag:       101,
	}
}

func (b *ReachableSearchBuffer) newSearch() {
	if b.tag >= 100 {
		if b.seen == nil {
			b.seen = make([]uint8, b.numBlocks)
		}
		for i := range b.seen {
			b.seen[i] = 101
		}
		b.tag = 0
	} else {
		b.tag++
	}
	b.queue = b.queue[:0]
}

func (b *ReachableSearchBuffer) queueBlock(v int) {
	if b.seen[v] != b.tag {
		b.seen[v] = b.tag
		b.queue = append(b.queue, v)
	}
}

func (b *ReachableSearchBuffer) search() (int, bool) {
	if len(b.queue) == 0 {
		return 0, false
	}
	v := b.queue[0]
	b.queue = b.queue[1:]
	return v, true
}

func (b *ReachableSearchBuffer) hasMore() bool {
	return len(b.queue) > 0
}

// ReachableAntecedents yields every block reachable from fromBlockIndex by
// following antecedent edges (transitive closure of what must be mined), not
// including fromBlockIndex itself. Each block is yielded at most once.
func ReachableAntecedents(pre PrecedenceConstraints, fromBlockIndex int, buffer *ReachableSearchBuffer) iter.Seq[int] {
	return reachable(fromBlockIndex, pre.Antecedents, buffer)
}

// ReachableSuccessors yields every block reachable from toBlockIndex by
// following successor edges, not including toBlockIndex itself.
func ReachableSuccessors(pre PrecedenceConstraints, toBlockIndex int, buffer *ReachableSearchBuffer) iter.Seq[int] {
	return reachable(toBlockIndex, func(v int) iter.Seq[int] {
		return Successors(pre, v)
	}, buffer)
}

// ReachableBlockSource in the C++ code, mineflow.cpp:982-1018.
func reachable(start int, fn func(int) iter.Seq[int], buffer *ReachableSearchBuffer) iter.Seq[int] {
	return func(yield func(int) bool) {
		buffer.newSearch()
		for v := range fn(start) {
			buffer.queueBlock(v)
		}
		for {
			v, ok := buffer.search()
			if !ok {
				return
			}
			for t := range fn(v) {
				buffer.queueBlock(t)
			}
			if !yield(v) {
				return
			}
		}
	}
}

// PartialReachableAntecedents performs a reachability search over antecedents,
// invoking cback for each discovered block. Returning false from cback stops
// the search from continuing past that block.
func PartialReachableAntecedents(pre PrecedenceConstraints, fromBlockIndex int, cback func(toBlockIndex int) bool, buffer *ReachableSearchBuffer) {
	partialSearch(fromBlockIndex, cback, pre.Antecedents, buffer)
}

// PartialReachableSuccessors performs a reachability search over successors,
// invoking cback for each discovered block. Returning false from cback stops
// the search from continuing past that block.
func PartialReachableSuccessors(pre PrecedenceConstraints, toBlockIndex int, cback func(fromBlockIndex int) bool, buffer *ReachableSearchBuffer) {
	partialSearch(toBlockIndex, cback, func(v int) iter.Seq[int] {
		return Successors(pre, v)
	}, buffer)
}

// PartialSearch in the C++ code, mineflow.cpp:1040-1058.
func partialSearch(start int, cback func(int) bool, fn func(int) iter.Seq[int], buffer *ReachableSearchBuffer) {
	buffer.newSearch()
	for to := range fn(start) {
		buffer.queueBlock(to)
	}

	for {
		v, ok := buffer.search()
		if !ok {
			return
		}
		if cback(v) {
			for to := range fn(v) {
				buffer.queueBlock(to)
			}
		}
	}
}

// ConsistentPrecedenceConstraints checks (primarily for testing) that the
// precedence constraints are consistent: correct counts, successors and
// antecedents correctly related, all constraints valid
// (mineflow.cpp:1128-1177).
func ConsistentPrecedenceConstraints(pre PrecedenceConstraints) bool {
	preNumBlocks := pre.NumBlocks()
	preNumConstraints := NumPrecedenceConstraints(pre)

	antecedents := make(map[int]map[int]bool)
	successors := make(map[int]map[int]bool)
	mySuccessors := make(map[int]map[int]bool)

	for blockIndex := 0; blockIndex < preNumBlocks; blockIndex++ {
		nAnte := 0
		for target := range pre.Antecedents(blockIndex) {
			if antecedents[blockIndex] == nil {
				antecedents[blockIndex] = make(map[int]bool)
			}
			antecedents[blockIndex][target] = true
			if mySuccessors[target] == nil {
				mySuccessors[target] = make(map[int]bool)
			}
			mySuccessors[target][blockIndex] = true
			nAnte++
		}
		if nAnte != len(antecedents[blockIndex]) {
			return false
		}
		nSucc := 0
		for target := range Successors(pre, blockIndex) {
			if successors[blockIndex] == nil {
				successors[blockIndex] = make(map[int]bool)
			}
			successors[blockIndex][target] = true
			nSucc++
		}
		if nSucc != len(successors[blockIndex]) {
			return false
		}
	}

	for blockIndex := 0; blockIndex < preNumBlocks; blockIndex++ {
		if !setEq(successors[blockIndex], mySuccessors[blockIndex]) {
			return false
		}
	}

	actualNumber := 0
	for blockIndex := 0; blockIndex < preNumBlocks; blockIndex++ {
		actualNumber += len(antecedents[blockIndex])
	}
	if actualNumber != preNumConstraints {
		return false
	}

	// could check for cycles..

	return true
}

// setEq compares two sets, treating a missing set as empty (as the C++
// unordered_map comparison does).
func setEq(a, b map[int]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}
