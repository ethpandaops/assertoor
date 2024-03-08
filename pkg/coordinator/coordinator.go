package coordinator

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/test"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web/server"
	"github.com/gorhill/cronexpr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Coordinator struct {
	// Config is the coordinator configuration.
	Config         *Config
	log            *logger.LogScope
	clientPool     *clients.ClientPool
	walletManager  *wallet.Manager
	webserver      *server.WebServer
	validatorNames *names.ValidatorNames
	globalVars     types.Variables
	metricsPort    int

	runIDCounter       uint64
	lastExecutedRunID  uint64
	testSchedulerMutex sync.Mutex

	testDescriptors      map[string]testDescriptorEntry
	testDescriptorsMutex sync.RWMutex
	testDescriptorIndex  uint64

	testRunMap           map[uint64]types.Test
	testQueue            []types.Test
	testHistory          []types.Test
	testRegistryMutex    sync.RWMutex
	testNotificationChan chan bool
}

type testDescriptorEntry struct {
	descriptor types.TestDescriptor
	dynamic    bool
	index      uint64
}

func NewCoordinator(config *Config, log logrus.FieldLogger, metricsPort int) *Coordinator {
	return &Coordinator{
		log: logger.NewLogger(&logger.ScopeOptions{
			Parent:      log,
			HistorySize: 5000,
		}),
		Config:      config,
		metricsPort: metricsPort,

		testDescriptors:      map[string]testDescriptorEntry{},
		testRunMap:           map[uint64]types.Test{},
		testQueue:            []types.Test{},
		testHistory:          []types.Test{},
		testNotificationChan: make(chan bool, 1),
	}
}

// Run executes the coordinator until completion.
func (c *Coordinator) Run(ctx context.Context) error {
	defer func() {
		if err := recover(); err != nil {
			c.log.GetLogger().WithError(err.(error)).Errorf("uncaught panic in coordinator.Run: %v, stack: %v", err, string(debug.Stack()))
		}
	}()

	c.log.GetLogger().
		WithField("build_version", buildinfo.GetVersion()).
		WithField("metrics_port", c.metricsPort).
		Info("starting assertoor")

	// init client pool
	clientPool, err := clients.NewClientPool(c.log.GetLogger())
	if err != nil {
		return err
	}

	c.clientPool = clientPool
	c.walletManager = wallet.NewManager(clientPool.GetExecutionPool(), c.log.GetLogger().WithField("module", "wallet"))

	for idx := range c.Config.Endpoints {
		err = clientPool.AddClient(&c.Config.Endpoints[idx])
		if err != nil {
			return err
		}
	}

	// init global variables
	c.globalVars = vars.NewVariables(nil)
	for name, value := range c.Config.GlobalVars {
		c.globalVars.SetVar(name, value)
	}

	// init webserver
	if c.Config.Web != nil && c.Config.Web.Server != nil {
		c.webserver, err = server.NewWebServer(c.Config.Web.Server, c.log.GetLogger())
		if err != nil {
			return err
		}

		if c.Config.Web.API != nil {
			err = c.webserver.ConfigureRoutes(c.Config.Web, c.log.GetLogger(), c)
			if err != nil {
				return err
			}
		}
	}

	//nolint:errcheck // ignore
	go c.startMetrics()

	// load validator names
	c.validatorNames = names.NewValidatorNames(c.Config.ValidatorNames, c.log.GetLogger())
	c.validatorNames.LoadValidatorNames()

	// load tests
	c.LoadTests(ctx)

	// start test scheduler
	go c.runTestScheduler(ctx)

	// start test cleanup routine
	go c.runTestCleanup(ctx)

	// run tests
	c.runTestExecutionLoop(ctx)

	return nil
}

func (c *Coordinator) Logger() logrus.FieldLogger {
	return c.log.GetLogger()
}

func (c *Coordinator) LogScope() *logger.LogScope {
	return c.log
}

func (c *Coordinator) ClientPool() *clients.ClientPool {
	return c.clientPool
}

func (c *Coordinator) WalletManager() *wallet.Manager {
	return c.walletManager
}

func (c *Coordinator) ValidatorNames() *names.ValidatorNames {
	return c.validatorNames
}

func (c *Coordinator) GlobalVariables() types.Variables {
	return c.globalVars
}

func (c *Coordinator) LoadTests(ctx context.Context) {
	descriptors := test.LoadTestDescriptors(ctx, c.globalVars, c.Config.Tests, c.Config.ExternalTests)
	errCount := 0

	c.testDescriptorsMutex.Lock()
	defer c.testDescriptorsMutex.Unlock()

	indexMap := map[string]uint64{}

	for id, descriptorEntry := range c.testDescriptors {
		if !descriptorEntry.dynamic {
			delete(c.testDescriptors, id)

			indexMap[id] = descriptorEntry.index
		}
	}

	for _, descriptor := range descriptors {
		if descriptor.Err() != nil {
			c.log.GetLogger().Errorf("error while loading test '%v': %v", descriptor.ID(), descriptor.Err())

			errCount++
		} else {
			entryIndex := indexMap[descriptor.ID()]
			if entryIndex == 0 {
				c.testDescriptorIndex++
				entryIndex = c.testDescriptorIndex
			}

			c.testDescriptors[descriptor.ID()] = testDescriptorEntry{
				descriptor: descriptor,
				dynamic:    false,
				index:      entryIndex,
			}
		}
	}

	c.log.GetLogger().Infof("loaded %v test descriptors (%v errors)", len(descriptors), errCount)
}

