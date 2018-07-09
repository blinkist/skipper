package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel [service]",
	Short: "Secure Shell into one of the service container instances' EC2 host machines",
	Long: `
TUNNELLL
`,
	Run: func(cmd *cobra.Command, args []string) {
		err := InvokeShell()
		if err != nil {
			fmt.Printf("error invoking shell: %v", err)
			os.Exit(-1)
		}
	},
}

var shellCmd = &cobra.Command{
	Use:   "shell <subcommand>",
	Short: "Shell commands",
}

var setkeypairCmd = &cobra.Command{
	Use:   "setkeypair [service]",
	Short: "Create a temporary EC2 keypair for the current user",
	Long: `
Lets say you have a service running a few container instances on various hosts.
Lets say you also need to debug some file configuration running in the container,
or you maybe need to check some files on disk of the Host, you can shell right
in there through this command using a menu system rather than figuring it out
yourself! Yay. I guess.
`,
	Run: func(cmd *cobra.Command, args []string) {
		SetKeypair()
	},
}

var purgeKeypairCmd = &cobra.Command{
	Use:   "purgekeypair [service]",
	Short: "Purge the current user's keypair on both filesystem and AWS",
	Run: func(cmd *cobra.Command, args []string) {
		// //		ec2client_ := ec2client.New()

		// //		GetDebugInstances(ec2client_)

		// 		tag := GetInstanceTag()
		// 		fmt.Println(*tag)
		// 		instances := ec2client_.GetInstancesWithTagName(tag)

		// 		for _, k := range instances {
		// 			fmt.Println("abc")
		// 			fmt.Println(k)

		// 		}
		// 		//keyname := GetKeyPairName()

	},
}

func init() {
	RootCmd.AddCommand(shellCmd)
	shellCmd.AddCommand(setkeypairCmd)
	shellCmd.AddCommand(purgeKeypairCmd)
	shellCmd.AddCommand(tunnelCmd)
}
