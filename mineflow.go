package mineflow

import (
	"errors"
	"fmt"
	"sort"
)

// PrecedenceConstraints describes the required ordering for mining blocks.
// A constraint from "from" to "to" means that if block "from" is mined,
// block "to" must also be mined.
type PrecedenceConstraints interface {
	NumBlocks() int
	Antecedents(blockIndex int) []int
}

// BlockValues provides the economic values for each block.
type BlockValues interface {
	NumBlocks() int
	BlockValue(blockIndex int) int64
}

// SliceBlockValues is a simple in-memory implementation of BlockValues.
type SliceBlockValues []int64

func (v SliceBlockValues) NumBlocks() int { return len(v) }
func (v SliceBlockValues) BlockValue(blockIndex int) int64 {
	if blockIndex < 0 || blockIndex >= len(v) {
		return 0
	}
	return v[blockIndex]
}

// ExplicitPrecedence stores precedence constraints as adjacency lists.
type ExplicitPrecedence struct {
	numBlocks   int
	antecedents map[int][]int
}

func NewExplicitPrecedence(numBlocks int) *ExplicitPrecedence {
	return &ExplicitPrecedence{
		numBlocks:   numBlocks,
		antecedents: make(map[int][]int, numBlocks),
	}
}

func (p *ExplicitPrecedence) NumBlocks() int { return p.numBlocks }

func (p *ExplicitPrecedence) Antecedents(blockIndex int) []int {
	if blockIndex < 0 || blockIndex >= p.numBlocks {
		return nil
	}
	vals := p.antecedents[blockIndex]
	out := make([]int, len(vals))
	copy(out, vals)
	return out
}

func (p *ExplicitPrecedence) AddConstraint(from, to int) error {
	if from < 0 || from >= p.numBlocks || to < 0 || to >= p.numBlocks {
		return fmt.Errorf("precedence out of range: %d -> %d", from, to)
	}
	p.antecedents[from] = append(p.antecedents[from], to)
	return nil
}

// Vector3I is a simple 3D integer offset.
type Vector3I struct {
	X int
	Y int
	Z int
}

// BlockDefinition describes a regular 3D block model.
type BlockDefinition struct {
	NumX int
	NumY int
	NumZ int
}

func (b BlockDefinition) NumBlocks() int {
	return b.NumX * b.NumY * b.NumZ
}

func (b BlockDefinition) GridIndex(x, y, z int) int {
	return x + y*b.NumX + z*b.NumX*b.NumY
}

func (b BlockDefinition) XYZIndices(idx int) (int, int, int) {
	return idx % b.NumX, (idx / b.NumX) % b.NumY, idx / (b.NumX * b.NumY)
}

// PrecedencePattern stores a collection of 3D offsets that define a precedence template.
type PrecedencePattern struct {
	Offsets []Vector3I
}

func NewPrecedencePattern(offsets []Vector3I) PrecedencePattern {
	out := PrecedencePattern{Offsets: make([]Vector3I, len(offsets))}
	copy(out.Offsets, offsets)
	return out
}

func (p PrecedencePattern) OneFive() PrecedencePattern {
	return NewPrecedencePattern([]Vector3I{{0, -1, 1}, {-1, 0, 1}, {0, 0, 1}, {1, 0, 1}, {0, 1, 1}})
}

func (p PrecedencePattern) OneNine() PrecedencePattern {
	offsets := make([]Vector3I, 0, 9)
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			offsets = append(offsets, Vector3I{X: i, Y: j, Z: 1})
		}
	}
	return NewPrecedencePattern(offsets)
}

func (p PrecedencePattern) Size() int { return len(p.Offsets) }

// Regular3DBlockModelPatternPrecedence applies a precedence pattern to a regular 3D block model.
type Regular3DBlockModelPatternPrecedence struct {
	blockDef BlockDefinition
	pattern  PrecedencePattern
}

func NewRegular3DBlockModelPatternPrecedence(blockDef BlockDefinition, pattern PrecedencePattern) *Regular3DBlockModelPatternPrecedence {
	return &Regular3DBlockModelPatternPrecedence{blockDef: blockDef, pattern: pattern}
}

func (p *Regular3DBlockModelPatternPrecedence) NumBlocks() int { return p.blockDef.NumBlocks() }

func (p *Regular3DBlockModelPatternPrecedence) Antecedents(blockIndex int) []int {
	if blockIndex < 0 || blockIndex >= p.NumBlocks() {
		return nil
	}

	x, y, z := p.blockDef.XYZIndices(blockIndex)
	if z >= p.blockDef.NumZ-1 {
		return nil
	}

	antecedents := make([]int, 0, len(p.pattern.Offsets))
	for _, off := range p.pattern.Offsets {
		candidateX := x + off.X
		candidateY := y + off.Y
		candidateZ := z + off.Z
		if candidateX < 0 || candidateX >= p.blockDef.NumX || candidateY < 0 || candidateY >= p.blockDef.NumY || candidateZ < 0 || candidateZ >= p.blockDef.NumZ {
			continue
		}
		antecedents = append(antecedents, p.blockDef.GridIndex(candidateX, candidateY, candidateZ))
	}

	sort.Slice(antecedents, func(i, j int) bool { return antecedents[i] < antecedents[j] })
	return antecedents
}

