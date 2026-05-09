package network

import (
	shared "mtu-tuner/internal/views/routes/api/network"
	wailstransport "mtu-tuner/internal/views/transports/wailsv3"
)

func NewService(dispatcher wailstransport.EventDispatcher) *NetworkService {
	return newGeneratedNetworkService(shared.NewRouter(), dispatcher)
}
