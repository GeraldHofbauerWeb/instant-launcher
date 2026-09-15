"""The Instant Launcher mark as geometry, shared by the SVG generator, the
OBJ writer and the Blender script.

Coordinates are Blender's (X right-front, Y depth, Z up), in centimetres.
The mark is a grass block cut into three instance slabs: a 5x5x5 cube of
one-unit slabs and gaps, the topmost one capped with a thin skin of grass.
Every part is a prism: a flat polygon extruded along one axis, so its faces
meet edge to edge and never overlap on screen.

The colours are ours, not Mojang's. Minecraft's own textures are game files
and may not be redistributed, so the block is drawn in its palette rather
than lifted from it.
"""

# The tile the mark sits on. The lower slabs are mixed towards it so the
# stack recedes into its own ground rather than sitting on it as three
# equally lit plates.
TILE = "#1E2128"
DIRT = "#A3714A"


def blend(a, b, t):
    """a mixed t of the way towards b, both "#rrggbb"."""
    ca = [int(a[i:i + 2], 16) for i in (1, 3, 5)]
    cb = [int(b[i:i + 2], 16) for i in (1, 3, 5)]
    return "#%02X%02X%02X" % tuple(int(x + (y - x) * t) for x, y in zip(ca, cb))


# Three steps down, the two factors the rail's slab() dims by. The icon does
# not land on the rail's steps of 2.6x and 2.2x, though: iso_svg.py lights
# every face and adds ambient on top, which compresses the difference to an
# even 1.6x a step. Measured on the generated faces, not assumed.
BASE = {
    "dirt":     DIRT,
    "dirt_mid": blend(DIRT, TILE, 0.30),
    "dirt_low": blend(DIRT, TILE, 0.55),
    "grass":    "#79BD4C",
}


def prism(poly, axis, a0, a1):
    """Faces of a polygon extruded from a0 to a1 along `axis` (0, 1 or 2).

    The polygon's two coordinates are the other two axes in ascending order.
    Returns (main, sides, back): the face at a1 (normal +axis), the side
    faces (outward), and the face at a0 (normal -axis). Each face is a list
    of (x, y, z) tuples wound counter-clockwise seen from outside.
    """
    n = len(poly)
    area = sum(poly[i][0] * poly[(i + 1) % n][1] - poly[(i + 1) % n][0] * poly[i][1] for i in range(n))
    if area < 0:
        poly = poly[::-1]
    others = [i for i in range(3) if i != axis]

    def p3(u, v, a):
        pt = [0.0, 0.0, 0.0]
        pt[others[0]], pt[others[1]], pt[axis] = u, v, a
        return tuple(pt)

    # CCW in (u, v) is outward for the +axis face when (u, v, axis) is a
    # right-handed order, which holds for axis 2 (x,y,z) and axis 0 (y,z,x)
    # but not for axis 1 (x,z,y), so that one flips.
    flip = axis == 1
    main = [p3(u, v, a1) for u, v in poly]
    back = [p3(u, v, a0) for u, v in poly]
    if flip:
        main, back = main[::-1], back[::-1]
    sides = []
    for i in range(n):
        j = (i + 1) % n
        a, b = poly[i], poly[j]
        quad = [p3(*a, a0), p3(*b, a0), p3(*b, a1), p3(*a, a1)]
        sides.append(quad[::-1] if flip else quad)
    return main, sides, back[::-1]


def build(slab=30.0, gap=30.0):
    """Returns the parts in drawing order: (name, material, main, sides, back)."""
    pitch, side = slab + gap, 2 * (slab + gap) + slab
    parts = []
    square = [(0, 0), (150, 0), (150, 150), (0, 150)]
    # The grass grows on the stack, not inside it, so it is its own prism
    # skimmed off the top slab: a green top face and a green rim around the
    # sides, the way a grass block reads from the side in game. The stack
    # keeps its full height, so the silhouette stays a 5x5x5 cube.
    cap = slab / 4
    for name, mat, z0 in (("slab_bottom", "dirt_low", 0),
                          ("slab_middle", "dirt_mid", pitch),
                          ("slab_top", "dirt", 2 * pitch)):
        z1 = z0 + slab - (cap if name == "slab_top" else 0)
        parts.append((name, mat) + prism(square, 2, z0, z1))
    parts.append(("grass_cap", "grass") + prism(square, 2, side - cap, side))
    return parts


def normal(pts):
    nx = ny = nz = 0.0
    for i in range(len(pts)):
        (x0, y0, z0), (x1, y1, z1) = pts[i], pts[(i + 1) % len(pts)]
        nx += (y0 - y1) * (z0 + z1)
        ny += (z0 - z1) * (x0 + x1)
        nz += (x0 - x1) * (y0 + y1)
    l = (nx * nx + ny * ny + nz * nz) ** 0.5 or 1.0
    return nx / l, ny / l, nz / l
