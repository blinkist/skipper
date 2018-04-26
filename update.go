package main

import (
	//"bytes"

	"strings"

	"github.com/blinkist/skipper/aws/ecsclient"
	"github.com/blinkist/skipper/helpers"
	"github.com/spf13/cobra"
)

var (
	argImageOverride          string
	argTaskOverride           string
	argTaskdefinitionOverride string
	argImageTag               string
	argWait                   bool
	argSets                   []string
	argUnsets                 []string
	argTargetGroup            string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update services",
	Long:  "View current cluster or service deployment status",
	Run: func(cmd *cobra.Command, args []string) {
		ecs := ecsclient.New()
		cluster, service := helpers.ServicePicker(ecs, args)

		changes := make(map[string]string)
		for _, change := range argSets {
			parts := strings.SplitN(change, "=", 2)

			var value string
			if len(parts) > 1 {
				value = parts[1]
			}

			changes[parts[0]] = value
		}

		removes := make(map[string]struct{})
		for _, field := range argUnsets {
			removes[field] = struct{}{}
		}
		ecs.UpdateService(&cluster, &service, nil, nil, true, &changes, &removes, &argTargetGroup)
	},
}

func init() {
	RootCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringVarP(&argImageOverride, "image_override", "i", "", "Override the image with our own image_url e.g. 1234.dkr.ecr.abc.amazonaws.com/image:latest")
	updateCmd.Flags().StringVarP(&argService, "service", "", "", "The name of the service")
	updateCmd.Flags().StringVarP(&argServiceType, "service_type", "", "web", "The name of the service")
	updateCmd.Flags().StringVarP(&argTaskdefinitionOverride, "taskdefinition_override", "", "", "The name of the task definition, this overrides the default service definition.")
	updateCmd.Flags().StringVarP(&argImageTag, "image_tag", "", "", "The image tag")
	updateCmd.Flags().StringVarP(&argTargetGroup, "targetGroup", "", "", "The placement targetGroup to use")
	updateCmd.Flags().StringArrayVar(&argSets, "set", nil, "key=value to be updated (can be used multiple times)")
	updateCmd.Flags().StringArrayVar(&argUnsets, "unset", nil, "key to be removed (can be used multiple times)")
}
