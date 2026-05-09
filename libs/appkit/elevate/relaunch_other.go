//go:build !windows

package elevate

func SupportsAdminRelaunch() bool {
	return false
}

func AdminRelaunchConfirmLabel() string {
	return "Relaunch as Admin"
}

func AdminRelaunchCancelLabel() string {
	return "Cancel"
}

func RelaunchCurrentProcessAsAdmin() error {
	return nil
}
