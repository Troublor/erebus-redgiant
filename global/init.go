package global

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
)

func init() {
	setupLog()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info().Msg("Got interrupt...")
		Cleanup()
		os.Exit(1)
	}()
}

var cleanupTasks []func()

func RegisterCleanupTask(task func()) {
	cleanupTasks = append(cleanupTasks, task)
}

func Cleanup() {
	for i := len(cleanupTasks) - 1; i >= 0; i-- {
		cleanupTasks[i]()
	}
}
