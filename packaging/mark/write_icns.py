"""Writes the macOS icon from the same model as the SVG, the OBJ and the .ico.

macOS takes an application's icon from an .icns inside its bundle, named by
CFBundleIconFile. A bare Unix binary has no bundle and so no icon at all,
which is what the launcher shipped for macOS until now: Finder and the Dock
drew the generic executable.

The mark is the borderless one, at the same DOCK_FILL proportion as the
Linux icon. macOS draws no ground under an icon either, and the tile was
always ours rather than the desktop's.

An .icns is a container of typed chunks; every type here carries a PNG,
which macOS has read since 10.7. The list covers both the plain sizes and
the @2x ones, because macOS picks by exact type rather than by scaling the
nearest — a set with gaps in it gets filled by resampling, and a resampled
32 px stack is mush.

Nothing here is imported beyond the standard library, for the same reason as
the .ico: a build step that needs a wheel installed first is a build step
that stops working.

    python3 write_icns.py ../instant-launcher.icns
"""
import struct
import sys

from iso_svg import ICON_MARK, faces
from write_ico import hexrgb, png, raster

# type code -> pixel size. Two types share 256 and two share 512: macOS asks
# for "256x256" and "128x128@2x" separately and will resample if only one is
# there, though the pixels are identical.
TYPES = (
    (b"icp4", 16),
    (b"icp5", 32),
    (b"ic11", 32),    # 16x16@2x
    (b"ic12", 64),    # 32x32@2x
    (b"ic07", 128),
    (b"ic13", 256),   # 128x128@2x
    (b"ic08", 256),
    (b"ic14", 512),   # 256x256@2x
    (b"ic09", 512),
    (b"ic10", 1024),  # 512x512@2x
)

# Supersampling, dropped for the large sizes: at 512 and above a 4x buffer is
# millions of cells of pure Python for an edge nobody can see the difference
# in.
def supersample(size):
    return 4 if size <= 256 else 2


def icns(chunks):
    """Pack (type, payload) into an .icns.

    Each chunk's length counts its own eight-byte header, and so does the
    file's; getting that wrong produces a file macOS reads as empty rather
    than as broken, which is a long afternoon.
    """
    body = b"".join(t + struct.pack(">I", len(d) + 8) + d for t, d in chunks)
    return b"icns" + struct.pack(">I", len(body) + 8) + body


if __name__ == "__main__":
    out = sys.argv[1] if len(sys.argv) > 1 else "../instant-launcher.icns"
    polys = [(hexrgb(col), pts) for col, pts in faces(30, 30, ICON_MARK)]

    drawn = {}
    chunks = []
    for code, size in TYPES:
        if size not in drawn:
            drawn[size] = png(raster(polys, size, supersample(size)), size)
            print("  drew", size)
        chunks.append((code, drawn[size]))

    data = icns(chunks)
    open(out, "wb").write(data)
    print("wrote", out, len(data), "bytes,", len(chunks), "entries,",
          len(drawn), "sizes")
