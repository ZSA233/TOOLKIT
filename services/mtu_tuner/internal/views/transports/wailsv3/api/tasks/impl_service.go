package tasks

import (
	shared "mtu-tuner/internal/views/routes/api/tasks"
	wailstransport "mtu-tuner/internal/views/transports/wailsv3"
)

func NewService(dispatcher wailstransport.EventDispatcher) *TasksService {
	return newGeneratedTasksService(shared.NewRouter(), dispatcher)
}
