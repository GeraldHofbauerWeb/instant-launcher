package gui

import (
	"fmt"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/GeraldHofbauerWeb/instant-launcher/internal/instance"
	"github.com/GeraldHofbauerWeb/instant-launcher/internal/launch"
	"github.com/GeraldHofbauerWeb/instant-launcher/internal/launcher"
)

// dialogs holds the modal states. At most one is open at a time.
type dialogs struct {
	scrim widget.Clickable

	create createDialog
	del    deleteDialog
	clean  cleanupDialog
	pick   picker
}

func newDialogs() dialogs {
	return dialogs{create: newCreateDialog(), pick: newPicker(), clean: newCleanupDialog()}
}

func (d *dialogs) Layout(gtx layout.Context, u *ui, snap launcher.Snapshot) layout.Dimensions {
	switch {
	case d.pick.open:
		// The picker sits above whatever opened it; that dialog keeps its
		// state and returns when the choice is made.
		if d.scrim.Clicked(gtx) {
			d.pick.open = false
		}
		return u.th.modal(gtx, &d.scrim, unit.Dp(460), func(gtx layout.Context) layout.Dimensions {
			return d.pick.Layout(gtx, u, snap)
		})
	case d.create.open:
		if d.scrim.Clicked(gtx) {
			d.create.open = false
		}
		return u.th.modal(gtx, &d.scrim, unit.Dp(540), func(gtx layout.Context) layout.Dimensions {
			return d.create.Layout(gtx, u, snap)
		})
	case d.del.open:
		if d.scrim.Clicked(gtx) {
			d.del.open = false
		}
		return u.th.modal(gtx, &d.scrim, unit.Dp(460), func(gtx layout.Context) layout.Dimensions {
			return d.del.Layout(gtx, u)
		})
	case d.clean.open:
		if d.scrim.Clicked(gtx) {
			d.clean.open = false
		}
		return u.th.modal(gtx, &d.scrim, unit.Dp(600), func(gtx layout.Context) layout.Dimensions {
			return d.clean.Layout(gtx, u, snap)
		})
	}
	return layout.Dimensions{}
}

func (d *dialogs) openCreate(snap launcher.Snapshot, cloneFrom string) {
	d.create.show(snap, cloneFrom)
}

func (d *dialogs) openDelete(inst instance.Instance) {
	d.del.inst = inst
	d.del.open = true
}

// openCleanup asks about one instance, or about all of them when name is
// empty.
func (d *dialogs) openCleanup(name string) { d.clean.show(name) }

// pickMinecraft opens the release list.
func (d *dialogs) pickMinecraft(u *ui, current string, onPick func(string)) {
	d.pick.show(u, "Minecraft version", pickMinecraft, "", "", current, true, onPick)
}

// pickLoaderVersion opens one loader's list for a game version.
func (d *dialogs) pickLoaderVersion(u *ui, kind instance.LoaderType, mc, current string, onPick func(string)) {
	d.pick.show(u, kind.Display()+" for "+mc, pickLoader, kind, mc, current, true, onPick)
}

// pickInstance opens the instance list.
func (d *dialogs) pickInstance(u *ui, current string, onPick func(string)) {
	d.pick.show(u, "Copy which instance?", pickInstances, "", "", current, false, onPick)
}

// --- new instance ---

// createDialog makes an instance. Empty is the default because cloning
// costs gigabytes and the shared store already supplies everything a fresh
// instance needs; duplicating an existing one is a click away.
type createDialog struct {
	open bool

	name *widget.Editor

	version       string
	pickVersion   widget.Clickable
	loaderChoices []widget.Clickable
	loaderVersion string
	pickLoaderVer widget.Clickable
	loader        instance.LoaderType

	fromScratch, fromClone, fromMinecraft widget.Clickable
	clone                                 bool
	cloneSource                           string
	pickSource                            widget.Clickable

	// minecraft makes the dialog an import of the official launcher's
	// .minecraft: the version is read from it rather than chosen, and the
	// worlds and screenshots can come along.
	minecraft                bool
	withSaves, withShots     bool
	toggleSaves, toggleShots widget.Clickable

	confirm, cancel widget.Clickable
}

func newCreateDialog() createDialog {
	return createDialog{
		name:   newEditor(),
		loader: instance.LoaderVanilla,
	}
}

