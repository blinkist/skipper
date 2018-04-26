package main

import (
	//"bytes"

	"github.com/spf13/cobra"

	"golang.org/x/crypto/ssh"
)

var session *ssh.Session
var exitTimeStamp int32

var tunnelCmd = &cobra.Command{
	Use:   "tunnel [service]",
	Short: "Secure Shell into one of the service container instances' EC2 host machines",
	Long: `
TUNNELLL
`,
	Run: func(cmd *cobra.Command, args []string) {
		InvokeShell()
	},
}

var shellCmd = &cobra.Command{
	Use:   "shell [service]",
	Short: "empty",
	Long: `

`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var setkeypairCmd = &cobra.Command{
	Use:   "setkeypair [service]",
	Short: "Secure Shell into one of the service container instances' EC2 host machines",
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

var listinstancesCmd = &cobra.Command{
	Use:   "listinstances [service]",
	Short: "We purge the keypair on both filesystem and AWS",
	Long: `
	We purge the keypair on both filesystem and AWS
`,
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
	exitTimeStamp = 0
	RootCmd.AddCommand(shellCmd)
	shellCmd.AddCommand(setkeypairCmd)
	shellCmd.AddCommand(listinstancesCmd)
	shellCmd.AddCommand(tunnelCmd)

}
