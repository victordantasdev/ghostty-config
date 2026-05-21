package main

import (
	"fmt"
	"io"
	"os"

	"ghostty-config/internal/cli"
)

var (
	exitFn = os.Exit
	stderr io.Writer = os.Stderr
)

func main() {
	run(cli.Execute)
}

func run(execute func() error) {
	if err := execute(); err != nil {
		fmt.Fprintln(stderr, err)
		exitFn(1)
	}
}
