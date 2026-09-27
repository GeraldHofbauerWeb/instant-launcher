package launcher

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/GeraldHofbauerWeb/instant-launcher/internal/instance"
	"github.com/GeraldHofbauerWeb/instant-launcher/internal/launch"
)

// playableInstance makes an instance with the directories a session writes
// into, and returns its path.
func playableInstance(t *testing.T, ctrl *Controller, name string) string {
	t.Helper()
	m := ctrl.Manager
	dir := filepath.Join(m.InstancesPath, name)
	for _, sub := range []string{"mods", "saves", "screenshots", "logs", "config"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// standIn starts a shell in place of the JVM, so the controller's watchGame
// runs against a real process without a game being anywhere near this test.
func standIn(t *testing.T, script string) *launch.Process {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the shell stand-in is Unix only")
	}
	p, err := launch.Start(context.Background(), launch.Spec{
		JavaPath: "/bin/sh",
		Args:     []string{"-c", script},
		GameDir:  t.TempDir(),
		LogDir:   t.TempDir(),
		Name:     "stand-in",
	})
	if err != nil {
		t.Fatalf("starting the stand-in: %v", err)
	}
	return p
}

// TestTheOverviewIsListedAgainAfterASession is the reported bug: play,
// quit, and the counts still describe the instance as it was when it was
// selected. A session writes a log every time and usually a screenshot or a
// world as well, so the numbers on the overview are wrong the moment the
// game closes.
func TestTheOverviewIsListedAgainAfterASession(t *testing.T) {
	ctrl := newTestController(t, nil)
	isolateInstances(t, ctrl)
	dir := playableInstance(t, ctrl, "pack")

	ctrl.Dispatch(ActionRefresh{})
	ctrl.Dispatch(ActionSelect{Name: "pack"})
	waitFor(t, ctrl, "the instance selected and listed", func(s Snapshot) bool {
		return s.Selected == "pack" && s.ContentFor == "pack"
	})
	if got := len(ctrl.Store().Snapshot().ContentOf(instance.ContentScreenshots)); got != 0 {
		t.Fatalf("screenshots before playing = %d, want 0", got)
	}

	proc := standIn(t, `exit 0`)
	go ctrl.watchGame("pack", proc)

	// What the session left behind, written while the game "ran".
	for _, f := range []string{"screenshots/2026-09-27_01.png", "logs/latest.log"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	waitFor(t, ctrl, "the overview to count what the session wrote", func(s Snapshot) bool {
		return len(s.ContentOf(instance.ContentScreenshots)) == 1 &&
			len(s.ContentOf(instance.ContentLogs)) == 1
	})
}

// TestASessionDoesNotBlankAnotherInstancesPanel: the store holds one
// listing, for whichever instance is selected. Reloading the one that just
// finished, when the player has since moved to another, would empty the
// panel in front of them.
func TestASessionDoesNotBlankAnotherInstancesPanel(t *testing.T) {
	ctrl := newTestController(t, nil)
	isolateInstances(t, ctrl)
	playableInstance(t, ctrl, "played")
	other := playableInstance(t, ctrl, "watched")
	if err := os.WriteFile(filepath.Join(other, "mods", "create.jar"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctrl.Dispatch(ActionRefresh{})
	ctrl.Dispatch(ActionSelect{Name: "watched"})
	waitFor(t, ctrl, "the other instance listed", func(s Snapshot) bool {
		return s.ContentFor == "watched" && len(s.ContentOf(instance.ContentMods)) == 1
	})

	proc := standIn(t, `exit 0`)
	done := make(chan struct{})
	go func() { ctrl.watchGame("played", proc); close(done) }()
	<-done

	// Give a stray reload a chance to land before the assertion.
	time.Sleep(200 * time.Millisecond)
	snap := ctrl.Store().Snapshot()
	if snap.ContentFor != "watched" {
		t.Errorf("the listing moved to %q; the panel on screen is blank", snap.ContentFor)
	}
	if got := len(snap.ContentOf(instance.ContentMods)); got != 1 {
		t.Errorf("mods shown for the selected instance = %d, want 1", got)
	}
}

// TestRefreshRelistsTheSelectedInstance: the button says Refresh, and the
// panel the player is looking at is the one thing it used to skip.
func TestRefreshRelistsTheSelectedInstance(t *testing.T) {
	ctrl := newTestController(t, nil)
	isolateInstances(t, ctrl)
	dir := playableInstance(t, ctrl, "pack")

	ctrl.Dispatch(ActionRefresh{})
	ctrl.Dispatch(ActionSelect{Name: "pack"})
	waitFor(t, ctrl, "the instance listed", func(s Snapshot) bool {
		return s.ContentFor == "pack" && len(s.ContentOf(instance.ContentMods)) == 0
	})

	// Something outside the launcher adds a mod — a download, a file
	// manager, a modpack tool.
	if err := os.WriteFile(filepath.Join(dir, "mods", "create.jar"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctrl.Dispatch(ActionRefresh{})
	waitFor(t, ctrl, "the new mod to be listed", func(s Snapshot) bool {
		return len(s.ContentOf(instance.ContentMods)) == 1
	})
}
