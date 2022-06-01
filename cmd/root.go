package cmd

import (
	"log"
	"os"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sync-test-coordinator",
	Short: "Runs a configured test until completion or error",
	Run: func(cmd *cobra.Command, args []string) {
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

		coord := coordinator.NewCoordinator(config, logr, metricsPort, lameDuckSeconds)

		if err := coord.Run(cmd.Context()); err != nil {
			log.Fatal(err)
		}

	},
}

var (
	cfgFile         string
	logFormat       string
	metricsPort     int
	lameDuckSeconds int
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ethereum-metrics-exporter.yaml)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "log-format", "json", "log format (default is text). Valid values are 'text', 'json'")

	rootCmd.Flags().IntVarP(&metricsPort, "metrics-port", "", 9090, "Port to serve Prometheus metrics on")
	rootCmd.Flags().IntVarP(&lameDuckSeconds, "lame-duck-seconds", "", 30, "Lame duck period in seconds (wait for this long after completion before terminating")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
