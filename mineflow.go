package mineflow

import (
	"errors"
	"fmt"
	"slices"
)

// infCapacity is used as an effectively infinite edge capacity in the flow network.
const infCapacity int64 = 1 << 60

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
// Block indices are dense [0, numBlocks), so a flat slice is used rather
// than a map for direct indexing.
type ExplicitPrecedence struct {
	antecedents [][]int
}

func NewExplicitPrecedence(numBlocks int) *ExplicitPrecedence {
	return &ExplicitPrecedence{
		antecedents: make([][]int, numBlocks),
	}
}

func (p *ExplicitPrecedence) NumBlocks() int { return len(p.antecedents) }

// Antecedents returns a copy of the antecedent list for the given block.
// Callers may freely mutate the returned slice.
func (p *ExplicitPrecedence) Antecedents(blockIndex int) []int {
	if blockIndex < 0 || blockIndex >= len(p.antecedents) {
		return nil
	}
	return slices.Clone(p.antecedents[blockIndex])
}

func (p *ExplicitPrecedence) AddConstraint(from, to int) error {
	n := len(p.antecedents)
	if from < 0 || from >= n || to < 0 || to >= n {
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

// InBounds reports whether the 3D indices fall within the block model dimensions.
func (b BlockDefinition) InBounds(x, y, z int) bool {
	return x >= 0 && x < b.NumX && y >= 0 && y < b.NumY && z >= 0 && z < b.NumZ
}

// PrecedencePattern stores a collection of 3D offsets that define a precedence template.
type PrecedencePattern struct {
	Offsets []Vector3I
}

func NewPrecedencePattern(offsets []Vector3I) PrecedencePattern {
	return PrecedencePattern{Offsets: slices.Clone(offsets)}
}

// OneFivePrecedencePattern returns the standard 1-to-5 precedence pattern.
func OneFivePrecedencePattern() PrecedencePattern {
	return NewPrecedencePattern([]Vector3I{
		{0, -1, 1}, {-1, 0, 1}, {0, 0, 1}, {1, 0, 1}, {0, 1, 1},
	})
}

// OneNinePrecedencePattern returns the standard 1-to-9 precedence pattern.
func OneNinePrecedencePattern() PrecedencePattern {
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
		cx, cy, cz := x+off.X, y+off.Y, z+off.Z
		if !p.blockDef.InBounds(cx, cy, cz) {
			continue
		}
		antecedents = append(antecedents, p.blockDef.GridIndex(cx, cy, cz))
	}
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
	for i := range blockValues {
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
	d := newDinic(n + 2)

	for i, value := range s.values {
		if value > 0 {
			d.addEdge(source, i, value)
		} else if value < 0 {
			d.addEdge(i, sink, -value)
		}
	}

	for from := 0; from < n; from++ {
		for _, to := range s.precedence.Antecedents(from) {
			if to < 0 || to >= n {
				return nil, fmt.Errorf("precedence target out of range: %d -> %d", from, to)
			}
			d.addEdge(from, to, infCapacity)
		}
	}

	_ = d.maxFlow(source, sink)

	return d.reachableFrom(source)[:n], nil
}

// SolveUltimatePit is a convenience wrapper that mirrors the README example.
// It returns the blocks in the optimal pit and any error encountered.
func SolveUltimatePit(values []int64, precedence [][]int64) ([]bool, error) {
	p := NewExplicitPrecedence(len(values))
	for _, pair := range precedence {
		if len(pair) != 2 {
			return nil, fmt.Errorf("precedence pair has %d elements, want 2", len(pair))
		}
		if err := p.AddConstraint(int(pair[0]), int(pair[1])); err != nil {
			return nil, err
		}
	}
	solver, err := NewPseudoSolver(p, values)
	if err != nil {
		return nil, err
	}
	return solver.Solve()
}

// --- Dinic max-flow implementation using a flat edge pool ---
//
// Edges are stored contiguously to avoid per-edge heap allocations.
// Adjacency is tracked via a linked-list-per-node using head/next arrays.

type edge struct {
	to  int
	cap int64
	rev int // index into edges slice of the reverse edge
}

type dinic struct {
	head  []int  // head[node] = index of first edge, -1 if none
	next  []int  // next[edgeIdx] = index of next edge from same node, -1 if none
	edges []edge // flat contiguous edge pool
}

func newDinic(n int) *dinic {
	head := make([]int, n)
	for i := range head {
		head[i] = -1
	}
	return &dinic{head: head}
}

func (d *dinic) addEdge(from, to int, cap int64) {
	fwdIdx := len(d.edges)
	revIdx := fwdIdx + 1
	d.edges = append(d.edges,
		edge{to: to, cap: cap, rev: revIdx},
		edge{to: from, cap: 0, rev: fwdIdx},
	)
	d.next = append(d.next, d.head[from], d.head[to])
	d.head[from] = fwdIdx
	d.head[to] = revIdx
}

func (d *dinic) maxFlow(source, sink int) int64 {
	n := len(d.head)
	flow := int64(0)

	// Pre-allocate BFS and DFS scratch space; reused across iterations.
	level := make([]int, n)
	queue := make([]int, 0, n)
	it := make([]int, n)

	for {
		// BFS to build level graph.
		for i := range level {
			level[i] = -1
		}
		level[source] = 0
		queue = queue[:0]
		queue = append(queue, source)
		for front := 0; front < len(queue); front++ {
			cur := queue[front]
			for ei := d.head[cur]; ei != -1; ei = d.next[ei] {
				e := &d.edges[ei]
				if e.cap > 0 && level[e.to] < 0 {
					level[e.to] = level[cur] + 1
					queue = append(queue, e.to)
				}
			}
		}
		if level[sink] < 0 {
			break
		}

		// DFS to push blocking flow.
		copy(it, d.head)
		var dfs func(int, int64) int64
		dfs = func(node int, pushed int64) int64 {
			if node == sink {
				return pushed
			}
			for it[node] != -1 {
				ei := it[node]
				e := &d.edges[ei]
				if e.cap > 0 && level[e.to] == level[node]+1 {
					res := dfs(e.to, min(pushed, e.cap))
					if res > 0 {
						e.cap -= res
						d.edges[e.rev].cap += res
						return res
					}
				}
				it[node] = d.next[ei]
			}
			return 0
		}

		for {
			pushed := dfs(source, infCapacity)
			if pushed == 0 {
				break
			}
			flow += pushed
		}
	}
	return flow
}

func (d *dinic) reachableFrom(source int) []bool {
	seen := make([]bool, len(d.head))
	queue := make([]int, 0, len(d.head))
	seen[source] = true
	queue = append(queue, source)
	for front := 0; front < len(queue); front++ {
		cur := queue[front]
		for ei := d.head[cur]; ei != -1; ei = d.next[ei] {
			e := &d.edges[ei]
			if e.cap > 0 && !seen[e.to] {
				seen[e.to] = true
				queue = append(queue, e.to)
			}
		}
	}
	return seen
}
