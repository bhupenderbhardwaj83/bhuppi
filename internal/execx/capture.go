package execx

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/bhupenderbhardwaj/bhuppi/internal/pager"
)

// RunCaptured runs argv with stdout piped into the pager. Argv is not rewritten.
// Stderr and stdin stay attached to the user's terminal (stdin is /dev/null
// only when stdinNull). The child's exit code is returned. q stops the rest
// of the output and kills the child.
func RunCaptured(argv []string, stdinNull bool, stdout, stderr io.Writer) (int, error) {
	if len(argv) == 0 {
		return 2, fmt.Errorf("empty argv")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stderr = stderr
	if stdinNull {
		cmd.Stdin = nil
	} else {
		cmd.Stdin = os.Stdin
	}
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return 1, err
	}
	if err := cmd.Start(); err != nil {
		return 1, err
	}
	perr := pager.Copy(stdout, pipe)
	if errors.Is(perr, pager.ErrQuit) {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_, _ = io.Copy(io.Discard, pipe)
		_ = cmd.Wait()
		return 0, nil
	}
	werr := cmd.Wait()
	if perr != nil {
		return 1, perr
	}
	if werr == nil {
		return 0, nil
	}
	if ee, ok := werr.(*exec.ExitError); ok {
		return ee.ExitCode(), nil
	}
	return 1, werr
}
