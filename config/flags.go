package config

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ConfigType int

const (
	String ConfigType = iota
	Int
	Uint
	Int8
	Uint8
	Int32
	Uint32
	Int64
	Uint64
	Bool
)

// Def in this package are those common ones used by almost every where.
type Def struct {
	Type     ConfigType // default to string
	Key      string
	KeyShort string // only valid in command line arguments, leave empty if not used
	Default  any
	Desc     string
}

var (
	CLogTraceFile = Def{
		Key:     "log.trace",
		Default: "logs/trace.log",
	}
	CLogLevel = Def{
		Type:    Uint8,
		Key:     "log.level",
		Default: uint8(zerolog.InfoLevel),
	}
	CLogFile = Def{
		Key:     "log.file",
		Default: "stdout",
	}
	CLogLocation = Def{
		Type:    Bool,
		Key:     "log.location",
		Default: false,
	}
)

var (
	CEthApiUrl = Def{
		Key:     "eth.url",
		Default: "http://localhost:8545",
	}
	CErigonRpc = Def{
		Key:     "erigon.rpc",
		Default: "localhost:9090",
	}
	CErigonDataDir = Def{
		Key:     "erigon.datadir",
		Default: "",
	}
	CTroubEthUrl = Def{
		Key:     "troubeth.url",
		Default: "http://localhost:2425",
	}
)

var (
	CMongoURL = Def{
		Key:     "mongo.url",
		Default: "mongodb://localhost:27017",
	}
	CMongoDatabase = Def{
		Key:     "mongo.database",
		Default: "erebus",
	}
)

var (
	CConcurrency = Def{
		Type:    Int,
		Key:     "concurrency",
		Default: 1,
	}
)

var GlobalFlagDefs = []Def{
	CLogTraceFile,
	CLogLevel,
	CLogFile,
	CLogLocation,

	CEthApiUrl,
	CErigonRpc,
	CErigonDataDir,
	CTroubEthUrl,

	CMongoURL,
	CMongoDatabase,

	CConcurrency,
}

type DefGroup struct {
	Name string
	Defs map[string]Def

	flagSet *pflag.FlagSet
}

func NewDefGroup(name string, defs ...Def) *DefGroup {
	defGroup := DefGroup{Defs: make(map[string]Def)}
	defGroup.Add(defs...)
	return &defGroup
}

func (g *DefGroup) Add(defs ...Def) {
	for _, def := range defs {
		g.Defs[def.Key] = def
	}
}

func (g *DefGroup) KeyOf(def Def) string {
	if g.Defs[def.Key] == def {
		return fmt.Sprintf("%s.%s", g.Name, def.Key)
	}
	panic(fmt.Sprintf("%s not found in group %s", def.Key, g.Name))
}

func (g *DefGroup) FlagSet() *pflag.FlagSet {
	if g.flagSet == nil {
		slice := make([]Def, 0, len(g.Defs))
		for _, def := range g.Defs {
			slice = append(slice, def)
		}
		g.flagSet = BuildFlagSet(g.Name, slice...)
	}
	return g.flagSet
}

func (g *DefGroup) BindToViper() {
	set := g.FlagSet()
	for k, def := range g.Defs {
		err := viper.BindPFlag(g.KeyOf(def), set.Lookup(k))
		if err != nil {
			panic(fmt.Errorf("failed to bind flag %s to viper: %w", k, err))
		}
	}
}

func loadFlags() {
	setupConfigs(GlobalFlagDefs...)
}

var GlobalFlagSet *pflag.FlagSet = BuildFlagSet(
	"erebus",
	GlobalFlagDefs...,
)

func BuildFlagSet(name string, defs ...Def) *pflag.FlagSet {
	flagSet := pflag.NewFlagSet(name, pflag.ContinueOnError)
	for _, def := range defs {
		switch def.Type {
		case String:
			if def.KeyShort != "" {
				flagSet.StringP(def.Key, def.KeyShort, def.Default.(string), def.Desc)
			} else {
				flagSet.String(def.Key, def.Default.(string), def.Desc)
			}
		case Int:
			if def.KeyShort != "" {
				flagSet.IntP(def.Key, def.KeyShort, def.Default.(int), def.Desc)
			} else {
				flagSet.Int(def.Key, def.Default.(int), def.Desc)
			}
		case Uint:
			if def.KeyShort != "" {
				flagSet.UintP(def.Key, def.KeyShort, def.Default.(uint), def.Desc)
			} else {
				flagSet.Uint(def.Key, def.Default.(uint), def.Desc)
			}
		case Int8:
			if def.KeyShort != "" {
				flagSet.Int8P(def.Key, def.KeyShort, def.Default.(int8), def.Desc)
			} else {
				flagSet.Int8(def.Key, def.Default.(int8), def.Desc)
			}
		case Uint8:
			if def.KeyShort != "" {
				flagSet.Uint8P(def.Key, def.KeyShort, def.Default.(uint8), def.Desc)
			} else {
				flagSet.Uint8(def.Key, def.Default.(uint8), def.Desc)
			}
		case Int32:
			if def.KeyShort != "" {
				flagSet.Int32P(def.Key, def.KeyShort, def.Default.(int32), def.Desc)
			} else {
				flagSet.Int32(def.Key, def.Default.(int32), def.Desc)
			}
		case Uint32:
			if def.KeyShort != "" {
				flagSet.Uint32P(def.Key, def.KeyShort, def.Default.(uint32), def.Desc)
			} else {
				flagSet.Uint32(def.Key, def.Default.(uint32), def.Desc)
			}
		case Int64:
			if def.KeyShort != "" {
				flagSet.Int64P(def.Key, def.KeyShort, def.Default.(int64), def.Desc)
			} else {
				flagSet.Int64(def.Key, def.Default.(int64), def.Desc)
			}
		case Uint64:
			if def.KeyShort != "" {
				flagSet.Uint64P(def.Key, def.KeyShort, def.Default.(uint64), def.Desc)
			} else {
				flagSet.Uint64(def.Key, def.Default.(uint64), def.Desc)
			}
		case Bool:
			if def.KeyShort != "" {
				flagSet.BoolP(def.Key, def.KeyShort, def.Default.(bool), def.Desc)
			} else {
				flagSet.Bool(def.Key, def.Default.(bool), def.Desc)
			}
		}
	}
	return flagSet
}

func setupConfigs(configDefs ...Def) {
	flagSet := BuildFlagSet("erebus", configDefs...)
	for _, def := range configDefs {
		viper.SetDefault(def.Key, def.Default)
	}
	flagSet.ParseErrorsWhitelist.UnknownFlags = true
	flagSet.Usage = func() {}
	_ = flagSet.Parse(os.Args[1:])
	err := viper.BindPFlags(flagSet)
	if err != nil {
		panic(fmt.Errorf("failed to bind flags to viper: %w", err))
	}
}
