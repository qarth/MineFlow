package mineflow

import (
	"errors"
	"fmt"
	"time"
)

// solver.go — port of the pseudoflow solver (Hochbaum's algorithm, modified
// for the ultimate pit problem) from mineflow.cpp:78-182 and 2096-2995.
//
// The C++ pointer-based Node/Arc pools are ported as index-based arenas: a
// single arcs slice holds the per-node root arcs at indices [0, numNodes) and
// precedence arcs appended after that. -1 is the null index. As in the C++
// PrecedenceArcPool, deleted arcs are never reclaimed (DeleteArc just clears
// them).
//
// Values and flow arithmetic use int64 (the default non-GMP C++ build). Note:
// SolveLargest (via SolveLargestValuesAdapter) multiplies block values by
// ~numBlocks and can overflow int64 on large models — the same limitation as
// the C++ build without GMP.

const nullIndex = -1

// node is impl::Node in the C++ code. Children form an intrusive singly
// linked list via firstChild/nextChild; nextScan is the scan cursor used by
// ProcessStrongRoot.
type node struct {
	excess     int64 // Positive for excess, negative for deficit
	toRoot     int   // arc index of the normalized tree arc
	label      int   // the 'distance' label
	firstChild int
	nextChild  int
	nextScan   int

	// Lazily initialized predecessor list (AntecedentsInfo in C++).
	outOfTree []int
	nextArc   int
	precInit  bool
}

// arc is impl::Arc in the C++ code. tail/head of nullIndex mark one of the
// main 'root' arcs (source/sink).
type arc struct {
	tail int
	head int
	flow int64
}

// nodePool is impl::NodePool + impl::PrecedenceArcPool in the C++ code.
type nodePool struct {
	pre      PrecedenceConstraints
	numNodes int

	labelCount []int
	buckets    [][]int // queues of strong-root node indices by label

	nodes []node
	arcs  []arc // [0, numNodes) are the root arcs; the rest are precedence arcs

	numPrecArcsUsed int
}

func newNodePool(pre PrecedenceConstraints) *nodePool {
	numNodes := pre.NumBlocks()
	p := &nodePool{
		pre:        pre,
		numNodes:   numNodes,
		labelCount: make([]int, 2),
		buckets:    make([][]int, 2),
		nodes:      make([]node, numNodes),
		arcs:       make([]arc, numNodes),
	}
	for i := range p.nodes {
		p.nodes[i] = node{
			excess:     0,
			toRoot:     i,
			label:      0,
			firstChild: nullIndex,
			nextChild:  nullIndex,
			nextScan:   nullIndex,
		}
		p.arcs[i] = arc{tail: nullIndex, head: nullIndex, flow: 0}
	}
	return p
}

// initializeNodeValue sets the node's initial excess from its block value and
// wires up its root arc (NodePool::InitializeNodeValue, mineflow.cpp:2789).
func (p *nodePool) initializeNodeValue(nodeIndex int, value int64) {
	n := &p.nodes[nodeIndex]
	a := &p.arcs[nodeIndex]

	n.excess = value

	if n.excess > 0 {
		n.label = 1
		p.labelCount[1]++
		p.pushStrongRoot(nodeIndex)

		a.tail = nullIndex
		a.head = nodeIndex
		a.flow = n.excess
	} else {
		n.label = 0
		p.labelCount[0]++

		a.tail = nodeIndex
		a.head = nullIndex
		a.flow = -n.excess
	}
}

// getNodeValue returns the original block value of a node
// (NodePool::GetNodeValue, mineflow.cpp:2837).
func (p *nodePool) getNodeValue(nodeIndex int) int64 {
	a := &p.arcs[nodeIndex]
	if a.tail != nullIndex {
		return -a.flow
	}
	return a.flow
}

func (p *nodePool) reconnectToRoot(nodeIndex int) {
	p.nodes[nodeIndex].toRoot = nodeIndex
}

func (p *nodePool) inMinimumCut(nodeIndex int) bool {
	return p.nodes[nodeIndex].label == p.numNodes
}

