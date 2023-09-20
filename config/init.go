package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

func init() {
	loadConfigFile()
	loadEnv()
	loadFlags()
}

// loadConfigFile loads the config file as REDGIANT_CONFIG env variable specifies (default: config.yaml).
func loadConfigFile() {
	if configFile, exist := os.LookupEnv("REDGIANT_CONFIG"); exist {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config") // name of config file (without extension)
		viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
		viper.AddConfigPath(".")      // optionally look for config in the current directory
	}
	// it doesn't matter if we don't find the config file, they can be passed via flags or env variables
	_ = viper.ReadInConfig()
}

func loadEnv() {
	viper.SetEnvPrefix("EREBUS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}
