/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/ethpandaops/assertoor/pkg/coordinator/tasks"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// tasksCmd represents the tasks command
var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Lists all available tasks",
	Run: func(_ *cobra.Command, _ []string) {
		available := tasks.AvailableTasks()

		yamlData, err := yaml.Marshal(&available)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(yamlData))
	},
}

func init() {
	rootCmd.AddCommand(tasksCmd)
}
