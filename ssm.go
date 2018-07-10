package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/blinkist/skipper/aws/ecsclient"
	"github.com/blinkist/skipper/aws/ssmclient"
	"github.com/blinkist/skipper/helpers"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func listSSM(service *string) {
	ssm := ssmclient.New()
	global := "global"
	listParams, _ := ssm.GetParameters(&global)
	green := color.New(color.FgGreen).SprintFunc()
	whitebold := color.New(color.FgWhite, color.Bold).SprintFunc()
	fmt.Printf("%s %s %s\n", "===", whitebold(global), whitebold("Config Vars"))
	for v := range listParams {
		key := strings.Replace(*listParams[v].Key, fmt.Sprintf("/application/%s/", global), "", -1)
		fmt.Printf("%-30s%s \t%s\n", green(key), ":", whitebold(*listParams[v].Value))
	}

	listParams, _ = ssm.GetParameters(service)

	fmt.Printf("%s %s %s\n", "===", whitebold(*service), whitebold("Config Vars"))
	for v := range listParams {
		key := strings.Replace(*listParams[v].Key, fmt.Sprintf("/application/%s/", *service), "", -1)
		fmt.Printf("%-30s%s \t%s\n", green(key), ":", whitebold(*listParams[v].Value))
	}

}

var ssmindexCmd = &cobra.Command{
	Use:   "ssm",
	Short: "SSM commands",
}

var ssmListCmd = &cobra.Command{
	Use:   "list",
	Short: "Modify configuration parameters",
	Long:  "Modify configuration parameters",
	Run: func(cmd *cobra.Command, args []string) {

		ecs := ecsclient.New()
		cluster, service := helpers.ServicePicker(ecs, args)
		stripped := strings.Replace(service, cluster+"-", "", -1)
		stripped = strings.Replace(stripped, "-web", "", -1)
		stripped = strings.Replace(stripped, "-worker", "", -1)
		listSSM(&stripped)
	},
}

var ssmDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Put one SSM parameter",
	Long:  "Put one SSM parameter",
	Run: func(cmd *cobra.Command, args []string) {

		ecs := ecsclient.New()
		cluster, service := helpers.ServicePicker(ecs, args)
		stripped := strings.Replace(service, cluster+"-", "", -1)
		stripped = strings.Replace(stripped, "-web", "", -1)
		stripped = strings.Replace(stripped, "-worker", "", -1)
		fmt.Println("")
		fmt.Println("Please enter the name of the parameter you want to delete. ")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name   : ")
		name, _ := reader.ReadString('\n')
		fmt.Printf("Are you sure you want to delete %s : ", strings.TrimRight(name, "\n"))
		ack, _ := reader.ReadString('\n')

		if strings.TrimRight(ack, "\n") != "yes" {
			fmt.Println("Not deleting, exitting")
			os.Exit(1)

		}
		fmt.Println("Deleting the key")

		ssm := ssmclient.New()

		err := ssm.DeleteParameter(&stripped, &name)
		if err != nil {
			panic(fmt.Sprintf("Error deleting: %s\n", err.Error()))
		}

	},
}

var ssmHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "History of one SSM parameter",
	Long:  "History of one SSM parameter",
	Run: func(cmd *cobra.Command, args []string) {
		ecs := ecsclient.New()
		cluster, service := helpers.ServicePicker(ecs, args)
		stripped := strings.Replace(service, cluster+"-", "", -1)
		stripped = strings.Replace(stripped, "-web", "", -1)
		stripped = strings.Replace(stripped, "-worker", "", -1)

		fmt.Println("")
		fmt.Println("Please enter the name of the configuration parameter ")
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name   : ")
		nameUntrimmed, _ := reader.ReadString('\n')
		name := strings.TrimRight(nameUntrimmed, "\n")

		ssm := ssmclient.New()
		green := color.New(color.FgGreen).SprintFunc()
		whitebold := color.New(color.FgWhite, color.Bold).SprintFunc()

		listParams, _ := ssm.GetParameterHistory(&stripped, &name)

		sort.Slice(listParams, func(i, j int) bool {
			return *listParams[i].Version > *listParams[j].Version
		})

		for v := range listParams {
			key := strings.Replace(*listParams[v].Key, fmt.Sprintf("/application/%s/", stripped), "", -1)
			fmt.Println("-------------------------------------------------------------------------------------")
			fmt.Printf("%-30s%s \t%s\n", green("VERSION"), ":", whitebold(*listParams[v].Version))
			fmt.Printf("%-30s%s \t%s\n", green("LastModifiedDate"), ":", whitebold(*listParams[v].LastModifiedDate))

			fmt.Printf("%-30s%s \t%s\n", green(key), ":", whitebold(*listParams[v].Value))
			fmt.Printf("%-30s%s \t%s\n", green("User"), ":", whitebold(*listParams[v].LastModifiedUser))
			fmt.Println("-------------------------------------------------------------------------------------")

		}
	},
}

var ssmPutCmd = &cobra.Command{
	Use:   "put",
	Short: "Put one SSM parameter",
	Long:  "Put one SSM parameter",
	Run: func(cmd *cobra.Command, args []string) {

		ecs := ecsclient.New()
		cluster, service := helpers.ServicePicker(ecs, args)
		stripped := strings.Replace(service, cluster+"-", "", -1)
		stripped = strings.Replace(stripped, "-web", "", -1)
		stripped = strings.Replace(stripped, "-worker", "", -1)
		fmt.Println("")
		fmt.Println("Please enter the name and the value of the configuration parameter ")
		fmt.Println("that you want to enter. Do not prepend /application/ etc. etc.")
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Name   : ")
		nameUntrimmed, _ := reader.ReadString('\n')
		name := strings.TrimRight(nameUntrimmed, "\n")
		fmt.Print("Value : ")
		valueUntrimmed, _ := reader.ReadString('\n')
		value := strings.TrimRight(valueUntrimmed, "\n")

		ssm := ssmclient.New()

		err := ssm.PutParameter(&stripped, &name, &value)
		if err != nil {
			panic(fmt.Sprintf("Error deleting: %s\n", err.Error()))
		}
	},
}

func init() {
	RootCmd.AddCommand(ssmindexCmd)
	ssmindexCmd.AddCommand(ssmListCmd)
	ssmindexCmd.AddCommand(ssmPutCmd)
	ssmindexCmd.AddCommand(ssmDeleteCmd)
	ssmindexCmd.AddCommand(ssmHistoryCmd)
}
