package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ShellService struct{}

func NewShellService() *ShellService {
	return &ShellService{}
}

func (service *ShellService) PickClashConfigPath() (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle("Select Clash Config").
		AddFilter("YAML", "*.yaml;*.yml").
		PromptForSingleSelection()
}

func (service *ShellService) PickBrowserPath() (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle("Select Browser Executable").
		AddFilter("Executables", "*.exe;*").
		PromptForSingleSelection()
}
