package mineflow

// This file ports BlockDefinition from mineflow.h / mineflow.cpp:325-408.
//
// Regular block models are organized such that the first block (1D index 0) is
// the left-most (lowest x), front-most (lowest y), bottom-most (lowest z)
// block. The 1D index increases by x fastest, then y, then z.

// BlockDefinition describes a simple regular block model: the number of
// blocks, its origin, and block size/spacing.
type BlockDefinition struct {
	NumX  int     // Number of blocks in x direction
	NumY  int     // Number of blocks in y direction
	NumZ  int     // Number of blocks in z direction
	MinX  float64 // Origin of blocks x
	MinY  float64 // Origin of blocks y
	MinZ  float64 // Origin of blocks z
	SizeX float64 // Size/spacing of blocks x
	SizeY float64 // Size/spacing of blocks y
	SizeZ float64 // Size/spacing of blocks z
}

// NewBlockDefinition creates a BlockDefinition with explicit counts, origin,
// and block sizes.
func NewBlockDefinition(numX, numY, numZ int,
	minX, minY, minZ, sizeX, sizeY, sizeZ float64) BlockDefinition {
	return BlockDefinition{
		NumX: numX, NumY: numY, NumZ: numZ,
		MinX: minX, MinY: minY, MinZ: minZ,
		SizeX: sizeX, SizeY: sizeY, SizeZ: sizeZ,
	}
}

// UnitModel returns a block model with unit origin and spacing.
func UnitModel(numX, numY, numZ int) BlockDefinition {
	return NewBlockDefinition(numX, numY, numZ, 0, 0, 0, 1, 1, 1)
}

// NumBlocks returns the total number of blocks in the model.
func (b BlockDefinition) NumBlocks() int {
	return b.NumX * b.NumY * b.NumZ
}

// GridIndex computes the 1D grid index from the 3D x, y, z indices.
func (b BlockDefinition) GridIndex(x, y, z int) int {
	return x + y*b.NumX + z*b.NumX*b.NumY
}

// XIndex computes the 3D x index from the 1D grid index.
func (b BlockDefinition) XIndex(idx int) int {
	return idx % b.NumX
}

// YIndex computes the 3D y index from the 1D grid index.
func (b BlockDefinition) YIndex(idx int) int {
	return (idx / b.NumX) % b.NumY
}

// ZIndex computes the 3D z index from the 1D grid index.
func (b BlockDefinition) ZIndex(idx int) int {
	return idx / (b.NumX * b.NumY)
}

// XYZIndices computes the 3D x, y, z indices from the 1D grid index.
func (b BlockDefinition) XYZIndices(idx int) (int, int, int) {
	return b.XIndex(idx), b.YIndex(idx), b.ZIndex(idx)
}

// OffsetIndex computes an offset 1D grid index.
func (b BlockDefinition) OffsetIndex(idx, ox, oy, oz int) int {
	return idx + ox + oy*b.NumX + oz*b.NumX*b.NumY
}

// InDef returns whether the block at the 3D indices would be inside this def.
func (b BlockDefinition) InDef(x, y, z int) bool {
	return x >= 0 && x < b.NumX && y >= 0 && y < b.NumY && z >= 0 && z < b.NumZ
}

// IndexInDef returns whether the block at the 1D index would be inside this def.
func (b BlockDefinition) IndexInDef(idx int) bool {
	return idx >= 0 && idx < b.NumBlocks()
}

// OffsetInDef returns whether the block at the offset from (x, y, z) would be
// inside this def.
func (b BlockDefinition) OffsetInDef(x, y, z, ox, oy, oz int) bool {
	return b.InDef(x+ox, y+oy, z+oz)
}

// IndexOffsetInDef returns whether the block at the offset from the 1D index
// would be inside this def.
func (b BlockDefinition) IndexOffsetInDef(idx, ox, oy, oz int) bool {
	return b.IndexInDef(b.OffsetIndex(idx, ox, oy, oz))
}
