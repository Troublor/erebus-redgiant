package global

import (
	"io"
	"os"
	"strings"

	"github.com/Troublor/erebus-redgiant/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func setupLog() {
	out := viper.GetString(config.CLogFile.Key)
	loc := viper.GetBool(config.CLogLocation.Key)
	level := viper.GetUint(config.CLogLevel.Key)
	zerolog.SetGlobalLevel(zerolog.Level(level))
	splits := strings.Split(out, ";")
	writers := make([]io.Writer, 0)
	for _, split := range splits {
		if split == "stdout" {
			writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout})
		} else if split == "stderr" {
			writers = append(writers, zerolog.ConsoleWriter{Out: os.Stderr})
		} else {
			f, err := os.OpenFile(split, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to open log file")
			}
			writers = append(writers, zerolog.SyncWriter(f))
		}
	}
	multiWriter := zerolog.MultiLevelWriter(writers...)
	loggerBuilder := zerolog.New(multiWriter).With()
	if loc {
		loggerBuilder = loggerBuilder.Caller()
	}
	loggerBuilder = loggerBuilder.Timestamp()
	logger := loggerBuilder.Logger()
	log.Logger = logger // we use global logger
}
