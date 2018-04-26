package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	argCpu     int64
	argSoftmem int64
	argHardmem int64
)

var setlimitsCmd = &cobra.Command{
	Use:   "setlimits",
	Short: "Set limits regarding CPU/SOFTMEM/HARDMEM of a service",
	Long:  "View current cluster or service deployment status",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Printf("No service defined")
			os.Exit(1)
		}

		err := checkInput()
		if err != nil {
			fmt.Printf("An error occured : %s", err)
			os.Exit(1)
		}
	},
}

func checkInput() error {
	if argSoftmem != -1 && argHardmem != -1 {
		if argSoftmem > argHardmem {
			return fmt.Errorf("reserved Soft memory needs to be smaller than Hard Memory")
		}
	}
	if argSoftmem == -1 && argHardmem == -1 && argCpu == -1 {
		return fmt.Errorf("at least one of the properties need to be set [cpu,softmem,hardmem]")
	}
	return nil
}

func init() {
	RootCmd.AddCommand(setlimitsCmd)
	setlimitsCmd.Flags().Int64VarP(&argCpu, "cpu", "", -1, "The amount of cpu units ")
	setlimitsCmd.Flags().Int64VarP(&argSoftmem, "softmem", "", -1, "The amount of soft-memory reserved MB ")
	setlimitsCmd.Flags().Int64VarP(&argHardmem, "hardmem", "", -1, "The amount of hard-memory reserved MB ")
}
