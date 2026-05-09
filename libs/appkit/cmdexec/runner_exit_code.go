package cmdexec

import (
	"os"
)

func exitCode(state *os.ProcessState, runErr error) int {
	if state != nil {
		return state.ExitCode()
	}
	if runErr != nil {
		return -1
	}
	return 0
}
