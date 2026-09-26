package main

import "testing"

// TestVersionIsStampable is a compile-time guard wearing a test's clothes:
// the assignment below only builds while Version is a variable. Make it a
// constant and this file stops compiling — which is the only warning there
// is. The linker's -X writes into a string variable and does nothing at all
// to a constant, without a word, so the build goes green and the binary
// quietly calls itself -dev. That is exactly what shipped in the CLI up to
// v2.2.1.
func TestVersionIsStampable(t *testing.T) {
	was := Version
	defer func() { Version = was }()

	Version = "stamped"
	if Version != "stamped" {
		t.Fatal("Version did not take the value assigned to it")
	}
}
