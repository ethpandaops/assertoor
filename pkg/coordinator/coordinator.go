package coordinator

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/buildinfo"
	"github.com/erigontech/assertoor/pkg/coordinator/clients"
	"github.com/erigontech/assertoor/pkg/coordinator/clients/consensus"
	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/erigontech/assertoor/pkg/coordinator/logger"
	"github.com/erigontech/assertoor/pkg/coordinator/names"
	"github.com/erigontech/assertoor/pkg/coordinator/test"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/vars"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet"
	"github.com/erigontech/assertoor/pkg/coordinator/web"
	"github.com/jmoiron/sqlx"
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

	registry *TestRegistry
	runner   *TestRunner
}

func NewCoordinator(config *Config, log logrus.FieldLogger, metricsPort int) *Coordinator {
	return &Coordinator{
		log: logger.NewLogger(&logger.ScopeOptions{
			Parent:     log,
			BufferSize: 5000,
		}),
		Config:      config,
		metricsPort: metricsPort,
	}
}

// Run executes the coordinator until completion.
func (c *Coordinator) Run(ctx context.Context) error {
	defer func() {
		if err := recover(); err != nil {
			var err2 error
			if errval, errok := err.(error); errok {
				err2 = errval
			}

			c.log.GetLogger().WithError(err2).Errorf("uncaught panic in coordinator.Run: %v, stack: %v", err, string(debug.Stack()))
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

	defer func() {
		fmt.Println("Closing database")
		//nolint:errcheck // ignore error
		c.database.CloseDB()
	}()

	// load state from database
	lastTestRunID := uint64(0)
	//nolint:errcheck // ignore missing state
	c.database.GetAssertoorState("test.lastRunId", &lastTestRunID)

	err = c.database.RunTransaction(func(tx *sqlx.Tx) error {
		return c.database.CleanupUncleanTestRuns(tx)
	})
	if err != nil {
		return err
	}

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

	// init test registry
	c.registry = NewTestRegistry(c)
	c.registry.LoadTests(ctx, c.Config.Tests, c.Config.ExternalTests)

	// init test runner
	c.runner = NewTestRunner(c, lastTestRunID)

	// start test scheduler
	go c.runner.RunTestScheduler(ctx)

	// start test cleanup routine
	go c.runner.RunTestCleanup(ctx, c.Config.Coordinator.TestRetentionTime.Duration)

	// start per epoch GC routine
	go c.runEpochGC(ctx)

	// start off queue test execution loop
	go c.runner.RunOffQueueTestExecutionLoop(ctx)

	// run test execution loop for queued tests
	c.runner.RunTestExecutionLoop(ctx, c.Config.Coordinator.MaxConcurrentTests)

	return nil
}

func (c *Coordinator) Logger() logrus.FieldLogger {
	return c.log.GetLogger()
}

func (c *Coordinator) LogReader() logger.LogReader {
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

func (c *Coordinator) TestRegistry() types.TestRegistry {
	return c.registry
}

func (c *Coordinator) GetTestByRunID(runID uint64) types.Test {
	testRef := c.runner.GetTestByRunID(runID)
	if testRef != nil {
		return testRef
	}

	if runID > math.MaxInt {
		return nil
	}

	testRef, err := test.LoadTestFromDB(c.database, runID)
	if err != nil {
		return nil
	}

	return testRef
}

func (c *Coordinator) GetTestQueue() []types.Test {
	return c.runner.GetTestQueue()
}

func (c *Coordinator) GetTestHistory(testID string, firstRunID, offset, limit uint64) (tests []types.Test, totalTests uint64) {
	dbTests, totalTests, err := c.database.GetTestRunRange(testID, firstRunID, offset, limit)
	if err != nil {
		return nil, 0
	}

	tests = make([]types.Test, len(dbTests))

	for idx, dbTest := range dbTests {
		if testRef := c.runner.GetTestByRunID(dbTest.RunID); testRef != nil {
			tests[idx] = testRef
		} else {
			tests[idx] = test.WrapDBTestRun(c.database, dbTest)
		}
	}

	return tests, totalTests
}

func (c *Coordinator) DeleteTestRun(runID uint64) error {
	testRef := c.runner.GetTestByRunID(runID)
	if testRef != nil {
		if testRef.Status() != types.TestStatusPending {
			return errors.New("cannot delete running test")
		}

		if !c.runner.RemoveTestFromQueue(runID) {
			return errors.New("could not remove test from queue")
		}
	}

	err := c.database.RunTransaction(func(tx *sqlx.Tx) error {
		return c.database.DeleteTestRun(tx, runID)
	})

	return err
}

func (c *Coordinator) ScheduleTest(descriptor types.TestDescriptor, configOverrides map[string]any, allowDuplicate, skipQueue bool) (types.TestRunner, error) {
	return c.runner.ScheduleTest(descriptor, configOverrides, allowDuplicate, skipQueue)
}

func (c *Coordinator) startMetrics() error {
	c.log.GetLogger().
		Info(fmt.Sprintf("Starting metrics server on :%v", c.metricsPort))

	http.Handle("/metrics", promhttp.Handler())

	//nolint:gosec // ignore
	err := http.ListenAndServe(fmt.Sprintf(":%v", c.metricsPort), nil)

	return err
}

func (c *Coordinator) runEpochGC(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			var err2 error
			if errval, errok := err.(error); errok {
				err2 = errval
			}

			c.log.GetLogger().WithError(err2).Panicf("uncaught panic in coordinator.runEpochGC: %v, stack: %v", err, string(debug.Stack()))
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
			currentSlot := uint64(networkTime / specs.SecondsPerSlot) //nolint:gosec // no overflow possible
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

				nextEpochDuration := time.Until(genesis.GenesisTime.Add(time.Duration((currentEpoch+1)*specs.SlotsPerEpoch) * specs.SecondsPerSlot)) //nolint:gosec // no overflow possible

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

			nextRunTime := genesis.GenesisTime.Add(time.Duration(nextGcSlot) * specs.SecondsPerSlot) //nolint:gosec // no overflow possible
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
