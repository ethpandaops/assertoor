package cmd

import (
	"os"

	"github.com/ethpandaops/assertoor/pkg/assertoor"
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
		config, err := assertoor.NewConfig(cfgFile)
		if err != nil {
			logrus.WithError(err).Error("failed to load configuration")
			os.Exit(1)
		}

		// Validate configuration
		if err := config.Validate(); err != nil {
			logrus.WithError(err).Error("configuration validation failed")
			os.Exit(1)
		}

		// Validate external test files exist and can be parsed
		for _, extTest := range config.ExternalTests {
			if extTest.File != "" {
				// Check if external test file exists
				if _, err := os.Stat(extTest.File); os.IsNotExist(err) {
					logrus.WithField("test", extTest.File).Error("external test file does not exist")
					os.Exit(1)
				}
			}
		}

		// Success - exit cleanly with no output
		os.Exit(0)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
