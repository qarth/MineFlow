package mineflow

import "testing"

// blockdef_test.go — port of the C++ TEST(Block, *) suite
// (mineflow.cpp:3576-3678).

func TestBlock1DIndices(t *testing.T) {
	def := UnitModel(10, 8, 5)

	type tc struct {
		idx, x, y, z int
	}
	for _, c := range []tc{
		{0, 0, 0, 0},
		{1, 1, 0, 0},
		{2, 2, 0, 0},
		{10, 0, 1, 0},
		{79, 9, 7, 0},
		{80, 0, 0, 1},
		{81, 1, 0, 1},
		{173, 3, 1, 2},
	} {
		if got := def.XIndex(c.idx); got != c.x {
			t.Fatalf("XIndex(%d): got %d want %d", c.idx, got, c.x)
		}
		if got := def.YIndex(c.idx); got != c.y {
			t.Fatalf("YIndex(%d): got %d want %d", c.idx, got, c.y)
		}
		if got := def.ZIndex(c.idx); got != c.z {
			t.Fatalf("ZIndex(%d): got %d want %d", c.idx, got, c.z)
		}
	}
}

func TestBlock3DIndices(t *testing.T) {
	nx, ny, nz := 10, 8, 5
	def := UnitModel(nx, ny, nz)

	k := 0
	for z := 0; z < nz; z++ {
		for y := 0; y < ny; y++ {
			for x := 0; x < nx; x++ {
				if def.XIndex(k) != x || def.YIndex(k) != y || def.ZIndex(k) != z {
					t.Fatalf("index %d does not map to (%d, %d, %d)", k, x, y, z)
				}
				if got := def.GridIndex(x, y, z); got != k {
					t.Fatalf("GridIndex(%d, %d, %d): got %d want %d", x, y, z, got, k)
				}
				k++
			}
		}
	}
}

func TestBlockNumBlocks(t *testing.T) {
	if got := UnitModel(10, 8, 5).NumBlocks(); got != 400 {
		t.Fatalf("got %d want 400", got)
	}
	if got := UnitModel(10, 10, 1).NumBlocks(); got != 100 {
		t.Fatalf("got %d want 100", got)
	}
	if got := UnitModel(32, 50, 20).NumBlocks(); got != 32000 {
		t.Fatalf("got %d want 32000", got)
	}
}

func TestBlockOffsetIndex(t *testing.T) {
	def := UnitModel(10, 8, 5)

	if got := def.OffsetIndex(0, 5, 0, 0); got != 5 {
		t.Fatalf("got %d want 5", got)
	}
	if got := def.OffsetIndex(2, 3, 0, 0); got != 5 {
		t.Fatalf("got %d want 5", got)
	}
	if got := def.OffsetIndex(5, 0, 1, 0); got != 15 {
		t.Fatalf("got %d want 15", got)
	}
	if got := def.OffsetIndex(15, 0, 0, 1); got != 95 {
		t.Fatalf("got %d want 95", got)
	}
}

func TestBlockInDef(t *testing.T) {
	nx, ny, nz := 10, 8, 5
	def := UnitModel(nx, ny, nz)

	if def.InDef(-1, 0, 0) || def.InDef(0, -1, 0) || def.InDef(0, 0, -1) {
		t.Fatal("negative indices should be out of the definition")
	}
	if def.IndexInDef(-1) || def.IndexInDef(nx*ny*nz) {
		t.Fatal("out-of-range 1D indices should be out of the definition")
	}

	k := 0
	for z := 0; z < nz; z++ {
		for y := 0; y < ny; y++ {
			for x := 0; x < nx; x++ {
				if !def.InDef(x, y, z) || !def.IndexInDef(k) {
					t.Fatalf("(%d, %d, %d) / %d should be in the definition", x, y, z, k)
				}
				k++
			}
		}
	}
}
