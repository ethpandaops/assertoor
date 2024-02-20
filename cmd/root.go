package cmd

import (
	"log"
	"os"

	"github.com/ethpandaops/assertoor/pkg/coordinator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "assertoor",
	Short: "Runs a configured test until completion or error",
	Run: func(cmd *cobra.Command, _ []string) {
		config, err := coordinator.NewConfig(cfgFile)
		if err != nil {
			log.Fatal(err)
		}

		logr := logrus.New()
		if logFormat == "json" {
			logr.SetFormatter(&logrus.JSONFormatter{})
			logr.Info("Log format set to json")
		} else if logFormat == "text" {
			logr.SetFormatter(&logrus.TextFormatter{})
			logr.Info("Log format set to text")
		}
		if verbose {
			logr.SetLevel(logrus.DebugLevel)
		}

		coord := coordinator.NewCoordinator(config, logr, metricsPort)

		if err := coord.Run(cmd.Context()); err != nil {
			log.Fatal(err)
		}

	},
}

var (
	cfgFile     string
	logFormat   string
	verbose     bool
	metricsPort int
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (default is text). Valid values are 'text', 'json'")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	rootCmd.Flags().IntVarP(&metricsPort, "metrics-port", "", 9090, "Port to serve Prometheus metrics on")
}
