package main

import (
	"fmt"
	"strings"

	"mtu-tuner/internal/core"

	"toolkit/libs/appkit/elevate"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (service *ShellService) PromptAdminRelaunch(reason string) (bool, error) {
	if !elevate.SupportsAdminRelaunch() {
		return false, fmt.Errorf("automatic admin relaunch is only supported on Windows")
	}

	app := application.Get()
	if app == nil {
		return false, fmt.Errorf("application is not ready")
	}

	message := buildAdminRelaunchMessage(reason)
	confirmed := false
	dialog := app.Dialog.Question().
		SetTitle("Administrator Permission Required").
		SetMessage(message)
	cancelButton := dialog.AddButton(elevate.AdminRelaunchCancelLabel()).SetAsCancel()
	relaunchButton := dialog.AddButton(elevate.AdminRelaunchConfirmLabel()).SetAsDefault().OnClick(func() {
		confirmed = true
	})
	dialog.SetCancelButton(cancelButton).SetDefaultButton(relaunchButton).Show()

	if !confirmed {
		return false, nil
	}
	if err := elevate.RelaunchCurrentProcessAsAdmin(); err != nil {
		return false, err
	}
	app.Quit()
	return true, nil
}

func buildAdminRelaunchMessage(reason string) string {
	action := strings.TrimSpace(reason)
	if action == "" {
		action = "apply MTU changes"
	}
	return fmt.Sprintf("Administrator privileges are required to %s.\n\nRelaunch %s as Administrator now?", action, core.AppName)
}
