package coordinator

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/test"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
	"github.com/ethpandaops/assertoor/pkg/coordinator/web/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Coordinator struct {
	// Config is the coordinator configuration.
	Config          *Config
	log             logrus.FieldLogger
	clientPool      *clients.ClientPool
	webserver       *server.WebServer
	validatorNames  *names.ValidatorNames
	tests           []types.Test
	metricsPort     int
	lameDuckSeconds int
}

func NewCoordinator(config *Config, log logrus.FieldLogger, metricsPort, lameDuckSeconds int) *Coordinator {
	return &Coordinator{
		log:             log,
		Config:          config,
		tests:           []types.Test{},
		metricsPort:     metricsPort,
		lameDuckSeconds: lameDuckSeconds,
	}
}

// Run executes the coordinator until completion.
func (c *Coordinator) Run(ctx context.Context) error {
	c.log.
		WithField("build_version", buildinfo.GetVersion()).
		WithField("metrics_port", c.metricsPort).
		WithField("lame_duck_seconds", c.lameDuckSeconds).
		Info("starting coordinator")

	// init client pool
	clientPool, err := clients.NewClientPool()
	if err != nil {
		return err
	}

	c.clientPool = clientPool

	for idx := range c.Config.Endpoints {
		err = clientPool.AddClient(&c.Config.Endpoints[idx])
		if err != nil {
			return err
		}
	}

	// init webserver
	if c.Config.Web != nil && c.Config.Web.Server != nil {
		c.webserver, err = server.NewWebServer(c.Config.Web.Server, c.log)
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

	// initialize tests
	for _, testCfg := range c.Config.Tests {
		testRef, err := test.CreateTest(c, testCfg, variables)
		if err != nil {
			return fmt.Errorf("failed initializing test '%v': %w", testCfg.Name, err)
		}

		c.tests = append(c.tests, testRef)
	}

	// load validator names
	c.validatorNames = names.NewValidatorNames(c.Config.ValidatorNames)
	c.validatorNames.LoadValidatorNames()

	// run tests
	c.runTests(ctx)

	if c.webserver == nil {
		c.log.WithField("seconds", c.lameDuckSeconds).Info("Initiating lame duck")
		time.Sleep(time.Duration(c.lameDuckSeconds) * time.Second)
		c.log.Info("lame duck complete")
	} else {
		<-ctx.Done()
	}

	c.log.Info("Shutting down..")

	return nil
}

func (c *Coordinator) Logger() logrus.FieldLogger {
	return c.log
}

func (c *Coordinator) NewVariables(parentScope types.Variables) types.Variables {
	return vars.NewVariables(parentScope)
}

func (c *Coordinator) ClientPool() *clients.ClientPool {
	return c.clientPool
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
	c.log.
		Info(fmt.Sprintf("Starting metrics server on :%v", c.metricsPort))

	http.Handle("/metrics", promhttp.Handler())

	//nolint:gosec // ignore
	err := http.ListenAndServe(fmt.Sprintf(":%v", c.metricsPort), nil)

	return err
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
