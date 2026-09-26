package launcher

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/GeraldHofbauerWeb/instant-launcher/internal/launch"
)

// TestStopGameIsNotReportedAsACrash walks the path the Stop button takes:
// a running game, ActionStopGame, and the state the workbench then draws
// from. The launcher used to print "The game ended with an error: exit
// status 143" over a world the JVM had just saved properly, because a
// deliberate stop and a crash reach the controller looking identical.
func TestStopGameIsNotReportedAsACrash(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the shell stand-in and SIGTERM are Unix only")
	}
	ctrl := newTestController(t, nil)

	// A stand-in for the JVM: it takes SIGTERM, tidies up and exits 143 of
	// its own accord, exactly as the real one does.
	proc, err := launch.Start(context.Background(), launch.Spec{
		JavaPath: "/bin/sh",
		Args:     []string{"-c", `trap 'exit 143' TERM; while true; do sleep 0.02; done`},
		GameDir:  t.TempDir(),
		LogDir:   t.TempDir(),
		Name:     "stand-in",
	})
	if err != nil {
		t.Fatalf("starting the stand-in: %v", err)
	}
	time.Sleep(200 * time.Millisecond) // let the trap be installed

	ctrl.mu.Lock()
	ctrl.game = proc
	ctrl.mu.Unlock()
	ctrl.Store().SetGame(GameState{Instance: "pack", PID: proc.Cmd.Process.Pid, Running: true})
	go ctrl.watchGame("pack", proc)

	ctrl.Dispatch(ActionStopGame{})

	snap := waitFor(t, ctrl, "the game to be recorded as finished", func(s Snapshot) bool {
		return !s.Game.Running
	})
	if snap.Game.ExitErr != nil {
		t.Errorf("a stop we asked for surfaced as an error: %v", snap.Game.ExitErr)
	}
	if snap.Status != "pack stopped" {
		t.Errorf("status = %q, want the stop confirmed", snap.Status)
	}
}

// TestGameCrashIsStillReported guards the other direction: the fix must not
// swallow a real failure, which is the whole reason the notice exists.
func TestGameCrashIsStillReported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the shell stand-in is Unix only")
	}
	ctrl := newTestController(t, nil)

	proc, err := launch.Start(context.Background(), launch.Spec{
		JavaPath: "/bin/sh",
		Args:     []string{"-c", `exit 1`},
		GameDir:  t.TempDir(),
		LogDir:   t.TempDir(),
		Name:     "stand-in",
	})
	if err != nil {
		t.Fatalf("starting the stand-in: %v", err)
	}

	ctrl.mu.Lock()
	ctrl.game = proc
	ctrl.mu.Unlock()
	ctrl.Store().SetGame(GameState{Instance: "pack", PID: proc.Cmd.Process.Pid, Running: true})
	go ctrl.watchGame("pack", proc)

	snap := waitFor(t, ctrl, "the crash to be recorded", func(s Snapshot) bool {
		return !s.Game.Running
	})
	if snap.Game.ExitErr == nil {
		t.Error("a game that died on its own was reported as a clean exit")
	}
}
