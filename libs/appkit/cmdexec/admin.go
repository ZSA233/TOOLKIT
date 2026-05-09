package cmdexec

import (
	"os"
	"runtime"
)

func IsAdmin() bool {
	if runtime.GOOS == "windows" {
		return isWindowsAdmin()
	}
	return os.Geteuid() == 0
}
