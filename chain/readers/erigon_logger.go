package readers

import (
	"github.com/ledgerwatch/log/v3"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

func erigonZeroLogHandler() log.Handler {
	return log.FuncHandler(func(r *log.Record) error {
		msg := log.TerminalFormatNoColor().Format(r)
		zlog.WithLevel(erigonLogLvlToZeroLogLvl(r.Lvl)).Msg(string(msg))
		return nil
	})
}

func erigonLogLvlToZeroLogLvl(lvl log.Lvl) zerolog.Level {
	switch lvl {
	case log.LvlCrit:
		return zerolog.FatalLevel
	case log.LvlError:
		return zerolog.ErrorLevel
	case log.LvlWarn:
		return zerolog.WarnLevel
	case log.LvlInfo:
		return zerolog.InfoLevel
	case log.LvlDebug:
		return zerolog.DebugLevel
	case log.LvlTrace:
		return zerolog.TraceLevel
	default:
		return zerolog.NoLevel
	}
}
