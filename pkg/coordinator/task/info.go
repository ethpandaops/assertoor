package task

type MapOfRunnableInfo map[string]RunnableInfo

// RunnableInfo contains information about a runnable task.
type RunnableInfo struct {
	Description string
	Config      interface{}
}
