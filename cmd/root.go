package cmd

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/coordinator"
	"github.com/ethpandaops/assertoor/pkg/coordinator/buildinfo"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "assertoor",
	Short: "Runs a configured test until completion or error",
	Run: func(cmd *cobra.Command, _ []string) {
		logr := logrus.New()

		if version {
			logr.Printf("Version: %s\n", buildinfo.GetVersion())
			return
		}

		config, err := coordinator.NewConfig(cfgFile)
		if err != nil {
			logr.Fatal(err)
		}

		if maxConcurrentTests > 0 {
			config.Coordinator.MaxConcurrentTests = maxConcurrentTests
		}

		switch logFormat {
		case "json":
			logr.SetFormatter(&logrus.JSONFormatter{})
			logr.Info("Log format set to json")
		case "text":
			logr.SetFormatter(&logrus.TextFormatter{})
			logr.Info("Log format set to text")
		default:
			logr.Fatalf("Invalid log format: %s", logFormat)
		}
		if verbose {
			logr.SetLevel(logrus.DebugLevel)
		}

		coord := coordinator.NewCoordinator(config, logr, metricsPort)

		if err := coord.Run(cmd.Context()); err != nil {
			logr.Fatal(err)
		}

	},
}

var (
	cfgFile            string
	logFormat          string
	verbose            bool
	metricsPort        int
	maxConcurrentTests uint64
	version            bool
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (default is text). Valid values are 'text', 'json'")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().Uint64Var(&maxConcurrentTests, "maxConcurrentTests", 0, "Number of tests to run concurrently")
	rootCmd.Flags().IntVarP(&metricsPort, "metrics-port", "", 9090, "Port to serve Prometheus metrics on")
	rootCmd.Flags().BoolVarP(&version, "version", "", false, "Print version information")
}