func (p *nodePool) pushStrongRoot(nodeIndex int) {
	label := p.nodes[nodeIndex].label
	for len(p.buckets) <= label {
		p.buckets = append(p.buckets, nil)
	}
	p.buckets[label] = append(p.buckets[label], nodeIndex)
}

func (p *nodePool) incrementLabel(nodeIndex int) {
	n := &p.nodes[nodeIndex]
	p.labelCount[n.label]--
	n.label++
	for len(p.labelCount) <= n.label {
		p.labelCount = append(p.labelCount, 0)
	}
	p.labelCount[n.label]++
}

// nextStrongRoot returns the next strong root to process, finalizing the
// labels of subtrees that can no longer merge (label == numNodes means in the
// minimum cut). Port of NodePool::NextStrongRoot (mineflow.cpp:2864-2907) —
// this is where InMinimumCut correctness comes from.
func (p *nodePool) nextStrongRoot() (int, bool) {
	for i := len(p.buckets) - 1; i > 0; i-- {
		queue := p.buckets[i]
		if len(queue) > 0 {
			if p.labelCount[i-1] > 0 {
				node := queue[0]
				p.buckets[i] = queue[1:]
				return node, true
			}

			for len(queue) > 0 {
				root := queue[0]
				p.forNodeAndChildren(root, func(v int) {
					p.labelCount[p.nodes[v].label]--
					p.nodes[v].label = p.numNodes
				})
				queue = queue[1:]
			}
			p.buckets[i] = queue
		} else {
			p.buckets = p.buckets[:i]
		}
	}

	if len(p.buckets[0]) == 0 {
		return nullIndex, false
	}

	queue := p.buckets[0]
	for len(queue) > 0 {
		root := queue[0]
		queue = queue[1:]

		p.incrementLabel(root)
		p.pushStrongRoot(root)
	}
	p.buckets[0] = queue

	node := p.buckets[1][0]
	p.buckets[1] = p.buckets[1][1:]
	return node, true
}

// initPrecedence lazily materializes the antecedent list of a node
// (NodePool::InitPrecedence, mineflow.cpp:2920).
func (p *nodePool) initPrecedence(nodeIndex int) {
	n := &p.nodes[nodeIndex]
	if ap, ok := p.pre.(ApproxAntecedentsProvider); ok {
		if cap(n.outOfTree) < ap.ApproxNumAntecedents(nodeIndex) {
			n.outOfTree = make([]int, 0, ap.ApproxNumAntecedents(nodeIndex))
		}
	}
	for targetIndex := range p.pre.Antecedents(nodeIndex) {
		n.outOfTree = append(n.outOfTree, targetIndex)
	}
}

// addChild / removeChild maintain the intrusive child lists
// (Node::AddChild / Node::RemoveChild, mineflow.cpp:2605-2632).
func (p *nodePool) addChild(parent, child int) {
	p.nodes[child].nextChild = p.nodes[parent].firstChild
	p.nodes[parent].firstChild = child
}

func (p *nodePool) removeChild(parent, child int) {
	if p.nodes[parent].firstChild == child {
		p.nodes[parent].firstChild = p.nodes[child].nextChild
		p.nodes[child].nextChild = nullIndex
		return
	}

	current := p.nodes[parent].firstChild
	for p.nodes[current].nextChild != child {
		current = p.nodes[current].nextChild
	}

	p.nodes[current].nextChild = p.nodes[child].nextChild
	p.nodes[child].nextChild = nullIndex
}

// forNodeAndChildren visits a node and its whole subtree
// (Node::ForNodeAndChildren, mineflow.cpp:2639 — recursive there, an explicit
// stack here).
func (p *nodePool) forNodeAndChildren(nodeIndex int, cback func(int)) {
	stack := []int{nodeIndex}
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		cback(v)
		for c := p.nodes[v].firstChild; c != nullIndex; c = p.nodes[c].nextChild {
			stack = append(stack, c)
		}
	}
}

