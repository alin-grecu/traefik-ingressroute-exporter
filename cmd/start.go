/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/alin-grecu/traefik-ingressroute-exporter/pkg/start"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the server.",
	Run: func(cmd *cobra.Command, args []string) {
		start.Main()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
