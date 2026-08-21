package mineflow

import (
	"fmt"
	"iter"
)

// mineflow.go — port of the fundamental interfaces from mineflow.h
// (IBlockValues, IPrecedenceConstraints) plus the ExplicitPrecedence
// implementation and convenience wrappers.
//
// The C++ input-iterator hierarchy (BlockIndexInputIteratorBase etc.) is
// replaced with Go iter.Seq sequences.

// PrecedenceConstraint is a single constraint: if the block at From is mined,
// the block at To must also be mined.
type PrecedenceConstraint struct {
	From int
	To   int
}

// PrecedenceConstraints describes the required ordering for mining blocks.
// Antecedents(from) yields the blocks that must be mined if "from" is mined.
// Sequences are lightweight and single-use, mirroring the C++ input iterators.
type PrecedenceConstraints interface {
	NumBlocks() int
	Antecedents(fromBlockIndex int) iter.Seq[int]
}

// SuccessorsProvider is an optional interface for precedence constraints that
// can efficiently enumerate successors (blocks that require "to").
type SuccessorsProvider interface {
	Successors(toBlockIndex int) iter.Seq[int]
}

// ApproxAntecedentsProvider is an optional interface for precedence
// constraints that can cheaply estimate the number of antecedents.
type ApproxAntecedentsProvider interface {
	ApproxNumAntecedents(fromBlockIndex int) int
}

// AntecedentsSlice materializes the antecedents of a block into a slice.
func AntecedentsSlice(pre PrecedenceConstraints, fromBlockIndex int) []int {
	out := make([]int, 0, approxNumAntecedents(pre, fromBlockIndex))
	for to := range pre.Antecedents(fromBlockIndex) {
		out = append(out, to)
	}
	return out
}

// NumAntecedents counts the antecedents of a block. Generally requires
// iterating, so it should be avoided in hot paths.
func NumAntecedents(pre PrecedenceConstraints, fromBlockIndex int) int {
	n := 0
	for range pre.Antecedents(fromBlockIndex) {
		n++
	}
	return n
}

func approxNumAntecedents(pre PrecedenceConstraints, fromBlockIndex int) int {
	if p, ok := pre.(ApproxAntecedentsProvider); ok {
		return p.ApproxNumAntecedents(fromBlockIndex)
	}
	return 0
}

// Successors yields the successors of a block. If the constraints implement
// SuccessorsProvider that is used; otherwise it falls back to scanning all
// precedence constraints (expensive).
func Successors(pre PrecedenceConstraints, toBlockIndex int) iter.Seq[int] {
	if p, ok := pre.(SuccessorsProvider); ok {
		return p.Successors(toBlockIndex)
	}
	return func(yield func(int) bool) {
		for c := range AllConstraints(pre) {
			if c.To == toBlockIndex {
				if !yield(c.From) {
					return
				}
			}
		}
	}
}

// SuccessorsSlice materializes the successors of a block into a slice.
func SuccessorsSlice(pre PrecedenceConstraints, toBlockIndex int) []int {
	out := make([]int, 0)
	for from := range Successors(pre, toBlockIndex) {
		out = append(out, from)
	}
	return out
}

// AllConstraints enumerates every precedence constraint. May be expensive.
func AllConstraints(pre PrecedenceConstraints) iter.Seq[PrecedenceConstraint] {
	return func(yield func(PrecedenceConstraint) bool) {
		for from := 0; from < pre.NumBlocks(); from++ {
			for to := range pre.Antecedents(from) {
				if !yield(PrecedenceConstraint{From: from, To: to}) {
					return
				}
			}
		}
	}
}

// NumPrecedenceConstraints counts all precedence constraints. May be expensive.
func NumPrecedenceConstraints(pre PrecedenceConstraints) int {
	n := 0
	for range AllConstraints(pre) {
		n++
	}
	return n
}

// BlockValues provides the economic values for each block.
type BlockValues interface {
	NumBlocks() int
	BlockValue(blockIndex int) int64
}

// SliceBlockValues is a simple in-memory implementation of BlockValues
// (VecBlockValues in the C++ code).
type SliceBlockValues []int64

func (v SliceBlockValues) NumBlocks() int { return len(v) }

func (v SliceBlockValues) BlockValue(blockIndex int) int64 {
	if blockIndex < 0 || blockIndex >= len(v) {
		return 0
	}
	return v[blockIndex]
}

// SetBlockValue sets the value of a block.
func (v SliceBlockValues) SetBlockValue(blockIndex int, value int64) {
	v[blockIndex] = value
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

func (p *ExplicitPrecedence) Antecedents(fromBlockIndex int) iter.Seq[int] {
	return func(yield func(int) bool) {
		if fromBlockIndex < 0 || fromBlockIndex >= p.numBlocks {
			return
		}
		for _, to := range p.antecedents[fromBlockIndex] {
			if !yield(to) {
				return
			}
		}
	}
}

// AddConstraint adds a precedence constraint: if "from" is mined, "to" must
// also be mined.
func (p *ExplicitPrecedence) AddConstraint(from, to int) error {
	if from < 0 || from >= p.numBlocks || to < 0 || to >= p.numBlocks {
		return fmt.Errorf("precedence out of range: %d -> %d", from, to)
	}
	p.antecedents[from] = append(p.antecedents[from], to)
	return nil
}

// SolveUltimatePit is a small convenience wrapper that mirrors the README
// example: given block values and (from, to) precedence pairs, return the
// blocks that belong to the maximum-profit closure.
func SolveUltimatePit(values []int64, precedence [][]int64) []bool {
	p := NewExplicitPrecedence(len(values))
	for _, pair := range precedence {
		if len(pair) != 2 {
			continue
		}
		_ = p.AddConstraint(int(pair[0]), int(pair[1]))
	}
	solver, err := NewPseudoSolver(p, SliceBlockValues(values))
	if err != nil {
		return make([]bool, len(values))
	}
	if _, err := solver.Solve(); err != nil {
		return make([]bool, len(values))
	}
	inCut := make([]bool, len(values))
	for i := range inCut {
		inCut[i] = solver.InMinimumCut(i)
	}
	return inCut
}