// findWeakAbove scans the out-of-tree antecedents for a node with label one
// below, swap-removing and returning it (Node::FindWeakAbove,
// mineflow.cpp:2649).
func (p *nodePool) findWeakAbove(nodeIndex int) int {
	n := &p.nodes[nodeIndex]
	if !n.precInit {
		p.initPrecedence(nodeIndex)
		n.precInit = true
	}
	for i := n.nextArc; i < len(n.outOfTree); i++ {
		to := n.outOfTree[i]
		if p.nodes[to].label == n.label-1 {
			n.nextArc = i
			n.outOfTree[i] = n.outOfTree[len(n.outOfTree)-1]
			n.outOfTree = n.outOfTree[:len(n.outOfTree)-1]
			return to
		}
	}

	n.nextArc = len(n.outOfTree)
	return nullIndex
}

// newArc allocates a precedence arc (PrecedenceArcPool::NewArc).
func (p *nodePool) newArc(tail, head int) int {
	p.arcs = append(p.arcs, arc{tail: tail, head: head, flow: 0})
	p.numPrecArcsUsed++
	return len(p.arcs) - 1
}

// deleteArc clears an arc; the memory is never reclaimed, as in the C++ pool
// (PrecedenceArcPool::DeleteArc).
func (p *nodePool) deleteArc(arcIndex int) {
	p.arcs[arcIndex] = arc{tail: nullIndex, head: nullIndex, flow: 0}
}

// SolveInfo holds statistics from a PseudoSolver.Solve call
// (PseudoSolverSolveInfo in C++).
type SolveInfo struct {
	ElapsedSeconds               float64
	NumNodes                     int
	NumContainedNodes            int
	NumUsedPrecedenceConstraints int
	ContainedValue               int64
}

func (i SolveInfo) String() string {
	return fmt.Sprintf(`PseudoSolverSolveInfo: %d input nodes
  Contained : %d / %d
  Used : %d precedence constraints
  Value : %d
  Elapsed : %8.2fs`, i.NumNodes, i.NumContainedNodes, i.NumNodes,
		i.NumUsedPrecedenceConstraints, i.ContainedValue, i.ElapsedSeconds)
}

// PseudoSolver implements the ultimate-pit optimization using Hochbaum's
// pseudoflow algorithm.
type PseudoSolver struct {
	pool *nodePool
	pre  PrecedenceConstraints

	nodePoolHasBeenInitialized bool
	minCutHasBeenSolved        bool
	largestSolution            []uint8
}

