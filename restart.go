package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/blinkist/skipper/aws/ecsclient"
	"github.com/blinkist/skipper/helpers"
	"github.com/spf13/cobra"
)

var (
	rotatingkillFlag  = false
	terminatekillFlag = false
)

var servicesRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "restart services",
	Long:  "Restart a servicestatus",
	Run: func(cmd *cobra.Command, args []string) {
		ecs := ecsclient.New()
		cluster, service := helpers.ServicePicker(ecs, args)

		printServiceStatus(cluster, service)
		if terminatekillFlag == true {
			terminatekill(&cluster, &service)
		} else if rotatingkillFlag == true {
			rotatingkill(&cluster, &service)
		} else {
			restartgracefully(&cluster, &service)
		}

	},
}

func rotatingkill(cluster *string, service *string) {

	var err error
	var tcs []*ecsclient.TaskInfo
	ecs := ecsclient.New()

	tcs, err = ecs.GetContainerInstances(cluster, service)
	if err != nil {
		fmt.Printf("Error getting container instances: %s\n", err)
		os.Exit(1)
	}

	if len(tcs) < 2 {
		fmt.Printf("There are less than 2 tasks running, please use the option --terminatekill")
		os.Exit(1)
	}

	taskarns := make([]*string, len(tcs))

	fmt.Println("Currently running tasks:")
	for i, ti := range tcs {
		taskarns[i] = ti.TaskArn
		fmt.Printf("%s:%s\t - %s - %s\n ", *ti.IpAddress, strconv.FormatInt(*ti.Hostport0, 10), *ti.TaskArn, *ti.Ec2InstanceId)
	}

	if helpers.GetYesNo("Start rotating kill tasks ?") {
		rotatingkillhelper(cluster, service, taskarns, len(taskarns))
		fmt.Printf("do it")
	} else {
		fmt.Printf("exitting")
		os.Exit(0)
	}
}

func sliceremove(s []*string, i int) []*string {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func rotatingkillhelper(cluster *string, service *string, arns []*string, initialcount int) bool {
	if len(arns) == 0 {
		fmt.Println("Done.")
		return true
	}

	ecs := ecsclient.New()
	taskarns, err := ecs.GetTaskArnsForService(cluster, service)
	if err != nil {
		fmt.Printf("Error getting tasks: %s", err)
		os.Exit(1)
	}

	if len(arns) == 0 {
		return true
	}

	tobekilledtask := arns[0]
	arns = sliceremove(arns, 0)

	_, errr := ecs.StopTask(cluster, tobekilledtask)
	if err != nil {
		fmt.Printf("Failed to kill task instance: %s", errr)
		os.Exit(1)
	}

	fmt.Println("----- Currently running tasks  --------------------------------------------------------")
	for _, task := range taskarns {
		fmt.Printf("Task: %s\n", *task)
	}
	fmt.Println("-----  to be killed tasks  --------------------------------------------------------")
	for _, task := range arns {
		fmt.Printf("Task: %s\n", *task)
	}
	fmt.Println("-----  Just killed task  --------------------------------------------------------")
	fmt.Printf("Task: %s\n", *tobekilledtask)
	fmt.Println("Waiting for the amount of tasks to be back to the initial value, in cases of autoscaling this could fail. Waiting for a task to come back again.")

	countmax := 600
	for countmax > 0 {
		fmt.Printf(". sleep . ")
		taskarns, err = ecs.GetTaskArnsForService(cluster, service)
		if err != nil {
			fmt.Printf("Error getting container instances: %s", err)
			os.Exit(1)
		}

		if len(taskarns) == initialcount {
			fmt.Println("=================================================")
			return rotatingkillhelper(cluster, service, arns, initialcount)
		}

		time.Sleep(time.Duration(5) * time.Second)
		countmax = countmax - 5
	}
	return false
}

func terminatekill(cluster *string, service *string) {
	if helpers.Confirm(fmt.Sprintf("KILL %s", strings.ToUpper(*service))) {
		ecs := ecsclient.New()
		taskarns, err := ecs.GetTaskArnsForService(cluster, service)
		if err != nil {
			fmt.Printf("Error getting tasks: %s", err)
			os.Exit(1)
		}
		for _, task := range taskarns {
			fmt.Printf("Killing Task: %s\n", *task)
			_, errr := ecs.StopTask(cluster, task)
			if err != nil {
				fmt.Printf("Failed to kill task instance: %s", errr)
				os.Exit(1)
			}
		}
	}

}

func printServiceStatus(cluster, service string) {
	ecs := ecsclient.New()

	serviceObj, err := ecs.FindService(&cluster, &service)
	if err != nil {
		log.Fatalf("Could not find service %s %s", cluster, service)
	}

	fmt.Printf("Cluster:\t\t%s\n", cluster)
	fmt.Printf("Service:\t\t%s\n", service)
	fmt.Printf("Task Definition:\t%s\n", path.Base(*serviceObj.TaskDefinition))
	fmt.Println("Deployments:")
	for _, d := range serviceObj.Deployments {
		fmt.Printf("%s %s (%d Desired, %d Pending, %d Running) %v\n", path.Base(*d.TaskDefinition), *d.Status, *d.DesiredCount, *d.PendingCount, *d.RunningCount, *d.CreatedAt)
	}
	task := path.Base(*serviceObj.TaskDefinition)
	defs, err := ecs.GetContainerDefinitions(&task)
	if err != nil {
		log.Fatalf("%v", err)
	}

	fmt.Println("---------------------------------------------------------------------------------------")

	for _, d := range defs {
		memreservation := "-1"
		if d.MemoryReservation != nil {
			memreservation = strconv.FormatInt(*d.MemoryReservation, 10)
		}
		fmt.Printf("CPU %v, Soft Memory limit: %s, Hard memory limit: %s\n", *d.Cpu, strconv.FormatInt(*d.Memory, 10), memreservation)
	}
}

func restartgracefully(cluster *string, service *string) {
	ecs := ecsclient.New()
	ecs.RestartService(cluster, service)
}

func init() {
	RootCmd.AddCommand(servicesRestartCmd)
	servicesRestartCmd.Flags().BoolVarP(&rotatingkillFlag, "rotatingkill", "r", false, "Kill all tasks but not at the same time aka. Rolling kill.")
	servicesRestartCmd.Flags().BoolVarP(&terminatekillFlag, "terminatekill", "t", false, "Kill al tasls at the same time.. FEAR THIS.")
}
