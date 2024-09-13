package coordinator

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/db"
	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/test"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web"
	"github.com/gorhill/cronexpr"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

type Coordinator struct {
	// Config is the coordinator configuration.
	Config          *Config
	log             *logger.LogScope
	database        *db.Database
	clientPool      *clients.ClientPool
	walletManager   *wallet.Manager
	webserver       *web.Server
	publicWebserver *web.Server
	validatorNames  *names.ValidatorNames
	globalVars      types.Variables
	metricsPort     int

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

	// self test BLS key generation
	// this seems broken for some CPU types, test and emit a warning here for debugging
	if err := c.testBlsMath(); err != nil {
		c.log.GetLogger().Warnf("BLS key generation self test failed: %v", err)
	}

	// init database
	database := db.NewDatabase(c.log.GetLogger())

	if c.Config.Database == nil {
		// use default in-memory database
		c.Config.Database = &db.DatabaseConfig{
			Engine: "sqlite",
			Sqlite: &db.SqliteDatabaseConfig{
				File: ":memory:?cache=shared",
			},
		}
	}

	err := database.InitDB(c.Config.Database)
	if err != nil {
		return err
	}

	err = database.ApplySchema(-2)
	if err != nil {
		return err
	}

	c.database = database

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
	if c.Config.Web != nil {
		if c.Config.Web.Server != nil {
			c.webserver, err = web.NewWebServer(c.Config.Web.Server, c.log.GetLogger())
			if err != nil {
				return err
			}

			err = c.webserver.ConfigureRoutes(c.Config.Web.Frontend, c.Config.Web.API, c, false)
			if err != nil {
				return err
			}
		}

		if c.Config.Web.PublicServer != nil {
			c.publicWebserver, err = web.NewWebServer(c.Config.Web.PublicServer, c.log.GetLogger().WithField("module", "public_web"))
			if err != nil {
				return err
			}

			err = c.publicWebserver.ConfigureRoutes(c.Config.Web.Frontend, nil, c, true)
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

	// start per epoch GC routine
	go c.runEpochGC(ctx)

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

func (c *Coordinator) Database() *db.Database {
	return c.database
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

func (c *Coordinator) runEpochGC(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.log.GetLogger().WithError(err.(error)).Panicf("uncaught panic in coordinator.runEpochGC: %v, stack: %v", err, string(debug.Stack()))
		}
	}()

	// await client readiness, which implies cache initialization
	if c.clientPool.GetConsensusPool().AwaitReadyEndpoint(ctx, consensus.AnyClient) == nil {
		return
	}

	genesis := c.clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	specs := c.clientPool.GetConsensusPool().GetBlockCache().GetSpecs()

	for {
		var sleepTime time.Duration

		networkTime := time.Since(genesis.GenesisTime)
		if networkTime < 0 {
			sleepTime = networkTime.Abs()
		} else {
			currentSlot := uint64(networkTime / specs.SecondsPerSlot)
			currentEpoch := currentSlot / specs.SlotsPerEpoch
			currentSlotIndex := currentSlot % specs.SlotsPerEpoch
			nextGcSlot := uint64(0)

			gcSlotDiff := uint64(2)
			if gcSlotDiff > specs.SlotsPerEpoch/2 {
				gcSlotDiff = 1
			}

			gcSlotIndex := specs.SlotsPerEpoch - gcSlotDiff - 1

			if currentSlotIndex == gcSlotIndex {
				select {
				case <-ctx.Done():
					return
				case <-time.After(specs.SecondsPerSlot / 2):
				}

				nextEpochDuration := time.Until(genesis.GenesisTime.Add(time.Duration((currentEpoch+1)*specs.SlotsPerEpoch) * specs.SecondsPerSlot))

				c.log.GetLogger().Infof("run GC (slot %v, %v sec before epoch %v)", currentSlot, nextEpochDuration.Seconds(), currentEpoch+1)
				runtime.GC()

				nextGcSlot = currentSlot + specs.SlotsPerEpoch
			} else {
				if currentSlotIndex < gcSlotIndex {
					nextGcSlot = currentSlot + (gcSlotIndex - currentSlotIndex)
				} else {
					nextGcSlot = currentSlot + (specs.SlotsPerEpoch - currentSlotIndex) + gcSlotIndex
				}
			}

			nextRunTime := genesis.GenesisTime.Add(time.Duration(nextGcSlot) * specs.SecondsPerSlot)
			sleepTime = time.Until(nextRunTime)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(sleepTime):
		}
	}
}

// testBlsMath tests the BLS math
// this seems broken for some CPU types
func (c *Coordinator) testBlsMath() error {
	mnemonic := "trigger mouse legal obey solve noble light employ shrug length kiwi make neutral friend divide like fortune outside trim install ocean gap token honey"
	pubkeys := []string{
		"0xb3c59dd04900cdcd10be94e31a9bf302ad9a323a1bb3fb710c44e7f5b7acd4ce35a590de88a640dce9b8dff3fc188a39",
		"0xa2caa2dc8b2295fe6ff78815cbe42a5103c668fb3a4e796a56d40145a192a2ce7e2be0d38cda931b6373e5c96d0f8a50",
		"0x8bfcfd33fda4385788b9c028f8025c35488b5187dfcd3901ac498a3ef0a6dbd0076e4d1f7b028c863e7e684713ff2521",
		"0x89e5207d07509abe027003c9adcc88649d072620e2583b212b2c1284d0a14aaf072b72e463997b39ca60bddeffa04896",
		"0x89d6cf68072d6a93aab7b4101d2c38cb514c2971460dd1430ae4a900969491b9927b5368d0c967bc0dc23b25798cde4c",
		"0x9861ce59afa1623bccee64b0caa6a195bb48ff60c932a0b355cc08f3ee6c3ab2a2dda4d7da04fe8c6ff07cfcf06c3081",
		"0xa9d49d74114bba6059528831ff053a9024e11f98a3eb3c5c73607ad78d7ca2d7379424a9ff0cc020523fdfed6cb3e6c3",
		"0x8f47915127f9b9692812f8bcd9b630b8766c9baac92540f222745fb2112e4c4ad5d72fd5462f5724a0faa550ba795f23",
		"0x97e8fafd9233f72a3da52805311bece0f405b64e80d0c831b5c8a2bb978b0a026f460e66789338a3e8057db6f5eece68",
		"0x8f0eb2ed68556fd0ee84c0b1fcf107617f88f4a9a390963b3b8a242a64ffcf481cd9424e74e6d2bb05caf40d3c16774f",
	}

	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return errors.New("mnemonic is not valid")
	}

	seed := bip39.NewSeed(mnemonic, "")

	for accountIdx := 0; accountIdx < 10; accountIdx++ {
		validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

		validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(seed, validatorKeyPath)
		if err != nil {
			return fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
		}

		validatorPubkey := validatorPrivkey.PublicKey().Marshal()
		validatorPubkeyStr := fmt.Sprintf("0x%x", validatorPubkey)

		if validatorPubkeyStr != pubkeys[accountIdx] {
			return fmt.Errorf("validator pubkey %v mismatch: %v != %v", accountIdx, validatorPubkeyStr, pubkeys[accountIdx])
		}
	}

	return nil
}
