// Package tui draws the Kocoro CLI home screen (Start) and keeps a line
// reader (Run) for pipes and tests. The full production UI is internal/tui.
package tui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Turn runs one user line and returns the assistant text.
type Turn func(text string) (string, error)

// Run loops until the user types exit or quit, or input ends.
func Run(in io.Reader, out io.Writer, turn Turn) error {
	sc := bufio.NewScanner(in)
	for {
		if _, err := fmt.Fprint(out, "> "); err != nil {
			return err
		}
		if !sc.Scan() {
			return sc.Err()
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		reply, err := turn(line)
		if err != nil {
			if _, werr := fmt.Fprintf(out, "error: %s\n", err.Error()); werr != nil {
				return werr
			}
			continue
		}
		if _, err := fmt.Fprintln(out, reply); err != nil {
			return err
		}
	}
}
