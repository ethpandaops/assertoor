/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/ethpandaops/minccino/pkg/coordinator/task/tasks"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// tasksCmd represents the tasks command
var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Lists all available tasks",
	Run: func(cmd *cobra.Command, args []string) {
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
