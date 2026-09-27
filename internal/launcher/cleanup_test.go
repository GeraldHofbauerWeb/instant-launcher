package launcher

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GeraldHofbauerWeb/instant-launcher/internal/instance"
)

// TestCleanupMeasuresThenRemovesAndRelists walks the path the dialog takes:
// open it, see a count, press the button, and find the overview already
// showing the new numbers rather than the ones from before.
func TestCleanupMeasuresThenRemovesAndRelists(t *testing.T) {
	ctrl := newTestController(t, nil)
	isolateInstances(t, ctrl)
	dir := filepath.Join(ctrl.Manager.InstancesPath, "pack")
	for _, sub := range []string{"logs", "crash-reports", "saves/world"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"logs/a.log", "logs/b.log", "crash-reports/c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("0123456789"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ctrl.Dispatch(ActionRefresh{})
	ctrl.Dispatch(ActionSelect{Name: "pack"})
	waitFor(t, ctrl, "the instance listed", func(s Snapshot) bool {
		return s.ContentFor == "pack" && len(s.ContentOf(instance.ContentLogs)) == 2
	})

	ctrl.Dispatch(ActionPlanCleanup{Name: "pack"})
	snap := waitFor(t, ctrl, "the plan to be measured", func(s Snapshot) bool {
		return s.Cleanup.For("pack", 0)
	})
	if snap.Cleanup.Plan.Files != 3 || snap.Cleanup.Plan.Bytes != 30 {
		t.Fatalf("plan = %d files / %d bytes, want 3 / 30", snap.Cleanup.Plan.Files, snap.Cleanup.Plan.Bytes)
	}

	ctrl.Dispatch(ActionCleanup{Name: "pack"})
	waitFor(t, ctrl, "the overview to show the cleaned instance", func(s Snapshot) bool {
		return len(s.ContentOf(instance.ContentLogs)) == 0 &&
			len(s.ContentOf(instance.ContentCrashReports)) == 0
	})

	// The world is the point of the whole exercise: it stays.
	if _, err := os.Stat(filepath.Join(dir, "saves", "world")); err != nil {
		t.Errorf("the cleanup took a world: %v", err)
	}
	// And the dialog's plan is cleared, so reopening it measures afresh
	// rather than showing the numbers it just acted on.
	if got := ctrl.Store().Snapshot().Cleanup; got.Measured {
		t.Errorf("a stale plan survived the cleanup: %+v", got)
	}
}

// TestGlobalCleanupFromTheSettingsScreen: an empty name means everything,
// including the launcher's own logs.
func TestGlobalCleanupFromTheSettingsScreen(t *testing.T) {
	ctrl := newTestController(t, nil)
	isolateInstances(t, ctrl)
	for _, inst := range []string{"one", "two"} {
		if err := os.MkdirAll(filepath.Join(ctrl.Manager.InstancesPath, inst, "logs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ctrl.Manager.InstancesPath, inst, "logs", "a.log"), []byte("xx"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	launcherLog := filepath.Join(ctrl.Manager.LauncherLogsDir(), "one-20260927.log")
	if err := os.MkdirAll(filepath.Dir(launcherLog), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(launcherLog, []byte("xx"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctrl.Dispatch(ActionRefresh{})
	ctrl.Dispatch(ActionPlanCleanup{})
	snap := waitFor(t, ctrl, "the global plan", func(s Snapshot) bool { return s.Cleanup.For("", 0) })
	if snap.Cleanup.Plan.Files != 3 {
		t.Fatalf("global plan = %d files, want 3 (two instances and the launcher)", snap.Cleanup.Plan.Files)
	}

	ctrl.Dispatch(ActionCleanup{})
	waitFor(t, ctrl, "the cleanup to be reported", func(s Snapshot) bool {
		return s.Status != "" && !s.Cleanup.Measured
	})
	if _, err := os.Stat(launcherLog); !os.IsNotExist(err) {
		t.Error("the launcher's own log survived a global cleanup")
	}
}
