package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var pullTag string
var crossPullActivated bool
var testCommand string
var forceBuild = false

var config = ""

func Init() {

	viper.SetConfigName("config")         // name of config file (without extension)
	viper.AddConfigPath("/etc/skipper/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.skipper") // call multiple times to add many search paths
	err := viper.ReadInConfig()           // Find and read the config file
	if err != nil {
		fmt.Println("Failed to read config", err)
	}

	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
}