// show opens the dialog. With a clone source it starts as a duplicate of
// that instance, taking its version and loader along.
func (d *createDialog) show(snap launcher.Snapshot, cloneFrom string) {
	d.open = true
	d.name.SetText("")
	d.loader = instance.LoaderVanilla
	d.loaderVersion = ""
	d.version = ""
	d.clone = cloneFrom != ""
	d.cloneSource = cloneFrom
	d.minecraft = false
	d.withSaves, d.withShots = true, true

	if snap.Editing.MinecraftVersion != "" {
		d.version = snap.Editing.MinecraftVersion
		if cloneFrom != "" {
			d.loader = snap.Editing.Loader.Type
			d.loaderVersion = snap.Editing.Loader.Version
			d.name.SetText(cloneFrom + "-copy")
		}
	}
}

func (d *createDialog) Layout(gtx layout.Context, u *ui, snap launcher.Snapshot) layout.Dimensions {
	th := u.th
	loaders := instance.LoaderTypes()
	for len(d.loaderChoices) < len(loaders) {
		d.loaderChoices = append(d.loaderChoices, widget.Clickable{})
	}

	for i, lt := range loaders {
		if d.loaderChoices[i].Clicked(gtx) && d.loader != lt {
			d.loader = lt
			// A version belongs to one loader; the next one starts fresh.
			d.loaderVersion = ""
		}
	}
	if d.pickVersion.Clicked(gtx) {
		u.dialogs.pickMinecraft(u, d.version, func(v string) {
			if v != d.version {
				d.loaderVersion = ""
			}
			d.version = v
		})
	}
	if d.pickLoaderVer.Clicked(gtx) {
		u.dialogs.pickLoaderVersion(u, d.loader, d.version, d.loaderVersion, func(v string) { d.loaderVersion = v })
	}
	if d.fromScratch.Clicked(gtx) {
		d.clone, d.minecraft = false, false
	}
	if d.fromClone.Clicked(gtx) {
		d.clone, d.minecraft = true, false
	}
	if d.fromMinecraft.Clicked(gtx) {
		d.clone, d.minecraft = false, true
		if strings.TrimSpace(d.name.Text()) == "" {
			d.name.SetText(instance.DefaultInstanceName)
		}
	}
	if d.toggleSaves.Clicked(gtx) {
		d.withSaves = !d.withSaves
	}
	if d.toggleShots.Clicked(gtx) {
		d.withShots = !d.withShots
	}
	if d.pickSource.Clicked(gtx) {
		u.dialogs.pickInstance(u, d.cloneSource, func(v string) { d.cloneSource = v })
	}
	if d.cancel.Clicked(gtx) {
		d.open = false
	}

	named := strings.TrimSpace(d.name.Text()) != ""
	ready := named && d.version != "" &&
		(d.loader == instance.LoaderVanilla || d.loaderVersion != "") &&
		(!d.clone || d.cloneSource != "")
	if d.minecraft {
		ready = named
	}
	if ready && d.minecraft && d.confirm.Clicked(gtx) {
		u.ctrl.Dispatch(launcher.ActionImport{
			Name:               strings.TrimSpace(d.name.Text()),
			IncludeSaves:       d.withSaves,
			IncludeScreenshots: d.withShots,
		})
		d.open = false
	}
	if ready && !d.minecraft && d.confirm.Clicked(gtx) {
		action := launcher.ActionCreate{
			Name:    strings.TrimSpace(d.name.Text()),
			Version: d.version,
			Loader:  instance.LoaderSpec{Type: d.loader, Version: d.loaderVersion},
		}
		if d.clone {
			action.Clone = d.cloneSource
		}
		u.ctrl.Dispatch(action)
		d.open = false
	}

	children := []layout.FlexChild{
		rigid(func(gtx layout.Context) layout.Dimensions {
			switch {
			case d.minecraft:
				return th.display(gtx, "Import your "+minecraftDir(snap))
			case d.clone && d.cloneSource != "":
				return th.display(gtx, "Duplicate "+d.cloneSource)
			}
			return th.display(gtx, "New instance")
		}),
		rigid(func(gtx layout.Context) layout.Dimensions { return th.field(gtx, d.name, "Name", "my-modpack") }),
	}
	// An import reads the version and loader from the official launcher's
	// profile, so there is nothing to choose.
	if !d.minecraft {
		children = append(children, d.versionChildren(u, loaders)...)
	}
	children = append(children,
		rigid(func(gtx layout.Context) layout.Dimensions { return d.layoutSource(gtx, u, snap) }),
		rigid(func(gtx layout.Context) layout.Dimensions {
			label := "Create"
			if d.minecraft {
				label = "Import"
			}
			return row(gtx, sp2,
				rigid(func(gtx layout.Context) layout.Dimensions {
					if !ready {
						return th.secondary(gtx, &d.confirm, label)
					}
					return th.primary(gtx, &d.confirm, nil, label)
				}),
				rigid(func(gtx layout.Context) layout.Dimensions { return th.ghost(gtx, &d.cancel, nil, "Cancel") }),
			)
		}),
	)
	return column(gtx, sp3, children...)
}

