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
		logr.SetFormatter(&logrus.JSONFormatter{})

		coord := coordinator.NewCoordinator(config, logr)

		if err := coord.Run(cmd.Context()); err != nil {
			log.Fatal(err)
		}

	},
}

var (
	cfgFile string
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ethereum-metrics-exporter.yaml)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
