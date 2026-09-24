package gui

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// drawCtx is a context good enough to measure a widget: the drawing helpers
// only need somewhere to put their operations and a metric to turn dp into
// pixels.
func drawCtx(ops *op.Ops, dpi float32) layout.Context {
	return layout.Context{
		Ops:         ops,
		Constraints: layout.Constraints{Max: image.Pt(1180, 760)},
		Metric:      unit.Metric{PxPerDp: dpi, PxPerSp: dpi},
	}
}

// TestMarkGlyphMatchesSlab is here because the Home button's mark once came
// out 20x22 where every instance in the rail below it was 22x24 — small
// enough to pass a reading of the code and large enough to see in the
// window. The two are drawn by different functions from different geometry,
// so nothing but a measurement keeps them the same size.
func TestMarkGlyphMatchesSlab(t *testing.T) {
	for _, dpi := range []float32{1, 1.25, 2} {
		var ops op.Ops
		gtx := drawCtx(&ops, dpi)

		glyph := markGlyph(gtx, railSlabSize)
		stack := slab(gtx, rgb(0x7DC057), railSlabSize)

		if glyph.Size.X != stack.Size.X {
			t.Errorf("at %gx: markGlyph is %d wide, slab is %d; the Home mark and the rail's must match",
				dpi, glyph.Size.X, stack.Size.X)
		}
		// Not equality: the block is a hair taller than a stack of slabs, so
		// at a fractional scale the two land on either side of a pixel
		// boundary. One pixel is the honest tolerance; two is the bug.
		if d := glyph.Size.Y - stack.Size.Y; d < -1 || d > 1 {
			t.Errorf("at %gx: markGlyph is %d tall, slab is %d; more than a pixel apart",
				dpi, glyph.Size.Y, stack.Size.Y)
		}
	}
}

// TestMarkGlyphHugsTheModel checks that the glyph carries none of the icon
// tile's padding: at 1x its box is the model's own, so it sits against the
// word beside it rather than a gap's width away.
func TestMarkGlyphHugsTheModel(t *testing.T) {
	var ops op.Ops
	gtx := drawCtx(&ops, 1)

	got := markGlyph(gtx, unit.Dp(372)).Size
	// The model spans 334.4 x 372 of the 512-unit tile, so a glyph 372 wide
	// is 372 x 414 — the tile's aspect would have made it square.
	if want := image.Pt(372, 413); got != want {
		t.Errorf("markGlyph box = %v, want %v (the model's proportions, not the tile's)", got, want)
	}
}
