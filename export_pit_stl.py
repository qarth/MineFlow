"""Export the pit shell from a MineFlow output file as a binary STL.

Usage:
    python export_pit_stl.py <output.txt> <nx> <ny> <nz> <sx> <sy> <sz> <slope_deg>

The input file contains one mined 1D block index per line (x fastest, then y,
then z), (sx, sy, sz) is the real-world block size, and slope_deg is the pit
slope parameter the model was run with — it is embedded in the output file
name: <input_base>_pit_<slope>deg.stl

The pit shell is the heightfield of the lowest mined block in each (x, y)
column; every 2x2 group of columns that are all mined yields two triangles.
Triangle normals point upward (into the mined void). Only the floor/wall
surface is exported — there is no lid or side skirt.
"""

import struct
import sys
from pathlib import Path


def main() -> None:
    path = sys.argv[1]
    nx, ny, nz = map(int, sys.argv[2:5])
    sx, sy, sz = map(float, sys.argv[5:8])
    slope = sys.argv[8]

    out_path = Path(path).with_name(f"{Path(path).stem}_pit_{slope}deg.stl")

    # Pit shell: lowest mined z for each (x, y) column.
    floor = [[nz] * ny for _ in range(nx)]
    n_mined = 0
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            idx = int(line)
            x = idx % nx
            y = (idx // nx) % ny
            z = idx // (nx * ny)
            if z < floor[x][y]:
                floor[x][y] = z
            n_mined += 1

    def vert(x: int, y: int) -> tuple[float, float, float]:
        return (x * sx, y * sy, floor[x][y] * sz)

    def normal(a, b, c):
        ux, uy, uz = b[0] - a[0], b[1] - a[1], b[2] - a[2]
        vx, vy, vz = c[0] - a[0], c[1] - a[1], c[2] - a[2]
        nxn = uy * vz - uz * vy
        nyn = uz * vx - ux * vz
        nzn = ux * vy - uy * vx
        length = (nxn * nxn + nyn * nyn + nzn * nzn) ** 0.5
        if length == 0:
            return (0.0, 0.0, 1.0)
        return (nxn / length, nyn / length, nzn / length)

    triangles = []
    for x in range(nx - 1):
        for y in range(ny - 1):
            if floor[x][y] == nz or floor[x + 1][y] == nz:
                continue
            if floor[x][y + 1] == nz or floor[x + 1][y + 1] == nz:
                continue
            a = vert(x, y)
            b = vert(x + 1, y)
            c = vert(x, y + 1)
            d = vert(x + 1, y + 1)
            for tri in ((a, b, d), (a, d, c)):
                n = normal(*tri)
                if n[2] < 0:  # normals point up, into the mined void
                    tri = (tri[0], tri[2], tri[1])
                    n = (-n[0], -n[1], -n[2])
                triangles.append((n, tri))

    with open(out_path, "wb") as f:
        f.write(b"MineFlow pit shell".ljust(80, b"\0"))
        f.write(struct.pack("<I", len(triangles)))
        for n, tri in triangles:
            f.write(struct.pack("<12fH", *n, *tri[0], *tri[1], *tri[2], 0))

    print(f"{n_mined} mined blocks, {len(triangles)} triangles")
    print(f"wrote {out_path}")


if __name__ == "__main__":
    main()
