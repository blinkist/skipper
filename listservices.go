package main

import "github.com/spf13/cobra"
import "github.com/blinkist/skipper/aws/ecsclient"
import "fmt"
import "sort"

func listServices() {
	ecs := ecsclient.New()
	clusters, _ := ecs.GetClusterNames()
	sort.Strings(clusters)

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

func init() {
	RootCmd.AddCommand(servicesListCmd)
}
