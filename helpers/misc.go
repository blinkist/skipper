package helpers

import (
	"bufio"
	"fmt"

	"log"
	"os"
	"sort"
	"strings"

	"github.com/blinkist/skipper/aws/ecsclient"

	"strconv"
)

func ServicePicker(ecs *ecsclient.Ecsclient, args []string) (string, string) {
	var cluster, service string
	if len(args) > 0 {
		cluster = args[0]
	}
	if len(args) > 1 {
		service = args[1]
	}

	clusterAndServices := make(map[string][]string)

	names, err := ecs.GetClusterNames()
	if err != nil {
		log.Fatalf("Error getting clusters %v", err)
	}

	for _, clusterName := range names {
		services, err := ecs.ListServices(&clusterName)

		if err != nil {
			log.Fatalf("Error listing services %v", err)
		}

		if len(services) > 0 {
			clusterAndServices[clusterName] = services
		}
	}

	if cluster == "" {
		clusternames := make([]string, 0, len(clusterAndServices))
		for k := range clusterAndServices {
			clusternames = append(clusternames, k)
		}

		cluster = PickOption(clusternames, "Please choose a cluster")

	}
	if service == "" {
		services := clusterAndServices[cluster]
		service = PickOption(services, "Please choose a service")

		if err != nil {
			log.Fatal(err)
		}
	}
	return cluster, service
}

func GetUserStringInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("\n >>> ")
	text, err := reader.ReadString('\n')
	return text, err
}

func GetUserIntInput() (int, error) {
	input, err := GetUserStringInput()

	if err != nil {
		return 0, fmt.Errorf("Could receive string %v", err)
	}

	if len(input) < 1 {
		return 0, fmt.Errorf("not enough arguments supplied")
	}

	myint, err := strconv.Atoi(input[:len(input)-1])

	if err != nil {
		return 0, fmt.Errorf("Could not convert input to integer %v", err)
	}
	return myint, nil
}

func PickOption(options []string, title string) string {
	if len(options) == 1 {
		return options[0]
	}
	for {
		fmt.Printf("%s:\n", title)
		sort.Strings(options)
		for choice, disp := range options {
			fmt.Printf("[%d]: %s\n", choice+1, disp)
		}
		myint, err := GetUserIntInput()

		myint--

		// We do not want a negative index
		if err == nil && myint >= 0 && myint < len(options) {
			return options[myint]
		}
	}
}

func GetYesNo(text string) bool {

	for {
		fmt.Printf("%s [y/n]: ", text)

		res, err := GetUserStringInput()
		if err != nil {
			log.Fatal(err)
		}

		res = strings.ToLower(strings.TrimSpace(res))

		if res == "y" || res == "yes" {
			return true
		} else if res == "n" || res == "no" {
			return false
		}
	}
}

func Confirm(repeat string) bool {

	for {
		fmt.Printf("Please confirm by writing the following [%s]: ", repeat)

		res, err := GetUserStringInput()

		if err != nil {
			log.Fatal(err)
		}
		res = strings.ToUpper(strings.TrimSpace(res))
		repeat = strings.ToUpper(strings.TrimSpace(repeat))
		if res == repeat {
			return true
		} else {
			return false
		}
	}
}