// versionChildren are the loader, version and install note rows of a new
// instance.
func (d *createDialog) versionChildren(u *ui, loaders []instance.LoaderType) []layout.FlexChild {
	th := u.th
	return []layout.FlexChild{
		rigid(func(gtx layout.Context) layout.Dimensions {
			return column(gtx, unit.Dp(6),
				rigid(func(gtx layout.Context) layout.Dimensions { return th.small(gtx, "Mod loader") }),
				rigid(func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(loaders))
					for i, lt := range loaders {
						i, lt := i, lt
						children = append(children, rigid(func(gtx layout.Context) layout.Dimensions {
							return th.pill(gtx, &d.loaderChoices[i], lt == d.loader, lt.Display())
						}))
					}
					return row(gtx, unit.Dp(6), children...)
				}),
			)
		}),
		rigid(func(gtx layout.Context) layout.Dimensions {
			return row(gtx, sp3,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return th.selectField(gtx, u, &d.pickVersion, "Minecraft version", d.version, "Choose a release")
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if d.loader == instance.LoaderVanilla {
						return layout.Dimensions{}
					}
					placeholder := "Choose a version"
					if d.version == "" {
						placeholder = "Minecraft version first"
					}
					return th.selectField(gtx, u, &d.pickLoaderVer, d.loader.Display()+" version", d.loaderVersion, placeholder)
				}),
			)
		}),
		rigid(func(gtx layout.Context) layout.Dimensions {
			if d.loader == instance.LoaderVanilla {
				return layout.Dimensions{}
			}
			return th.wrapped(gtx, d.loader.Display()+" is installed into the shared store when the instance is created.", th.P.TextDim)
		}),
	}
}

func (d *createDialog) layoutSource(gtx layout.Context, u *ui, snap launcher.Snapshot) layout.Dimensions {
	th := u.th
	children := []layout.FlexChild{
		rigid(func(gtx layout.Context) layout.Dimensions { return th.small(gtx, "Start from") }),
		rigid(func(gtx layout.Context) layout.Dimensions {
			return row(gtx, unit.Dp(6),
				rigid(func(gtx layout.Context) layout.Dimensions {
					return th.pill(gtx, &d.fromScratch, !d.clone && !d.minecraft, "Empty")
				}),
				rigid(func(gtx layout.Context) layout.Dimensions {
					return th.pill(gtx, &d.fromClone, d.clone, "A copy of an instance")
				}),
				rigid(func(gtx layout.Context) layout.Dimensions {
					if !snap.CanImport && !d.minecraft {
						return layout.Dimensions{}
					}
					return th.pill(gtx, &d.fromMinecraft, d.minecraft, "Your "+minecraftDir(snap))
				}),
			)
		}),
	}

	if d.minecraft {
		children = append(children,
			rigid(func(gtx layout.Context) layout.Dimensions {
				return row(gtx, unit.Dp(6),
					rigid(func(gtx layout.Context) layout.Dimensions { return th.smallIn(gtx, "Also copy", th.P.TextDim) }),
					rigid(func(gtx layout.Context) layout.Dimensions { return th.pill(gtx, &d.toggleSaves, d.withSaves, "Worlds") }),
					rigid(func(gtx layout.Context) layout.Dimensions {
						return th.pill(gtx, &d.toggleShots, d.withShots, "Screenshots")
					}),
				)
			}),
			rigid(func(gtx layout.Context) layout.Dimensions {
				return th.wrapped(gtx, "Mods, configs, packs and the game options are copied, the game files go into "+
					"the shared store, and the version is read from the official launcher. "+
					minecraftDir(snap)+" itself stays exactly as it is.", th.P.TextDim)
			}),
		)
		return column(gtx, unit.Dp(6), children...)
	}

	if !d.clone {
		children = append(children, rigid(func(gtx layout.Context) layout.Dimensions {
			return th.wrapped(gtx, "Ready at once. The game files come from the shared store.", th.P.TextDim)
		}))
		return column(gtx, unit.Dp(6), children...)
	}

	children = append(children,
		rigid(func(gtx layout.Context) layout.Dimensions {
			return th.selectField(gtx, u, &d.pickSource, "Instance to copy", d.cloneSource, "Choose an instance")
		}),
		rigid(func(gtx layout.Context) layout.Dimensions {
			return th.wrapped(gtx, "Mods, configs and packs are copied. Worlds and screenshots are not.", th.P.TextDim)
		}),
	)
	return column(gtx, unit.Dp(6), children...)
}

