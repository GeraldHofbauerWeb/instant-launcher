package instance

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// cleanupManager builds a manager over a temporary directory with two
// instances, and returns it.
func cleanupManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	m := &Manager{
		AppDir:        dir,
		InstancesPath: filepath.Join(dir, "instances"),
		MinecraftPath: filepath.Join(dir, "minecraft"),
		BackupPath:    dir,
	}
	if err := os.MkdirAll(m.InstancesPath, 0o755); err != nil {
		t.Fatal(err)
	}
	return m
}

// write puts a file of n bytes at path, aged by age.
func write(t *testing.T, path string, n int, age time.Duration) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

// TestCleanupTakesOnlyWhatTheGameRewrites is the guarantee the button rests
// on. Everything else in an instance belongs to the player, and no age and
// no scope may ever reach it.
func TestCleanupTakesOnlyWhatTheGameRewrites(t *testing.T) {
	m := cleanupManager(t)
	inst := filepath.Join(m.InstancesPath, "pack")
	write(t, filepath.Join(inst, "logs", "latest.log"), 100, 0)
	write(t, filepath.Join(inst, "crash-reports", "crash.txt"), 50, 0)
	write(t, filepath.Join(inst, "screenshots", "shot.png"), 999, 0)
	write(t, filepath.Join(inst, "mods", "create.jar"), 999, 0)
	write(t, filepath.Join(inst, "config", "jei.toml"), 999, 0)
	if err := os.MkdirAll(filepath.Join(inst, "saves", "world"), 0o755); err != nil {
		t.Fatal(err)
	}

	plan, err := m.RunCleanup("pack", 0)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Files != 2 || plan.Bytes != 150 {
		t.Errorf("removed %d files / %d bytes, want 2 / 150", plan.Files, plan.Bytes)
	}

	for _, keep := range []string{"screenshots/shot.png", "mods/create.jar", "config/jei.toml", "saves/world"} {
		if _, err := os.Stat(filepath.Join(inst, keep)); err != nil {
			t.Errorf("cleanup took %s, which is the player's: %v", keep, err)
		}
	}
	for _, gone := range []string{"logs/latest.log", "crash-reports/crash.txt"} {
		if _, err := os.Stat(filepath.Join(inst, gone)); !os.IsNotExist(err) {
			t.Errorf("%s survived the cleanup", gone)
		}
	}
}

// TestCleanupRespectsTheCutoff: the age in the dialog decides, and a file on
// the young side of it is not counted as kept-by-accident but as kept.
func TestCleanupRespectsTheCutoff(t *testing.T) {
	m := cleanupManager(t)
	inst := filepath.Join(m.InstancesPath, "pack")
	write(t, filepath.Join(inst, "logs", "old.log"), 10, 40*24*time.Hour)
	write(t, filepath.Join(inst, "logs", "recent.log"), 10, 2*24*time.Hour)

	plan, err := m.PlanCleanup("pack", 30*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Files != 1 || plan.Kept != 1 {
		t.Fatalf("plan: %d to go, %d kept; want 1 and 1", plan.Files, plan.Kept)
	}

	// A plan measures and does not touch anything.
	if _, err := os.Stat(filepath.Join(inst, "logs", "old.log")); err != nil {
		t.Error("PlanCleanup deleted a file")
	}

	if _, err := m.RunCleanup("pack", 30*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(inst, "logs", "old.log")); !os.IsNotExist(err) {
		t.Error("the old log survived")
	}
	if _, err := os.Stat(filepath.Join(inst, "logs", "recent.log")); err != nil {
		t.Error("the recent log was taken")
	}
}

// TestGlobalCleanupReachesEveryInstanceAndTheLauncherItself: the launcher's
// own start-up logs are the heap nothing in the window lists, so the global
// cleanup is the only thing that will ever clear them.
func TestGlobalCleanupReachesEveryInstanceAndTheLauncherItself(t *testing.T) {
	m := cleanupManager(t)
	write(t, filepath.Join(m.InstancesPath, "one", "logs", "a.log"), 10, 0)
	write(t, filepath.Join(m.InstancesPath, "two", "crash-reports", "b.txt"), 20, 0)
	write(t, filepath.Join(m.LauncherLogsDir(), "one-20260927.log"), 30, 0)

	plan, err := m.PlanCleanup("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Files != 3 || plan.Bytes != 60 {
		t.Fatalf("plan = %d files / %d bytes, want 3 / 60; groups %+v", plan.Files, plan.Bytes, plan.Groups)
	}

	var sawLauncher bool
	for _, g := range plan.Groups {
		if g.Instance == "" {
			sawLauncher = true
			if g.Label() != "Launcher logs" {
				t.Errorf("launcher group labelled %q", g.Label())
			}
		}
	}
	if !sawLauncher {
		t.Error("the launcher's own logs are not in a global cleanup")
	}

	// Biggest heap first, so the dialog reads usefully.
	for i := 1; i < len(plan.Groups); i++ {
		if plan.Groups[i-1].Bytes < plan.Groups[i].Bytes {
			t.Errorf("groups are not ordered by size: %+v", plan.Groups)
			break
		}
	}
}

// TestPlanAndRunAgree: the number in the confirmation must be the number
// that then happens, or the dialog is a guess.
func TestPlanAndRunAgree(t *testing.T) {
	m := cleanupManager(t)
	write(t, filepath.Join(m.InstancesPath, "pack", "logs", "a.log"), 11, 0)
	write(t, filepath.Join(m.InstancesPath, "pack", "logs", "b.log"), 22, 0)
	write(t, filepath.Join(m.InstancesPath, "pack", "crash-reports", "c.txt"), 33, 0)

	plan, err := m.PlanCleanup("pack", 0)
	if err != nil {
		t.Fatal(err)
	}
	done, err := m.RunCleanup("pack", 0)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Files != done.Files || plan.Bytes != done.Bytes {
		t.Errorf("planned %d/%d, removed %d/%d", plan.Files, plan.Bytes, done.Files, done.Bytes)
	}
	if done.Failed != 0 {
		t.Errorf("%d files would not go", done.Failed)
	}
}

// TestCleanupOfAnEmptyInstanceIsNotAnError: a fresh instance has no logs,
// and asking to clean it is a no-op rather than a failure.
func TestCleanupOfAnEmptyInstanceIsNotAnError(t *testing.T) {
	m := cleanupManager(t)
	if err := os.MkdirAll(filepath.Join(m.InstancesPath, "fresh"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := m.RunCleanup("fresh", 0)
	if err != nil {
		t.Fatalf("cleaning a fresh instance failed: %v", err)
	}
	if !plan.Empty() {
		t.Errorf("plan over a fresh instance is not empty: %+v", plan)
	}
}
