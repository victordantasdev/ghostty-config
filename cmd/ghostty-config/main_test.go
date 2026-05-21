package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRunSuccess(t *testing.T) {
	called := false
	prevExit := exitFn
	exitFn = func(int) { called = true }
	defer func() { exitFn = prevExit }()

	run(func() error { return nil })
	if called {
		t.Errorf("exit should not be called on success")
	}
}

func TestRunFailure(t *testing.T) {
	prevExit := exitFn
	prevStderr := stderr
	defer func() {
		exitFn = prevExit
		stderr = prevStderr
	}()

	code := 0
	exitFn = func(c int) { code = c }
	buf := &bytes.Buffer{}
	stderr = buf

	run(func() error { return errors.New("boom") })
	if code != 1 {
		t.Errorf("code = %d", code)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Errorf("stderr = %q", buf.String())
	}
}

func TestMainInvokesRun(t *testing.T) {
	prevExit := exitFn
	prevStderr := stderr
	prevArgs := os.Args
	defer func() {
		exitFn = prevExit
		stderr = prevStderr
		os.Args = prevArgs
	}()
	exitFn = func(int) {}
	stderr = &bytes.Buffer{}
	// --version is short-circuited by cobra, returns nil, run() does not exit.
	os.Args = []string{"ghostty-config", "--version"}
	main()
}
