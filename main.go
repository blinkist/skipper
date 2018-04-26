package main

import (
	"fmt"
	"os"

	"github.com/blinkist/skipper/config"
	"github.com/spf13/cobra"
)

var cfgFile string

var (
	argService     string
	argServiceType string
	argTimeout     int
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "skipper",
	Short: "Helper tools for Amazon's Elastic Container Service",
	Long:  `Wraps the ECS SDK in a more user-friendly (for me at least) way`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
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
	Execute()
}

func init() {
	config.Init()
	RootCmd.Flags().IntVarP(&argTimeout, "timeout", "", 300, "Default timeout for task replacement.")
}
