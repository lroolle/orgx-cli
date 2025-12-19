package iostreams

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/term"
)

type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	colorEnabled bool
	stdinTTY     bool
	stdoutTTY    bool
	stderrTTY    bool
	neverPrompt  bool
}

func System() *IOStreams {
	stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
	stdoutTTY := term.IsTerminal(int(os.Stdout.Fd()))
	stderrTTY := term.IsTerminal(int(os.Stderr.Fd()))

	return &IOStreams{
		In:           os.Stdin,
		Out:          os.Stdout,
		ErrOut:       os.Stderr,
		colorEnabled: stdoutTTY,
		stdinTTY:     stdinTTY,
		stdoutTTY:    stdoutTTY,
		stderrTTY:    stderrTTY,
	}
}

func Test() (ios *IOStreams, stdin *bytes.Buffer, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	stdin = &bytes.Buffer{}
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}

	ios = &IOStreams{
		In:           stdin,
		Out:          stdout,
		ErrOut:       stderr,
		colorEnabled: false,
		stdinTTY:     false,
		stdoutTTY:    false,
		stderrTTY:    false,
		neverPrompt:  true,
	}

	return
}

func (s *IOStreams) ColorEnabled() bool {
	return s.colorEnabled
}

func (s *IOStreams) IsStdinTTY() bool {
	return s.stdinTTY
}

func (s *IOStreams) IsStdoutTTY() bool {
	return s.stdoutTTY
}

func (s *IOStreams) IsStderrTTY() bool {
	return s.stderrTTY
}

func (s *IOStreams) CanPrompt() bool {
	return s.stdinTTY && s.stdoutTTY && !s.neverPrompt
}

func (s *IOStreams) SetColorEnabled(v bool) {
	s.colorEnabled = v
}

func (s *IOStreams) SetStdinTTY(v bool) {
	s.stdinTTY = v
}

func (s *IOStreams) SetStdoutTTY(v bool) {
	s.stdoutTTY = v
}

func (s *IOStreams) SetStderrTTY(v bool) {
	s.stderrTTY = v
}

func (s *IOStreams) SetNeverPrompt(v bool) {
	s.neverPrompt = v
}
