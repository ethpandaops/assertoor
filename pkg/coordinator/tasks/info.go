package tasks

type MapOfRunnableInfo map[string]RunnableInfo

// RunnableInfo contains information about a runnable task.
type RunnableInfo struct {
	Description string
	Config      interface{}
}

func AvailableTasks() MapOfRunnableInfo {
	available := MapOfRunnableInfo{}
	for _, taskDescriptor := range AvailableTaskDescriptors {
		available[taskDescriptor.Name] = RunnableInfo{
			Description: taskDescriptor.Description,
			Config:      taskDescriptor.Config,
		}
	}
	return available
}
