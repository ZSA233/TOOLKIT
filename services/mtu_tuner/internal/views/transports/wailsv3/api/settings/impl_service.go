package settings

import (
	shared "mtu-tuner/internal/views/routes/api/settings"
	wailstransport "mtu-tuner/internal/views/transports/wailsv3"
)

func NewService(dispatcher wailstransport.EventDispatcher) *SettingsService {
	return newGeneratedSettingsService(shared.NewRouter(), dispatcher)
}
