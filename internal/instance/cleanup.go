package instance

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// CleanupKinds are the only content a cleanup is allowed to touch: what the
// game writes for its own sake and writes again the next time it runs.
//
// Worlds, screenshots, mods and configuration are deliberately not here, and
// there is no option to add them. A button that can delete a world is a
// button that will one day delete a world; those already have their own
// per-file delete, where the player is looking at the one thing they mean.
func CleanupKinds() []ContentKind {
	return []ContentKind{ContentLogs, ContentCrashReports}
}

// LauncherLogsDir is where the launcher writes its own log of each start,
// one file per launch, beside the instances rather than inside one. Nothing
// in the window lists them, so they are the one heap that grows without
// anybody ever seeing it.
func (m *Manager) LauncherLogsDir() string {
	return filepath.Join(m.AppDir, "logs")
}

// CleanupGroup is one heap a cleanup found: a kind of throwaway file in one
// instance, or the launcher's own logs.
type CleanupGroup struct {
	// Instance is the instance the files belong to, or empty for the
	// launcher's own logs, which belong to no instance.
	Instance string
	Kind     ContentKind
	Files    int
	Bytes    int64
}

// Label names the heap for a player.
func (g CleanupGroup) Label() string {
	if g.Instance == "" {
		return "Launcher logs"
	}
	return g.Instance + " · " + g.Kind.Label()
}

// CleanupPlan is what a cleanup would remove, or did.
type CleanupPlan struct {
	Groups []CleanupGroup
	Files  int
	Bytes  int64
	// Kept counts files left behind because they are newer than the cutoff.
	// A plan that removes nothing reads very differently when it is because
	// there is nothing there than when it is because everything is recent.
	Kept int
	// Failed counts files that would not go. A log the running game still
	// holds open cannot be removed on Windows; that is worth saying and not
	// worth failing over.
	Failed int
}

// Empty reports whether there is nothing to do.
func (p CleanupPlan) Empty() bool { return p.Files == 0 }

// CleanupAges are the cutoffs the window offers. Zero means everything,
// which is the honest default for files the game rewrites anyway.
func CleanupAges() []time.Duration {
	return []time.Duration{0, 7 * 24 * time.Hour, 30 * 24 * time.Hour, 90 * 24 * time.Hour}
}

// CleanupAgeLabel names a cutoff, short enough that four of them fit across
// a dialog. "Over 30 days" says which side of the line is taken without a
// caption above the row explaining it.
func CleanupAgeLabel(d time.Duration) string {
	if d <= 0 {
		return "Everything"
	}
	return "Over " + strconv.Itoa(int(d/(24*time.Hour))) + " days"
}

// PlanCleanup measures what a cleanup would remove without removing
// anything. An empty instanceName means every instance and the launcher's
// own logs; olderThan of zero means everything, whatever its age.
func (m *Manager) PlanCleanup(instanceName string, olderThan time.Duration) (CleanupPlan, error) {
	return m.cleanup(instanceName, olderThan, false)
}

// RunCleanup removes the files and reports what actually went, which is not
// always what was planned: a file can disappear, or refuse to, between the
// two.
func (m *Manager) RunCleanup(instanceName string, olderThan time.Duration) (CleanupPlan, error) {
	return m.cleanup(instanceName, olderThan, true)
}

// cleanup walks the heaps once, measuring and optionally deleting. The two
// share a body so a plan can never describe something the run would not do.
func (m *Manager) cleanup(instanceName string, olderThan time.Duration, remove bool) (CleanupPlan, error) {
	cutoff := time.Time{}
	if olderThan > 0 {
		cutoff = time.Now().Add(-olderThan)
	}

	names := []string{instanceName}
	global := instanceName == ""
	if global {
		instances, err := m.ListInstances()
		if err != nil {
			return CleanupPlan{}, err
		}
		names = names[:0]
		for _, inst := range instances {
			names = append(names, inst.Name)
		}
	}

	var plan CleanupPlan
	for _, name := range names {
		for _, kind := range CleanupKinds() {
			group, err := m.cleanupContent(name, kind, cutoff, remove, &plan)
			if err != nil {
				return CleanupPlan{}, err
			}
			if group.Files > 0 {
				plan.Groups = append(plan.Groups, group)
			}
		}
	}
	if global {
		group, err := m.cleanupLauncherLogs(cutoff, remove, &plan)
		if err != nil {
			return CleanupPlan{}, err
		}
		if group.Files > 0 {
			plan.Groups = append(plan.Groups, group)
		}
	}

	// Biggest first: the heap worth clearing is the one to read first.
	sort.SliceStable(plan.Groups, func(i, j int) bool {
		return plan.Groups[i].Bytes > plan.Groups[j].Bytes
	})
	return plan, nil
}

// cleanupContent handles one kind in one instance, adding to the totals.
func (m *Manager) cleanupContent(name string, kind ContentKind, cutoff time.Time, remove bool, plan *CleanupPlan) (CleanupGroup, error) {
	entries, err := m.ListContent(name, kind)
	if err != nil {
		return CleanupGroup{}, err
	}
	group := CleanupGroup{Instance: name, Kind: kind}
	for _, e := range entries {
		if !cutoff.IsZero() && e.ModTime.After(cutoff) {
			plan.Kept++
			continue
		}
		if remove {
			if err := m.DeleteContent(name, kind, e.Name); err != nil {
				plan.Failed++
				continue
			}
		}
		group.Files++
		group.Bytes += e.Size
		plan.Files++
		plan.Bytes += e.Size
	}
	return group, nil
}

// cleanupLauncherLogs handles the launcher's own log directory, which is not
// an instance and so has no content listing of its own.
func (m *Manager) cleanupLauncherLogs(cutoff time.Time, remove bool, plan *CleanupPlan) (CleanupGroup, error) {
	dir := m.LauncherLogsDir()
	items, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return CleanupGroup{}, nil
		}
		return CleanupGroup{}, err
	}

	group := CleanupGroup{}
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		info, err := item.Info()
		if err != nil {
			continue
		}
		if !cutoff.IsZero() && info.ModTime().After(cutoff) {
			plan.Kept++
			continue
		}
		if remove {
			if err := os.Remove(filepath.Join(dir, item.Name())); err != nil {
				plan.Failed++
				continue
			}
		}
		group.Files++
		group.Bytes += info.Size()
		plan.Files++
		plan.Bytes += info.Size()
	}
	return group, nil
}