// PseudoSolver implements the core ultimate-pit optimization using a min-cut
// formulation equivalent to the pseudoflow approach used by MineFlow.
type PseudoSolver struct {
	precedence PrecedenceConstraints
	values     []int64
}

func NewPseudoSolver(precedence PrecedenceConstraints, values []int64) (*PseudoSolver, error) {
	if precedence == nil {
		return nil, errors.New("precedence constraints are required")
	}
	if len(values) != precedence.NumBlocks() {
		return nil, fmt.Errorf("value count %d does not match block count %d", len(values), precedence.NumBlocks())
	}
	return &PseudoSolver{precedence: precedence, values: values}, nil
}

func NewPseudoSolverFromValues(precedence PrecedenceConstraints, values BlockValues) (*PseudoSolver, error) {
	if values == nil {
		return nil, errors.New("block values are required")
	}
	blockValues := make([]int64, values.NumBlocks())
	for i := 0; i < values.NumBlocks(); i++ {
		blockValues[i] = values.BlockValue(i)
	}
	return NewPseudoSolver(precedence, blockValues)
}

// Solve returns the blocks that belong to the maximum-profit closure.
func (s *PseudoSolver) Solve() ([]bool, error) {
	if s == nil || s.precedence == nil {
		return nil, errors.New("solver is not initialized")
	}

	n := s.precedence.NumBlocks()
	source := n
	sink := n + 1
	dinic := newDinic(n + 2)

	for i, value := range s.values {
		if value > 0 {
			dinic.addEdge(source, i, value)
		} else if value < 0 {
			dinic.addEdge(i, sink, -value)
		}
	}

	for from := 0; from < n; from++ {
		for _, to := range s.precedence.Antecedents(from) {
			if to < 0 || to >= n {
				return nil, fmt.Errorf("precedence target out of range: %d -> %d", from, to)
			}
			dinic.addEdge(from, to, int64(1<<60))
		}
	}

	_ = dinic.maxFlow(source, sink)

	seen := dinic.reachableFrom(source)
	inCut := make([]bool, n)
	for i := 0; i < n; i++ {
		inCut[i] = seen[i]
	}
	return inCut, nil
}

// SolveUltimatePit is a small convenience wrapper that mirrors the README example.
func SolveUltimatePit(values []int64, precedence [][]int64) []bool {
	p := NewExplicitPrecedence(len(values))
	for _, pair := range precedence {
		if len(pair) != 2 {
			continue
		}
		_ = p.AddConstraint(int(pair[0]), int(pair[1]))
	}
	solver, err := NewPseudoSolver(p, values)
	if err != nil {
		return make([]bool, len(values))
	}
	inCut, err := solver.Solve()
	if err != nil {
		return make([]bool, len(values))
	}
	return inCut
}

type edge struct {
	to  int
	cap int64
	rev int
}

type dinic struct {
	g [][]*edge
}

func newDinic(n int) *dinic {
	g := make([][]*edge, n)
	return &dinic{g: g}
}

func (d *dinic) addEdge(from, to int, cap int64) {
	fwd := &edge{to: to, cap: cap, rev: len(d.g[to])}
	rev := &edge{to: from, cap: 0, rev: len(d.g[from])}
	d.g[from] = append(d.g[from], fwd)
	d.g[to] = append(d.g[to], rev)
}

func (d *dinic) maxFlow(source, sink int) int64 {
	flow := int64(0)
	for {
		level := make([]int, len(d.g))
		for i := range level {
			level[i] = -1
		}
		level[source] = 0
		queue := []int{source}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			for _, e := range d.g[cur] {
				if e.cap > 0 && level[e.to] < 0 {
					level[e.to] = level[cur] + 1
					queue = append(queue, e.to)
				}
			}
		}
		if level[sink] < 0 {
			break
		}

		it := make([]int, len(d.g))
		var dfs func(int, int64) int64
		dfs = func(node int, pushed int64) int64 {
			if node == sink {
				return pushed
			}
			for ; it[node] < len(d.g[node]); it[node]++ {
				e := d.g[node][it[node]]
				if e.cap > 0 && level[e.to] == level[node]+1 {
					res := dfs(e.to, min64(pushed, e.cap))
					if res > 0 {
						e.cap -= res
						d.g[e.to][e.rev].cap += res
						return res
					}
				}
			}
			return 0
		}

		for {
			pushed := dfs(source, int64(1<<60))
			if pushed == 0 {
				break
			}
			flow += pushed
		}
	}
	return flow
}

func (d *dinic) reachableFrom(source int) []bool {
	seen := make([]bool, len(d.g))
	queue := []int{source}
	seen[source] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, e := range d.g[cur] {
			if e.cap > 0 && !seen[e.to] {
				seen[e.to] = true
				queue = append(queue, e.to)
			}
		}
	}
	return seen
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
