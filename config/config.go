package config

import (
	"strings"

	"github.com/spf13/viper"
)

var (
	pullTag            string
	crossPullActivated bool
	testCommand        string
	forceBuild         = false

	config = ""
)

func Init() error {

	viper.SetConfigName("config")         // name of config file (without extension)
	viper.AddConfigPath("/etc/skipper/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.skipper") // call multiple times to add many search paths
	err := viper.ReadInConfig()           // Find and read the config file
	switch err.(type) {
	case viper.ConfigFileNotFoundError:
		err = nil
	}
	if err != nil {
		return err
	}
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	return nil
}
