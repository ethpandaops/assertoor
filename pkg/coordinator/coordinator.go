package coordinator

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Coordinator struct {
	// Config is the coordinator configuration.
	Config          *Config
	log             *logger.LogScope
	clientPool      *clients.ClientPool
	walletManager   *wallet.Manager
	webserver       *server.WebServer
	validatorNames  *names.ValidatorNames
	tests           []types.Test
	metricsPort     int
	lameDuckSeconds int
}

func NewCoordinator(config *Config, log logrus.FieldLogger, metricsPort, lameDuckSeconds int) *Coordinator {
	return &Coordinator{
		log: logger.NewLogger(&logger.ScopeOptions{
			Parent:      log,
			HistorySize: 5000,
		}),
		Config:          config,
		tests:           []types.Test{},
		metricsPort:     metricsPort,
		lameDuckSeconds: lameDuckSeconds,
	}
}

// Run executes the coordinator until completion.
func (c *Coordinator) Run(ctx context.Context) error {
	defer func() {
		if err := recover(); err != nil {
			logrus.WithError(err.(error)).Errorf("uncaught panic coordinator: %v, stack: %v", err, string(debug.Stack()))
		}
	}()

	c.log.GetLogger().
		WithField("build_version", buildinfo.GetVersion()).
		WithField("metrics_port", c.metricsPort).
		WithField("lame_duck_seconds", c.lameDuckSeconds).
		Info("starting coordinator")

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

	// init webserver
	if c.Config.Web != nil && c.Config.Web.Server != nil {
		c.webserver, err = server.NewWebServer(c.Config.Web.Server, c.log.GetLogger())
		if err != nil {
			return err
		}

		if c.Config.Web.Frontend != nil {
			err = c.webserver.StartFrontend(c.Config.Web.Frontend, c)
			if err != nil {
				return err
			}
		}
	}

	//nolint:errcheck // ignore
	go c.startMetrics()

	// load global variables
	variables := c.NewVariables(nil)
	for name, value := range c.Config.GlobalVars {
		variables.SetVar(name, value)
	}

	// load validator names
	c.validatorNames = names.NewValidatorNames(c.Config.ValidatorNames, c.log.GetLogger())
	c.validatorNames.LoadValidatorNames()

	// load tests
	err = c.loadTests(ctx, variables)
	if err != nil {
		return err
	}

	c.log.GetLogger().Infof("Loaded %v tests", len(c.tests))

	// run tests
	c.runTests(ctx)

	if c.webserver == nil {
		c.log.GetLogger().WithField("seconds", c.lameDuckSeconds).Info("Initiating lame duck")
		time.Sleep(time.Duration(c.lameDuckSeconds) * time.Second)
		c.log.GetLogger().Info("lame duck complete")
	} else {
		<-ctx.Done()
	}

	c.log.GetLogger().Info("Shutting down..")

	return nil
}

func (c *Coordinator) Logger() logrus.FieldLogger {
	return c.log.GetLogger()
}

func (c *Coordinator) LogScope() *logger.LogScope {
	return c.log
}

func (c *Coordinator) NewVariables(parentScope types.Variables) types.Variables {
	return vars.NewVariables(parentScope)
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

func (c *Coordinator) GetTests() []types.Test {
	tests := make([]types.Test, len(c.tests))
	copy(tests, c.tests)

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

func (c *Coordinator) loadTests(ctx context.Context, globalVars types.Variables) error {
	// load configured tests
	for _, testCfg := range c.Config.Tests {
		testRef, err := test.CreateTest(c, testCfg, globalVars)
		if err != nil {
			return fmt.Errorf("failed initializing test '%v': %w", testCfg.Name, err)
		}

		c.tests = append(c.tests, testRef)
	}

	// load external tests
	for _, extTestCfg := range c.Config.ExternalTests {
		testConfig, err := c.loadExternalTestConfig(ctx, extTestCfg)
		if err != nil {
			return err
		}

		if extTestCfg.Name != "" {
			testConfig.Name = extTestCfg.Name
		}

		if extTestCfg.Timeout != nil {
			testConfig.Timeout = *extTestCfg.Timeout
		}

		for k, v := range extTestCfg.Config {
			testConfig.Config[k] = v
		}

		for k, v := range extTestCfg.ConfigVars {
			testConfig.ConfigVars[k] = v
		}

		testRef, err := test.CreateTest(c, testConfig, globalVars)
		if err != nil {
			return fmt.Errorf("failed initializing external test '%v': %w", testConfig.Name, err)
		}

		c.tests = append(c.tests, testRef)
	}

	return nil
}

func (c *Coordinator) loadExternalTestConfig(ctx context.Context, extTestCfg *test.ExternalConfig) (*test.Config, error) {
	var reader io.Reader

	if strings.HasPrefix(extTestCfg.File, "http://") || strings.HasPrefix(extTestCfg.File, "https://") {
		client := &http.Client{Timeout: time.Second * 120}

		req, err := http.NewRequestWithContext(ctx, "GET", extTestCfg.File, http.NoBody)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error loading test config from url: %v, result: %v %v", extTestCfg.File, resp.StatusCode, resp.Status)
		}

		reader = resp.Body
	} else {
		f, err := os.Open(extTestCfg.File)
		if err != nil {
			return nil, fmt.Errorf("error loading test config from file %v: %w", extTestCfg.File, err)
		}

		defer f.Close()

		reader = f
	}

	decoder := yaml.NewDecoder(reader)
	testConfig := &test.Config{}

	err := decoder.Decode(testConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding external test config %v: %v", extTestCfg.File, err)
	}

	return testConfig, nil
}

func (c *Coordinator) runTests(ctx context.Context) {
	// run tests
	for _, testRef := range c.tests {
		if err := testRef.Validate(); err != nil {
			testRef.Logger().Errorf("test validation failed: %v", err)
			continue
		}

		if err := testRef.Run(ctx); err != nil {
			testRef.Logger().Errorf("test execution failed: %v", err)
			continue
		}
	}
}
