package iostreams

import (
	"testing"
)

func TestSystem(t *testing.T) {
	ios := System()

	if ios.In == nil {
		t.Error("In should not be nil")
	}
	if ios.Out == nil {
		t.Error("Out should not be nil")
	}
	if ios.ErrOut == nil {
		t.Error("ErrOut should not be nil")
	}
}

func TestTest(t *testing.T) {
	ios, stdin, stdout, stderr := Test()

	if ios.In != stdin {
		t.Error("In should be stdin buffer")
	}
	if ios.Out != stdout {
		t.Error("Out should be stdout buffer")
	}
	if ios.ErrOut != stderr {
		t.Error("ErrOut should be stderr buffer")
	}

	if ios.ColorEnabled() {
		t.Error("ColorEnabled should be false in test mode")
	}
	if ios.IsStdinTTY() {
		t.Error("IsStdinTTY should be false in test mode")
	}
	if ios.IsStdoutTTY() {
		t.Error("IsStdoutTTY should be false in test mode")
	}
	if ios.CanPrompt() {
		t.Error("CanPrompt should be false in test mode")
	}
}

func TestSetters(t *testing.T) {
	ios, _, _, _ := Test()

	ios.SetColorEnabled(true)
	if !ios.ColorEnabled() {
		t.Error("ColorEnabled should be true after SetColorEnabled(true)")
	}

	ios.SetStdinTTY(true)
	if !ios.IsStdinTTY() {
		t.Error("IsStdinTTY should be true after SetStdinTTY(true)")
	}

	ios.SetStdoutTTY(true)
	if !ios.IsStdoutTTY() {
		t.Error("IsStdoutTTY should be true after SetStdoutTTY(true)")
	}

	ios.SetNeverPrompt(false)
	if !ios.CanPrompt() {
		t.Error("CanPrompt should be true when stdin and stdout are TTY and neverPrompt is false")
	}

	ios.SetNeverPrompt(true)
	if ios.CanPrompt() {
		t.Error("CanPrompt should be false when neverPrompt is true")
	}
}
