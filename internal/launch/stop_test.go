package launch

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// fakeGame starts a shell standing in for the JVM. script runs in the
// foreground, so a trap in it reaches the process we signal.
func fakeGame(t *testing.T, script string) *Process {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the shell stand-in and SIGTERM are Unix only; Windows takes the kill path")
	}
	p, err := Start(context.Background(), Spec{
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

// TestStopIsRememberedThroughTheExitCode is the whole point of Process.Stopped.
// The stand-in behaves exactly as the JVM does: it catches the signal, tidies
// up, and then exits 143 by its own choice. os/exec sees a plain non-zero
// exit, so the exit code cannot tell the caller what happened — only the flag
// can.
func TestStopIsRememberedThroughTheExitCode(t *testing.T) {
	p := fakeGame(t, `trap 'exit 143' TERM; while true; do sleep 0.02; done`)

	// Give the shell a moment to install the trap; signalled before that, it
	// dies on the default action and the test proves nothing.
	time.Sleep(200 * time.Millisecond)

	if err := p.Stop(5 * time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := p.Wait(); err == nil {
		t.Fatal("the stand-in exited cleanly; it was supposed to exit 143, so this test no longer tests anything")
	}
	if !p.Stopped() {
		t.Error("Stopped() is false after Stop(); the launcher will report the stop as a crash")
	}
}

// TestStoppedStaysFalseWhenTheGameFailsOnItsOwn is the other half: a genuine
// crash must keep being reported as one.
func TestStoppedStaysFalseWhenTheGameFailsOnItsOwn(t *testing.T) {
	p := fakeGame(t, `exit 1`)

	if err := p.Wait(); err == nil {
		t.Fatal("the stand-in exited 0")
	}
	if p.Stopped() {
		t.Error("Stopped() is true for a game nobody stopped")
	}
}

// TestStopAfterTheGameIsGoneClaimsNothing covers the race where the game dies
// on its own just as the button is pressed: Stop finds nothing to signal, and
// must not then take credit for the crash.
func TestStopAfterTheGameIsGoneClaimsNothing(t *testing.T) {
	p := fakeGame(t, `exit 1`)
	_ = p.Wait()

	if err := p.Stop(time.Second); err != nil {
		t.Fatalf("Stop on a dead process: %v", err)
	}
	if p.Stopped() {
		t.Error("Stopped() is true although Stop had nothing left to signal")
	}
}
