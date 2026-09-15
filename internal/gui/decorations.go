package gui

import (
	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// The title bar is ours on every platform, and it has to be: left alone, the
// window wears whatever the desktop feels like giving it. On GNOME under
// Wayland the desktop gives it nothing, Gio steps in with a bar of its own,
// and it paints that bar with Gio's stock theme — Material indigo, which
// belongs to no part of this program. On Windows it is the system's own bar,
// white. Two machines, two strangers above the same window. Asking for the
// launcher's green in both places is asking to draw the bar.
//
// What we give up is small and Gio hands most of it back. Undecorated, its
// Windows driver answers WM_NCHITTEST itself: the edges still resize, the
// bar still reports as the caption, so dragging, snapping and double-click
// to maximise stay native, and the window even keeps its drop shadow. Under
// Wayland the pointer picks up an edge exactly as it did before — that path
// was only ever active for undecorated windows, which is what we already
// were there.
//
// A word of warning to anyone tempted to make this conditional on whether
// the system decorates: the window will not tell you. app.Config.Decorated
// reads true whenever *something* decorates the window, and Gio's own
// fallback bar counts, so the flag is true in every case that matters and
// the condition is dead code.
type decorations struct {
	state widget.Decorations
}

// configure follows the window's config, which is where the maximize button
// learns which of its two shapes to draw; widget.Decorations leaves that to
// the caller.
func (d *decorations) configure(cnf app.Config) {
	d.state.Maximized = cnf.Mode == app.Maximized
}

// decoActions is what the bar offers: drag it to move the window, and the
// three buttons on the right. Resizing is not in the list because it is not
// the bar's job — the pointer takes a window edge in the driver.
const decoActions = system.ActionMinimize | system.ActionMaximize |
	system.ActionUnmaximize | system.ActionClose | system.ActionMove

// layout draws the title bar above the window's content.
func (d *decorations) layout(gtx layout.Context, th *Theme, w *app.Window, title string, content layout.Widget) layout.Dimensions {
	// Only when a button was actually hit: Perform reaches into the window,
	// and a frame that decorates nothing should not touch it at all. It also
	// lets the offscreen renderer lay the bar out with no window behind it.
	if actions := d.state.Update(gtx); actions != 0 {
		w.Perform(actions)
	}
	style := material.Decorations(th.decoTheme(), &d.state, decoActions, title)
	// The title bar is where the wordmark lives now that the top bar's
	// button says Home, so it is set like a wordmark: the display face,
	// bold, rather than Gio's plain body default.
	style.Title.Font.Typeface = faceDisplay
	style.Title.Font.Weight = font.Bold
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(style.Layout),
		layout.Flexed(1, content),
	)
}

// decoTheme is the theme material.Decorations reads: it takes the bar's
// ground from ContrastBg and its title and buttons from ContrastFg. Ours
// reserves that pair for the Play button, so the title bar gets a copy with
// the brand green in their place.
func (t *Theme) decoTheme() *material.Theme {
	deco := *t.Theme
	deco.Palette.ContrastBg = t.P.Grass
	deco.Palette.ContrastFg = t.P.GrassInk
	return &deco
}
