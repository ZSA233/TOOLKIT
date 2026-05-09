package app

import "fmt"

func (service *Service) hasAdminPrivileges() bool {
	if service != nil && service.isAdmin != nil {
		return service.isAdmin()
	}
	return false
}

func (service *Service) ensureAdminPrivilegesFor(action string) error {
	if service.hasAdminPrivileges() {
		return nil
	}
	// Reject MTU-changing workflows before background work starts so the GUI can
	// surface an actionable elevation prompt instead of a short-lived no-op task.
	return fmt.Errorf("%s requires admin/root privileges", action)
}