func (c *Coordinator) AddTestDescriptor(testDescriptor types.TestDescriptor) error {
	if testDescriptor.Err() != nil {
		return fmt.Errorf("cannot add failed test descriptor: %v", testDescriptor.Err())
	}

	if testDescriptor.ID() == "" {
		return fmt.Errorf("cannot add test descriptor without ID")
	}

	c.testDescriptorsMutex.Lock()
	defer c.testDescriptorsMutex.Unlock()

	entryIndex := c.testDescriptors[testDescriptor.ID()].index
	if entryIndex == 0 {
		c.testDescriptorIndex++
		entryIndex = c.testDescriptorIndex
	}

	c.testDescriptors[testDescriptor.ID()] = testDescriptorEntry{
		descriptor: testDescriptor,
		dynamic:    true,
		index:      entryIndex,
	}

	return nil
}

func (c *Coordinator) GetTestDescriptors() []types.TestDescriptor {
	c.testDescriptorsMutex.RLock()
	defer c.testDescriptorsMutex.RUnlock()

	descriptors := make([]types.TestDescriptor, len(c.testDescriptors))
	idx := 0

	for _, descriptorEntry := range c.testDescriptors {
		descriptors[idx] = descriptorEntry.descriptor
		idx++
	}

	sort.Slice(descriptors, func(a, b int) bool {
		entryA := c.testDescriptors[descriptors[a].ID()]
		entryB := c.testDescriptors[descriptors[b].ID()]

		return entryA.index < entryB.index
	})

	return descriptors
}

func (c *Coordinator) GetTestByRunID(runID uint64) types.Test {
	c.testRegistryMutex.RLock()
	defer c.testRegistryMutex.RUnlock()

	return c.testRunMap[runID]
}

func (c *Coordinator) GetTestQueue() []types.Test {
	c.testRegistryMutex.RLock()
	defer c.testRegistryMutex.RUnlock()

	tests := make([]types.Test, len(c.testQueue))
	copy(tests, c.testQueue)

	return tests
}

func (c *Coordinator) GetTestHistory() []types.Test {
	c.testRegistryMutex.RLock()
	defer c.testRegistryMutex.RUnlock()

	tests := make([]types.Test, len(c.testHistory))
	copy(tests, c.testHistory)

	return tests
}

func (c *Coordinator) startMetrics() error {
	c.log.GetLogger().
		Info(fmt.Sprintf("Starting metrics server on :%v", c.metricsPort))

	http.Handle("/metrics", promhttp.Handler())

	//nolint:gosec // ignore
	err := http.ListenAndServe(fmt.Sprintf(":%v", c.metricsPort), nil)

	return err
}

func (c *Coordinator) ScheduleTest(descriptor types.TestDescriptor, configOverrides map[string]any, allowDuplicate bool) (types.Test, error) {
	if descriptor.Err() != nil {
		return nil, fmt.Errorf("cannot create test from failed test descriptor: %w", descriptor.Err())
	}

	testRef, err := c.createTestRun(descriptor, configOverrides, allowDuplicate)
	if err != nil {
		return nil, err
	}

	select {
	case c.testNotificationChan <- true:
	default:
	}

	return testRef, nil
}

func (c *Coordinator) createTestRun(descriptor types.TestDescriptor, configOverrides map[string]any, allowDuplicate bool) (types.Test, error) {
	c.testSchedulerMutex.Lock()
	defer c.testSchedulerMutex.Unlock()

	if !allowDuplicate {
		for _, queuedTest := range c.GetTestQueue() {
			if queuedTest.TestID() == descriptor.ID() {
				return nil, fmt.Errorf("test already in queue")
			}
		}
	}

	c.runIDCounter++
	runID := c.runIDCounter

	testRef, err := test.CreateTest(runID, descriptor, c.Logger().WithField("module", "test"), c)
	if err != nil {
		return nil, fmt.Errorf("failed initializing test run #%v '%v': %w", runID, descriptor.Config().Name, err)
	}

	if configOverrides != nil {
		testVars := testRef.GetTestVariables()
		for cfgKey, cfgValue := range configOverrides {
			testVars.SetVar(cfgKey, cfgValue)
		}
	}

	c.testRegistryMutex.Lock()
	c.testQueue = append(c.testQueue, testRef)
	c.testRunMap[runID] = testRef
	c.testRegistryMutex.Unlock()

	return testRef, nil
}

