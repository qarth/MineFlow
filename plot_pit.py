"""Plot a MineFlow output file (mined block indices) as a 3D pit surface.

Usage:
    python plot_pit.py <output.txt> <nx> <ny> <nz> <sx> <sy> <sz> <image.png>

The input file contains one mined 1D block index per line (x fastest, then y,
then z), and (sx, sy, sz) is the real-world block size used to scale the axes.

Since the mined region is everything above the pit walls, plotting the whole
solid just shows a flat top. Instead this plots the pit shell: for each (x, y)
column, the lowest mined block — i.e. the pit floor/wall surface.
"""

import sys

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np


def main() -> None:
    path, nx, ny, nz = sys.argv[1], *map(int, sys.argv[2:5])
    sx, sy, sz = map(float, sys.argv[5:8])
    image = sys.argv[8]

    indices = np.loadtxt(path, dtype=np.int64)
    x = indices % nx
    y = (indices // nx) % ny
    z = indices // (nx * ny)

    # Pit shell: lowest mined z for each (x, y) column.
    floor = np.full((nx, ny), nz, dtype=np.int64)
    np.minimum.at(floor, (x, y), z)

    mask = floor < nz
    xs, ys = np.nonzero(mask)
    zs = floor[xs, ys]
    print(f"{len(indices)} mined blocks, pit shell spans {len(xs)} columns, "
          f"depth range z = {zs.min()}..{zs.max()} of {nz - 1}")

    fig = plt.figure(figsize=(12, 9))
    ax = fig.add_subplot(projection="3d")
    sc = ax.scatter(xs * sx, ys * sy, zs * sz, s=4, c=zs * sz, cmap="viridis",
                    linewidths=0, marker="s")
    ax.set_box_aspect((nx * sx, ny * sy, 4 * nz * sz))  # exaggerate z for readability
    ax.set_xlabel(f"x ({sx:g} per block)")
    ax.set_ylabel(f"y ({sy:g} per block)")
    ax.set_zlabel(f"z ({sz:g} per block)")
    ax.set_title(f"{path}: ultimate pit, {len(indices)} mined blocks")
    ax.view_init(elev=35, azim=-60)
    fig.colorbar(sc, ax=ax, shrink=0.5, label="pit floor elevation")

    fig.savefig(image, dpi=150, bbox_inches="tight")
    print(f"wrote {image}")


if __name__ == "__main__":
    main()
