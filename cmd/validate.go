package cmd

import (
	"context"
	"os"

	"github.com/ethpandaops/assertoor/pkg/coordinator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate assertoor configuration",
	Long:  `Validates the assertoor configuration file for syntax and semantic correctness`,
	Args:  cobra.NoArgs,
	Run: func(_ *cobra.Command, _ []string) {
		// Set up minimal logging for error output
		logrus.SetLevel(logrus.ErrorLevel)
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}

		// Check if config file is specified
		if cfgFile == "" {
			logrus.Error("no configuration file specified")
			os.Exit(1)
		}

		// Load configuration
		config, err := coordinator.NewConfig(cfgFile)
		if err != nil {
			logrus.WithError(err).Error("failed to load configuration")
			os.Exit(1)
		}

		// Validate configuration
		if err := config.Validate(); err != nil {
			logrus.WithError(err).Error("configuration validation failed")
			os.Exit(1)
		}

		// Create a minimal coordinator instance just for test loading validation
		// We don't need to run the full coordinator, just validate external tests can be loaded
		coord := coordinator.NewCoordinator(config, logrus.StandardLogger(), metricsPort)
		testRegistry := coordinator.NewTestRegistry(coord)

		// Validate external tests can be loaded
		ctx := context.Background()
		for _, extTest := range config.ExternalTests {
			_, err := testRegistry.AddExternalTest(ctx, extTest)
			if err != nil {
				logrus.WithError(err).WithField("test", extTest.File).Error("failed to load external test")
				os.Exit(1)
			}
		}

		// Success - exit cleanly with no output
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