// --- delete instance ---

// deleteDialog asks before an instance and everything in it goes.
type deleteDialog struct {
	open            bool
	inst            instance.Instance
	confirm, cancel widget.Clickable
}

func (d *deleteDialog) Layout(gtx layout.Context, u *ui) layout.Dimensions {
	th := u.th
	if d.cancel.Clicked(gtx) {
		d.open = false
	}
	if d.confirm.Clicked(gtx) {
		d.open = false
		u.ctrl.Dispatch(launcher.ActionDelete{Name: d.inst.Name})
	}

	return column(gtx, sp3,
		rigid(func(gtx layout.Context) layout.Dimensions { return th.display(gtx, "Delete "+d.inst.Name+"?") }),
		rigid(func(gtx layout.Context) layout.Dimensions {
			return th.wrapped(gtx, "Its mods, configs, worlds and screenshots are removed from disk. "+
				"There is no undo. The shared game files stay.", th.P.TextMid)
		}),
		rigid(func(gtx layout.Context) layout.Dimensions {
			return th.wrapped(gtx, d.inst.Path, th.P.TextDim)
		}),
		rigid(func(gtx layout.Context) layout.Dimensions {
			return row(gtx, sp2,
				rigid(func(gtx layout.Context) layout.Dimensions {
					return th.layoutButton(gtx, &d.confirm, buttonStyle{
						bg: th.P.Bad, hoverBg: mix(th.P.Bad, rgb(0xFFFFFF), 0.1), fg: rgb(0xFFFFFF),
						size: sizeBody, inset: layout.Inset{Top: sp2, Bottom: sp2, Left: sp3, Right: sp3},
					}, th.buttonLabel(buttonStyle{fg: rgb(0xFFFFFF), size: sizeBody}, "Delete "+d.inst.Name))
				}),
				rigid(func(gtx layout.Context) layout.Dimensions { return th.ghost(gtx, &d.cancel, nil, "Keep it") }),
			)
		}),
	)
}

// --- clean up ---

// cleanupDialog asks before it deletes, and asks with a number. It measures
// what a cleanup would take, shows it heap by heap, and only then offers the
// button.
//
// Nothing it can remove is irreplaceable — logs and crash reports are
// written again the next time the game runs — so the confirming button is an
// ordinary one rather than the red of the delete dialog. What makes the
// question worth asking at all is the size: clearing a year of logs is worth
// seeing coming, and so is a cleanup that would free nothing.
type cleanupDialog struct {
	open bool
	// name is the instance, or empty for every instance and the launcher's
	// own logs.
	name string
	// age indexes instance.CleanupAges.
	age  int
	ages []widget.Clickable

	confirm, cancel widget.Clickable

	// asked is the (instance, age) a plan has been requested for, so the
	// dialog asks once per change and not once per frame.
	asked    string
	askedAge time.Duration
	hasAsked bool
}

func newCleanupDialog() cleanupDialog {
	return cleanupDialog{ages: make([]widget.Clickable, len(instance.CleanupAges()))}
}

// olderThan is the cutoff the pills currently select.
func (d *cleanupDialog) olderThan() time.Duration {
	ages := instance.CleanupAges()
	if d.age < 0 || d.age >= len(ages) {
		return 0
	}
	return ages[d.age]
}

func (d *cleanupDialog) show(name string) {
	d.open = true
	d.name = name
	d.age = 0
	d.hasAsked = false
}

// title names what is about to be cleaned.
func (d *cleanupDialog) title() string {
	if d.name == "" {
		return "Clean up every instance?"
	}
	return "Clean up " + d.name + "?"
}

