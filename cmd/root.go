package main

import (
	"fmt"

	"github.com/Troublor/erebus-redgiant/config"
	"github.com/Troublor/erebus-redgiant/global"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	rootCmd = &cobra.Command{
		Use: "redgaint",
	}
)

// Execute executes the root command.
func Execute() error {
	defer global.Cleanup()
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().AddFlagSet(config.GlobalFlagSet)
	err := viper.BindPFlags(config.GlobalFlagSet)
	if err != nil {
		panic(fmt.Errorf("failed to bind global flags: %w", err))
	}

	rootCmd.AddCommand(collectCmd)
	rootCmd.AddCommand(migrateCmd)
}

func initConfig() {
}