func (c *Coordinator) runTestExecutionLoop(ctx context.Context) {
	concurrencyLimit := c.Config.Coordinator.MaxConcurrentTests
	if concurrencyLimit < 1 {
		concurrencyLimit = 1
	}

	semaphore := make(chan bool, concurrencyLimit)

	for {
		var nextTest types.Test

		c.testRegistryMutex.Lock()
		if len(c.testQueue) > 0 {
			nextTest = c.testQueue[0]
			c.testQueue = c.testQueue[1:]
			c.testHistory = append(c.testHistory, nextTest)
		}
		c.testRegistryMutex.Unlock()

		if nextTest != nil {
			// run next test
			testFunc := func(nextTest types.Test) {
				defer func() { <-semaphore }()
				c.runTest(ctx, nextTest)
			}
			semaphore <- true

			go testFunc(nextTest)
		} else {
			// sleep and wait for queue notification
			select {
			case <-ctx.Done():
				return
			case <-c.testNotificationChan:
			case <-time.After(60 * time.Second):
			}
		}
	}
}

func (c *Coordinator) runTest(ctx context.Context, testRef types.Test) {
	c.lastExecutedRunID = testRef.RunID()

	if err := testRef.Validate(); err != nil {
		testRef.Logger().Errorf("test validation failed: %v", err)
		return
	}

	if err := testRef.Run(ctx); err != nil {
		testRef.Logger().Errorf("test execution failed: %v", err)
	}
}

func (c *Coordinator) runTestScheduler(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.log.GetLogger().WithError(err.(error)).Panicf("uncaught panic in coordinator.runTestScheduler: %v, stack: %v", err, string(debug.Stack()))
		}
	}()

	// startup scheduler
	for _, testDescr := range c.getStartupTests() {
		_, err := c.ScheduleTest(testDescr, nil, false)
		if err != nil {
			c.Logger().Errorf("could not schedule startup test execution for %v (%v): %v", testDescr.ID(), testDescr.Config().Name, err)
		}
	}

	// cron scheduler
	cronTime := time.Unix((time.Now().Unix()/60)*60, 0)

	for {
		cronTime = cronTime.Add(1 * time.Minute)
		cronTimeDiff := time.Since(cronTime)

		if cronTimeDiff < 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(cronTimeDiff.Abs()):
			}
		}

		for _, testDescr := range c.getCronTests(cronTime) {
			_, err := c.ScheduleTest(testDescr, nil, false)
			if err != nil {
				c.Logger().Errorf("could not schedule cron test execution for %v (%v): %v", testDescr.ID(), testDescr.Config().Name, err)
			}
		}
	}
}

func (c *Coordinator) getStartupTests() []types.TestDescriptor {
	descriptors := []types.TestDescriptor{}

	for _, testDescr := range c.GetTestDescriptors() {
		if testDescr.Err() != nil {
			continue
		}

		testConfig := testDescr.Config()
		if testConfig.Schedule == nil || testConfig.Schedule.Startup {
			descriptors = append(descriptors, testDescr)
		}
	}

	return descriptors
}

func (c *Coordinator) getCronTests(cronTime time.Time) []types.TestDescriptor {
	descriptors := []types.TestDescriptor{}

	for _, testDescr := range c.GetTestDescriptors() {
		if testDescr.Err() != nil {
			continue
		}

		testConfig := testDescr.Config()
		if testConfig.Schedule == nil || len(testConfig.Schedule.Cron) == 0 {
			continue
		}

		triggerTest := false

		for _, cronExprStr := range testConfig.Schedule.Cron {
			cronExpr, err := cronexpr.Parse(cronExprStr)
			if err != nil {
				c.Logger().Errorf("invalid cron expression for test %v (%v): %v", testDescr.ID(), testConfig.Name, err)
				break
			}

			next := cronExpr.Next(cronTime.Add(-1 * time.Second))
			if next.Compare(cronTime) == 0 {
				triggerTest = true
				break
			}
		}

		if !triggerTest {
			continue
		}

		descriptors = append(descriptors, testDescr)
	}

	return descriptors
}

func (c *Coordinator) runTestCleanup(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.log.GetLogger().WithError(err.(error)).Panicf("uncaught panic in coordinator.runTestCleanup: %v, stack: %v", err, string(debug.Stack()))
		}
	}()

	retentionTime := c.Config.Coordinator.TestRetentionTime.Duration
	if retentionTime <= 0 {
		retentionTime = 14 * 24 * time.Hour
	}

	cleanupInterval := 1 * time.Hour
	if retentionTime <= 4*time.Hour {
		cleanupInterval = 10 * time.Minute
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(cleanupInterval):
		}

		c.cleanupTestHistory(retentionTime)
	}
}

func (c *Coordinator) cleanupTestHistory(retentionTime time.Duration) {
	c.testRegistryMutex.Lock()
	defer c.testRegistryMutex.Unlock()

	cleanedHistory := []types.Test{}

	for _, test := range c.testHistory {
		if test.Status() != types.TestStatusPending && test.StartTime().Add(retentionTime).Compare(time.Now()) == -1 {
			test.Logger().Infof("cleanup test")
			continue
		}

		cleanedHistory = append(cleanedHistory, test)
	}

	c.testHistory = cleanedHistory
}