func (d *cleanupDialog) Layout(gtx layout.Context, u *ui, snap launcher.Snapshot) layout.Dimensions {
	th := u.th
	age := d.olderThan()

	// Ask for a count when the dialog opens and whenever the age changes,
	// exactly once each: a dispatch per frame would queue hundreds of walks
	// over the same directories before the first one answered.
	if !d.hasAsked || d.asked != d.name || d.askedAge != age {
		d.asked, d.askedAge, d.hasAsked = d.name, age, true
		u.dispatch(launcher.ActionPlanCleanup{Name: d.name, OlderThan: age})
	}

	for i := range d.ages {
		if d.ages[i].Clicked(gtx) {
			d.age = i
		}
	}
	if d.cancel.Clicked(gtx) {
		d.open = false
	}

	plan := snap.Cleanup
	ready := plan.For(d.name, age)
	if d.confirm.Clicked(gtx) && ready && !plan.Plan.Empty() {
		d.open = false
		u.dispatch(launcher.ActionCleanup{Name: d.name, OlderThan: age})
	}

	children := []layout.FlexChild{
		rigid(func(gtx layout.Context) layout.Dimensions { return th.display(gtx, d.title()) }),
		rigid(func(gtx layout.Context) layout.Dimensions {
			what := "Logs and crash reports."
			if d.name == "" {
				what = "Logs and crash reports in every instance, and the launcher's own start-up logs."
			}
			return th.wrapped(gtx, what+" The game writes them again as it runs. "+
				"Worlds, screenshots, mods and configuration are never touched.", th.P.TextMid)
		}),
		rigid(func(gtx layout.Context) layout.Dimensions {
			kids := make([]layout.FlexChild, 0, len(d.ages))
			for i, a := range instance.CleanupAges() {
				i := i
				kids = append(kids, rigid(func(gtx layout.Context) layout.Dimensions {
					return th.pill(gtx, &d.ages[i], i == d.age, instance.CleanupAgeLabel(a))
				}))
			}
			return row(gtx, sp2, kids...)
		}),
		rigid(func(gtx layout.Context) layout.Dimensions {
			return d.layoutPlan(gtx, u, snap, ready)
		}),
	}

	children = append(children, rigid(func(gtx layout.Context) layout.Dimensions {
		return row(gtx, sp2,
			rigid(func(gtx layout.Context) layout.Dimensions {
				label := "Clean up"
				if ready && !plan.Plan.Empty() {
					label = "Clean up " + launch.FormatBytes(plan.Plan.Bytes)
				}
				if !ready || plan.Plan.Empty() {
					// Measured and empty, or still counting: there is
					// nothing to confirm yet.
					return th.secondary(gtx, &d.confirm, label)
				}
				return th.primary(gtx, &d.confirm, u.ic.Delete, label)
			}),
			rigid(func(gtx layout.Context) layout.Dimensions {
				return th.ghost(gtx, &d.cancel, nil, "Cancel")
			}),
		)
	}))

	return column(gtx, sp3, children...)
}

// layoutPlan shows what was counted: the total, then the heaps behind it.
func (d *cleanupDialog) layoutPlan(gtx layout.Context, u *ui, snap launcher.Snapshot, ready bool) layout.Dimensions {
	th := u.th
	if !ready {
		return th.smallIn(gtx, "Counting…", th.P.TextDim)
	}

	plan := snap.Cleanup.Plan
	if plan.Empty() {
		msg := "Nothing to clean up."
		if plan.Kept > 0 {
			// Not the same thing as an empty instance, and saying so saves
			// the player wondering whether the button works.
			msg = fmt.Sprintf("Nothing that old. %d files are newer than the cutoff.", plan.Kept)
		}
		return th.smallIn(gtx, msg, th.P.TextDim)
	}

	kids := []layout.FlexChild{
		rigid(func(gtx layout.Context) layout.Dimensions {
			line := fmt.Sprintf("%d files · %s", plan.Files, launch.FormatBytes(plan.Bytes))
			if plan.Kept > 0 {
				line += fmt.Sprintf(" · %d newer files stay", plan.Kept)
			}
			return th.bodyMedium(gtx, line)
		}),
	}
	// The heaps, biggest first, and only the first few: a global cleanup
	// over a dozen instances would otherwise be a wall of two-line entries.
	const shown = 6
	for i, g := range plan.Groups {
		if i >= shown {
			kids = append(kids, rigid(func(gtx layout.Context) layout.Dimensions {
				return th.smallIn(gtx, fmt.Sprintf("and %d more", len(plan.Groups)-shown), th.P.TextDim)
			}))
			break
		}
		g := g
		kids = append(kids, rigid(func(gtx layout.Context) layout.Dimensions {
			return row(gtx, sp2,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return th.smallIn(gtx, g.Label(), th.P.TextMid)
				}),
				rigid(func(gtx layout.Context) layout.Dimensions {
					return th.monoIn(gtx, fmt.Sprintf("%d · %s", g.Files, launch.FormatBytes(g.Bytes)), th.P.TextDim)
				}),
			)
		}))
	}
	return column(gtx, unit.Dp(4), kids...)
}
