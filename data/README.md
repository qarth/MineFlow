Data is given as just raw, integerized / anonymized, economic block values.
These are regular block models.
Blocks are in order from the leftmost (smallest x), frontmost (smallest y), lowest (smallest z), to the rightmost (largest x), backmost (largest y), highest (largest z).
x cycles fastest, then y, then z.

bauxitemed: 120x120x26
cucase: 170x215x50
cupipe: 180x180x85
mclaughlingeo: 140x296x68
sim2d76: 75x1x40

``` Go
Note how the IJK block index is used to shape the index to a grid of number x blocks, num y blocks, num z block which form the grid bounds.
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
```

wget --mirror --convert-links --adjust-extension --page-requisites --no-parent https://minelib.org/v1/Datasets.xhtml