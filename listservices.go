package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/blinkist/skipper/aws/ecsclient"
)

func init() {
	RootCmd.AddCommand(servicesListCmd)

}

func listServices() {
	ecs := ecsclient.New()
	clusters, err := ecs.GetClusterNames()
	if err != nil {
		panic(fmt.Sprintf("unhandled error: %v", err))
	}
	sort.Strings(clusters)
	if len(clusters) < 0 {
		fmt.Printf("no clusters found\n")
		os.Exit(0)
	}

	for _, cluster := range clusters {
		services, _ := ecs.ListServices(&cluster)
		if len(services) > 0 {
			fmt.Printf("Cluster:\t%s\n", cluster)

			sort.Strings(services)
			for _, disp := range services {
				fmt.Printf("[ - ]: %s\n", disp)
			}
		}
	}
}

var servicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "View current the different clusters and services",
	Long:  "View current cluster or service deployment status",
	Run: func(cmd *cobra.Command, args []string) {
		listServices()
	},
}
