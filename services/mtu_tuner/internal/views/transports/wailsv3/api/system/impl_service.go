package system

import (
	shared "mtu-tuner/internal/views/routes/api/system"
	wailstransport "mtu-tuner/internal/views/transports/wailsv3"
)

func NewService(dispatcher wailstransport.EventDispatcher) *SystemService {
	return newGeneratedSystemService(shared.NewRouter(), dispatcher)
}