// NewPseudoSolver creates a solver over the given precedence constraints,
// initialized with the given block values (read once to init the structure).
// values may be nil, in which case UpdateValues must be called before Solve.
func NewPseudoSolver(pre PrecedenceConstraints, values BlockValues) (*PseudoSolver, error) {
	if pre == nil {
		return nil, errors.New("precedence constraints must be defined")
	}
	s := &PseudoSolver{
		pool: newNodePool(pre),
		pre:  pre,
	}
	if values != nil {
		if err := s.UpdateValues(values); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// NumNodes returns the number of nodes (blocks) in the solver.
func (s *PseudoSolver) NumNodes() int {
	return s.pool.numNodes
}

// UpdateValues re-initializes the solver with new block values. You must call
// Solve again afterwards.
func (s *PseudoSolver) UpdateValues(values BlockValues) error {
	if values == nil {
		return errors.New("values must be non nil")
	}
	if values.NumBlocks() != s.pool.numNodes {
		return errors.New("argument num blocks disagree")
	}

	if !s.nodePoolHasBeenInitialized {
		for nodeIndex := 0; nodeIndex < s.pool.numNodes; nodeIndex++ {
			s.pool.initializeNodeValue(nodeIndex, values.BlockValue(nodeIndex))
		}
		s.nodePoolHasBeenInitialized = true
	} else {
		// As in the C++ code, incremental re-normalization is an open
		// question; for now we just reset everything.
		s.pool = newNodePool(s.pre)
		for nodeIndex := 0; nodeIndex < s.pool.numNodes; nodeIndex++ {
			s.pool.initializeNodeValue(nodeIndex, values.BlockValue(nodeIndex))
		}
	}
	s.minCutHasBeenSolved = false
	return nil
}

// walkToRoot re-orients the tree arcs along the path from strongNode to the
// root, attaching the path to weakNode via newArc; returns the (former) root
// of the strong tree (PseudoSolver::WalkToRoot, mineflow.cpp:2207).
func (s *PseudoSolver) walkToRoot(strongNode, weakNode, newArc int) int {
	p := s.pool
	current := strongNode
	newParent := weakNode

	q := p.arcs[p.nodes[current].toRoot]
	for q.tail != nullIndex && q.head != nullIndex {
		oldArc := p.nodes[current].toRoot
		p.nodes[current].toRoot = newArc
		var oldParent int
		if q.tail == current {
			oldParent = q.head
		} else {
			oldParent = q.tail
		}

		p.removeChild(oldParent, current)
		p.addChild(newParent, current)

		newParent = current
		current = oldParent
		newArc = oldArc
		q = p.arcs[p.nodes[current].toRoot]
	}

	p.nodes[current].toRoot = newArc
	p.addChild(newParent, current)
	return current
}

// split disconnects current from parent when the arc flow is insufficient,
// reconnecting current to the root as a new strong root
// (PseudoSolver::Split, mineflow.cpp:2235).
func (s *PseudoSolver) split(current, parent, arcIndex int) {
	p := s.pool
	p.nodes[current].excess -= p.arcs[arcIndex].flow
	p.nodes[parent].excess += p.arcs[arcIndex].flow
	p.deleteArc(arcIndex)
	p.nodes[parent].outOfTree = append(p.nodes[parent].outOfTree, current)
	p.removeChild(parent, current)
	p.reconnectToRoot(current)
	p.pushStrongRoot(current)
}

// pushFlow pushes excess up the path from strongRoot
// (PseudoSolver::PushFlow, mineflow.cpp:2251).
func (s *PseudoSolver) pushFlow(strongRoot int) {
	p := s.pool
	prevExcess := int64(1)
	current := strongRoot
	for {
		tr := p.nodes[current].toRoot
		parent := p.arcs[tr].tail
		if parent == current {
			parent = p.arcs[tr].head
		}
		if p.nodes[current].excess > 0 && parent != nullIndex {
			prevExcess = p.nodes[parent].excess

			if p.arcs[tr].tail == current {
				// up
				p.nodes[parent].excess += p.nodes[current].excess
				p.arcs[tr].flow += p.nodes[current].excess
				p.nodes[current].excess = 0
			} else {
				if p.arcs[tr].flow >= p.nodes[current].excess {
					p.nodes[parent].excess += p.nodes[current].excess
					p.arcs[tr].flow -= p.nodes[current].excess
					p.nodes[current].excess = 0
				} else {
					s.split(current, parent, tr)
				}
			}
		} else {
			break
		}
		current = parent
	}

	if p.nodes[current].excess > 0 && prevExcess <= 0 {
		p.pushStrongRoot(current)
	}
}

// merge attaches the strong node's tree to the weak node and pushes flow
// (PseudoSolver::Merge, mineflow.cpp:2332).
func (s *PseudoSolver) merge(strongNode, weakNode int) {
	newArc := s.pool.newArc(strongNode, weakNode)
	strongRoot := s.walkToRoot(strongNode, weakNode, newArc)
	s.pushFlow(strongRoot)
}

// processChildren advances the scan cursor over the children of node,
// bumping its label when all remaining children have higher labels
// (PseudoSolver::ProcessChildren, mineflow.cpp:2339).
func (s *PseudoSolver) processChildren(nodeIndex int) {
	p := s.pool
	n := &p.nodes[nodeIndex]

	// Loop over the remaining children (might be all of them!)
	for n.nextScan != nullIndex {
		if p.nodes[n.nextScan].label == n.label {
			return
		}
		n.nextScan = p.nodes[n.nextScan].nextChild
	}

	p.incrementLabel(nodeIndex)
	n.nextArc = 0
}

// processStrongRoot scans the strong tree for a merge, otherwise bumps
// labels (PseudoSolver::ProcessStrongRoot, mineflow.cpp:2357).
func (s *PseudoSolver) processStrongRoot(strongRoot int) {
	p := s.pool
	inLabel := p.nodes[strongRoot].label
	p.nodes[strongRoot].nextScan = p.nodes[strongRoot].firstChild

	weak := p.findWeakAbove(strongRoot)
	if weak != nullIndex {
		s.merge(strongRoot, weak)
		return
	}

	strongNode := strongRoot
	s.processChildren(strongRoot)

	for strongNode != nullIndex {
		for p.nodes[strongNode].nextScan != nullIndex {
			temp := p.nodes[strongNode].nextScan
			p.nodes[strongNode].nextScan = p.nodes[temp].nextChild
			strongNode = temp
			p.nodes[strongNode].nextScan = p.nodes[strongNode].firstChild

			weak = p.findWeakAbove(strongNode)
			if weak != nullIndex {
				s.merge(strongNode, weak)
				return
			}

			s.processChildren(strongNode)
		}

		tr := p.arcs[p.nodes[strongNode].toRoot]
		temp := tr.head
		if temp == strongNode {
			temp = tr.tail
		}
		strongNode = temp

		if strongNode != nullIndex {
			s.processChildren(strongNode)
		}
	}

	if p.nodes[strongRoot].label <= inLabel {
		panic("processStrongRoot: label did not increase")
	}
	p.pushStrongRoot(strongRoot)
}

// Solve runs the pseudoflow algorithm and returns solve statistics.
func (s *PseudoSolver) Solve() (*SolveInfo, error) {
	start := time.Now()
	if s.nodePoolHasBeenInitialized {
		for {
			strongRoot, ok := s.pool.nextStrongRoot()
			if !ok {
				break
			}
			s.processStrongRoot(strongRoot)
		}
	}
	elapsed := time.Since(start)

	info := &SolveInfo{
		ElapsedSeconds:               elapsed.Seconds(),
		NumNodes:                     s.pool.numNodes,
		NumUsedPrecedenceConstraints: s.pool.numPrecArcsUsed,
	}
	for nodeIndex := 0; nodeIndex < s.pool.numNodes; nodeIndex++ {
		if s.pool.inMinimumCut(nodeIndex) {
			info.NumContainedNodes++
			info.ContainedValue += s.pool.getNodeValue(nodeIndex)
		}
	}
	s.minCutHasBeenSolved = true
	return info, nil
}

// InMinimumCut reports whether the node belongs to the minimum cut (i.e. the
// block is mined). Only valid after Solve.
func (s *PseudoSolver) InMinimumCut(nodeIndex int) bool {
	return s.pool.inMinimumCut(nodeIndex)
}

// Constants for the largest-solution state machine (mineflow.cpp:2454-2457).
const (
	largestDefinitelyOut = 0
	largestDefinitelyIn  = 1
	largestInProcess     = 2
	largestUnknown       = 10
)

// SolveLargest solves for the largest minimum cut. Warning: with int64 values
// (via SolveLargestValuesAdapter) this can overflow on large models — the same
// limitation as the C++ build without GMP.
func (s *PseudoSolver) SolveLargest() (*SolveInfo, error) {
	start := time.Now()
	if !s.minCutHasBeenSolved {
		if _, err := s.Solve(); err != nil {
			return nil, err
		}
	}

	p := s.pool
	numNodes := p.numNodes

	if len(s.largestSolution) != numNodes {
		s.largestSolution = make([]uint8, numNodes)
	}
	for i := range s.largestSolution {
		s.largestSolution[i] = largestUnknown
	}

	var toCheck [][]int
	for nodeIndex := 0; nodeIndex < numNodes; nodeIndex++ {
		if s.largestSolution[nodeIndex] == largestUnknown {
			if p.inMinimumCut(nodeIndex) {
				s.largestSolution[nodeIndex] = largestDefinitelyIn
			} else {
				// Walk up to the root of this tree.
				n := nodeIndex
				q := p.arcs[p.nodes[n].toRoot]
				for q.tail != nullIndex && q.head != nullIndex {
					if q.tail == n {
						n = q.head
					} else {
						n = q.tail
					}
					q = p.arcs[p.nodes[n].toRoot]
				}

				nExcessZero := p.nodes[n].excess == 0
				setBranchTo := uint8(largestDefinitelyOut)
				if nExcessZero {
					setBranchTo = largestInProcess
				}

				var thisBranch []int
				p.forNodeAndChildren(n, func(v int) {
					s.largestSolution[v] = setBranchTo
					if nExcessZero {
						thisBranch = append(thisBranch, v)
					}
				})
				if nExcessZero {
					toCheck = append(toCheck, thisBranch)
				}
			}
		}
	}

	// These roots have an excess of zero
	if len(toCheck) > 0 {
		buffer := NewReachableSearchBuffer(numNodes)
		for _, branch := range toCheck {
			whatItIs := uint8(largestUnknown)
			for _, v := range branch {
				if s.largestSolution[v] != largestInProcess {
					whatItIs = s.largestSolution[v]
					break
				}
			}
			if whatItIs != largestUnknown {
				for _, v := range branch {
					s.largestSolution[v] = whatItIs
				}
			} else {
				foundDefOut := false

				var thisSearch []int

				for _, l := range branch {
					thisSearch = append(thisSearch, l)

					// do search
					if !foundDefOut {
						PartialReachableAntecedents(s.pre, l, func(v int) bool {
							switch s.largestSolution[v] {
							case largestDefinitelyOut:
								foundDefOut = true
								return false
							case largestDefinitelyIn:
								return false
							case largestInProcess:
								thisSearch = append(thisSearch, v)
								return !foundDefOut
							}
							return false
						}, buffer)
					}
				}

				setSearchTo := uint8(largestDefinitelyIn)
				if foundDefOut {
					setSearchTo = largestDefinitelyOut
				}
				for _, v := range thisSearch {
					s.largestSolution[v] = setSearchTo
				}
			}
		}
	}
	elapsed := time.Since(start)

	info := &SolveInfo{
		ElapsedSeconds: elapsed.Seconds(),
		NumNodes:       numNodes,
	}
	for nodeIndex := 0; nodeIndex < numNodes; nodeIndex++ {
		if s.largestSolution[nodeIndex] > 0 {
			info.NumContainedNodes++
			info.ContainedValue += p.getNodeValue(nodeIndex)
		}
	}
	return info, nil
}

// InLargestMinimumCut reports whether the node belongs to the largest minimum
// cut. Only valid after SolveLargest; panics otherwise (the C++ code throws).
func (s *PseudoSolver) InLargestMinimumCut(nodeIndex int) bool {
	if len(s.largestSolution) == s.pool.numNodes {
		return s.largestSolution[nodeIndex] > 0
	}
	panic("call solve largest")
}

// SolveLargestValuesAdapter adapts block values for SolveLargest
// (mineflow.cpp:2932-2995, non-GMP path): the underlying value v becomes
// v*(nNonNeg+1)+1 for v >= 0 and v*(nNonNeg+1) for v < 0, where nNonNeg is
// the number of non-negative blocks. With int64 values this can overflow on
// large models.
type SolveLargestValuesAdapter struct {
	values               BlockValues
	numNonNegativeBlocks int64
}

func NewSolveLargestValuesAdapter(values BlockValues) *SolveLargestValuesAdapter {
	if values == nil {
		panic("must supply values to solve largest adapter")
	}
	a := &SolveLargestValuesAdapter{values: values}
	n := values.NumBlocks()
	for i := 0; i < n; i++ {
		if values.BlockValue(i) >= 0 {
			a.numNonNegativeBlocks++
		}
	}
	a.numNonNegativeBlocks++
	return a
}

func (a *SolveLargestValuesAdapter) NumBlocks() int {
	return a.values.NumBlocks()
}

func (a *SolveLargestValuesAdapter) BlockValue(blockIndex int) int64 {
	v := a.values.BlockValue(blockIndex)
	if v >= 0 {
		return v*a.numNonNegativeBlocks + 1
	}
	return v * a.numNonNegativeBlocks
}
