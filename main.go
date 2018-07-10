package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/blinkist/skipper/config"
)

var (
	cfgFile string

	argService     string
	argServiceType string
	argTimeout     int

	RootCmd = &cobra.Command{
		Use:     "skipper",
		Short:   "Helper tools for Amazon's Elastic Container Service",
		Long:    `Skipper is a command-line tool to help working with Amazon ECS clusters`,
		Version: "0.0.1",
		// Uncomment the following line if your bare application
		// has an action associated with it:
		//	Run: func(cmd *cobra.Command, args []string) { },
	}
)

func init() {
	RootCmd.Flags().IntVarP(&argTimeout, "timeout", "", 300, "Default timeout for task replacement.")
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func main() {
	if err := config.Init(); err != nil {
		fmt.Println("error initialising config:", err)
		os.Exit(1)
	}
	Execute()
}
