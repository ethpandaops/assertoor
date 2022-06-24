package test

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/task"
)

type Metrics struct {
	testName     string
	TaskInfo     *prometheus.GaugeVec
	TestDuration *prometheus.GaugeVec
	TestResult   *prometheus.GaugeVec

	CurrentTask  *prometheus.GaugeVec
	TotalTasks   *prometheus.GaugeVec
	TaskDuration *prometheus.GaugeVec
	TestInfo     *prometheus.GaugeVec
}

// NewMetrics returns a new Metrics instance.
func NewMetrics(namespace, testName string) Metrics {
	return Metrics{
		testName: testName,
		TestInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "test_info",
				Help:      "Information about the coordinator test",
			},
			[]string{"test", "description"},
		),
		TestDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "test_duration_ms",
				Help:      "The duration of the test (in milliseconds.)",
			},
			[]string{"test"},
		),
		TestResult: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "test_result",
				Help:      "The result of the test.",
			},
			[]string{"test", "result"},
		),
		CurrentTask: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "current_task",
				Help:      "The current task being executed",
			},
			[]string{"test", "task", "index"},
		),
		TotalTasks: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "total_tasks",
				Help:      "The total number of tasks",
			},
			[]string{"test"},
		),
		TaskDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "task_duration_ms",
				Help:      "The duration of the task (in milliseconds.)",
			},
			[]string{"test", "task", "index"},
		),
		TaskInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "task_info",
				Help:      "The tasks that will be run",
			},
			[]string{"test", "task", "description", "title", "index"},
		),
	}
}

func (m *Metrics) Register() {
	prometheus.MustRegister(m.TestInfo)
	prometheus.MustRegister(m.TestDuration)
	prometheus.MustRegister(m.CurrentTask)
	prometheus.MustRegister(m.TotalTasks)
	prometheus.MustRegister(m.TaskDuration)
	prometheus.MustRegister(m.TaskInfo)
}

func (m *Metrics) SetTestInfo(description string) {
	m.TestInfo.WithLabelValues(m.testName, description).Set(1)
}

func (m *Metrics) SetCurrentTask(name string, index int) {
	m.CurrentTask.Reset()
	m.CurrentTask.WithLabelValues(m.testName, name, fmt.Sprintf("%d", index)).Set(float64(index))
}

func (m *Metrics) SetTotalTasks(total float64) {
	m.TotalTasks.WithLabelValues(m.testName).Set(total)
}

func (m *Metrics) SetTestDuration(duration float64) {
	m.TestDuration.WithLabelValues(m.testName).Set(duration)
}

func (m *Metrics) SetTaskDuration(name, index string, duration float64) {
	m.TaskDuration.WithLabelValues(m.testName, name, index).Set(duration)
}

func (m *Metrics) SetTaskInfo(ta task.Runnable, index int) {
	m.TaskInfo.WithLabelValues(m.testName, ta.Name(), ta.Description(), ta.Title(), fmt.Sprintf("%d", index)).Set(float64(index))
}
