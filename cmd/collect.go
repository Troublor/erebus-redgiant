package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Troublor/erebus-redgiant/config"
	"github.com/Troublor/erebus-redgiant/dataset"
	"github.com/Troublor/erebus-redgiant/global"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/semaphore"
)

// flags.
var (
	cCollectFrom = config.Def{
		Type:     config.Uint64,
		Key:      "from",
		KeyShort: "f",
		Default:  uint64(64),
		Desc:     "block number to search from",
	}
	cCollectTo = config.Def{
		Type:     config.Uint64,
		Key:      "to",
		KeyShort: "t",
		Default:  uint64(0),
		Desc:     "block number to search to",
	}
	cCollectWindow = config.Def{
		Type:     config.Int,
		Key:      "window",
		KeyShort: "w",
		Default:  1,
		Desc:     "window size",
	}
	cCollectStep = config.Def{
		Type:     config.Int,
		Key:      "step",
		KeyShort: "s",
		Default:  1,
		Desc:     "step size",
	}
	cCollectConcurrencyDegree = config.Def{
		Type:     config.Int,
		Key:      "concurrency",
		KeyShort: "c",
		Default:  runtime.NumCPU(),
		Desc:     "concurrency degree",
	}
	cCollectWindowParallel = config.Def{
		Type:     config.Int64,
		Key:      "window-parallel",
		KeyShort: "p",
		Default:  int64(1),
		Desc:     "window parallel degree",
	}
	cCollectPrefetch = config.Def{
		Type:    config.Int,
		Key:     "prefetch",
		Default: 1,
		Desc:    "prefetch degree",
	}
	cCollectWindowSearchTimeout = config.Def{
		Type:    config.String,
		Key:     "window-search-timeout",
		Default: "15s",
		Desc:    "window search timeout",
	}
	cCollectLocalizeTimeout = config.Def{
		Type:    config.String,
		Key:     "localize-timeout",
		Default: "15s",
		Desc:    "localize timeout",
	}
)

var cCollectGroup = config.NewDefGroup("collect",
	cCollectFrom,
	cCollectTo,
	cCollectWindow,
	cCollectStep,
	cCollectConcurrencyDegree,
	cCollectWindowParallel,
	cCollectPrefetch,
	cCollectWindowSearchTimeout,
	cCollectLocalizeTimeout,
)

var collectCmd = &cobra.Command{
	Use: "collect",
	Run: func(cmd *cobra.Command, args []string) {
		searchForAttacks()
	},
}

func init() {
	collectCmd.Flags().AddFlagSet(cCollectGroup.FlagSet())
	cCollectGroup.BindToViper()
}

func searchForAttacks() {
	var (
		from                = viper.GetUint64(cCollectGroup.KeyOf(cCollectFrom))
		to                  = viper.GetUint64(cCollectGroup.KeyOf(cCollectTo))
		step                = viper.GetInt(cCollectGroup.KeyOf(cCollectStep))
		window              = viper.GetInt(cCollectGroup.KeyOf(cCollectWindow))
		concurrency         = viper.GetInt(cCollectGroup.KeyOf(cCollectConcurrencyDegree))
		windowParallel      = viper.GetInt64(cCollectGroup.KeyOf(cCollectWindowParallel))
		prefetch            = viper.GetInt(cCollectGroup.KeyOf(cCollectPrefetch))
		windowSearchTimeout = viper.GetString(cCollectGroup.KeyOf(cCollectWindowSearchTimeout))
		localizeTimeout     = viper.GetString(cCollectGroup.KeyOf(cCollectLocalizeTimeout))
	)
	windowSearchTO, err := time.ParseDuration(windowSearchTimeout)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse window search timeout")
	}
	localizeTO, err := time.ParseDuration(localizeTimeout)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse localize timeout")
	}

	startTime := time.Now()
	interrupted := false

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Info().Str("signal", sig.String()).Msg("Got signal, canceling")
		interrupted = true
		global.CtxCancel()
	}()

	log.Info().
		Uint64("from", from).
		Uint64("to", to).
		Int("window", window).
		Int("step", step).
		Int("concurrency", concurrency).
		Int64("window-parallel", windowParallel).
		Int("prefetch", prefetch).
		Str("window-search-timeout", windowSearchTimeout).
		Str("localize-timeout", localizeTimeout).
		Msg("Search in block range")

	err = collect(
		from, to,
		window, step, windowParallel,
		prefetch, windowSearchTO, localizeTO,
	)
	if err != nil {
		log.Error().Err(err).Msg("Error occurs when collecting dataset")
	}
	log.Info().
		Uint64("from", from).
		Uint64("to", to).
		Msg("Search finished")
}

func collect(
	from, to uint64, window, step int, windowParallel int64,
	prefetch int, windowSearchTimeout, localizeTimeout time.Duration,
) error {
	startTime := time.Now()

	history := dataset.NewTxHistory(global.BlockchainReader(), global.TroubEth())
	historyWorkerPool, err := ants.NewPool(prefetch)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create history worker pool")
		return err
	}
	defer historyWorkerPool.Release()

	dbName := viper.GetString(config.CMongoDatabase.Key)
	log.Info().Str("db", dbName).Msg("Connecting to mongo")
	db := global.MongoDbClient().Database(dbName)

	searcher := dataset.NewAttackSearcher(global.BlockchainReader(), history)
	searcher.SetPool(global.GoroutinePool())
	searcher.SetAttackHandler(func(session *dataset.TxHistorySession, attack *dataset.Attack) {
		_, err := attack.Analyze(global.BlockchainReader(), session)
		if err != nil {
			l := log.Error().Err(err).
				Str("attackTx", attack.AttackTxRecord.Tx.Hash().Hex()).
				Str("victimTx", attack.VictimTxRecord.Tx.Hash().Hex())
			if attack.ProfitTxRecord != nil {
				l = l.Str("profitTx", attack.ProfitTxRecord.Tx.Hash().Hex())
			}
			l.Msg("Failed to analyze attack")
		}

		// save attack to database
		_, err = db.Collection("attacks").InsertOne(global.Ctx(), attack.AsAttackBSON())
		if err != nil {
			l := log.Error().Err(err).
				Str("attackTx", attack.AttackTxRecord.Tx.Hash().Hex()).
				Str("victimTx", attack.VictimTxRecord.Tx.Hash().Hex())
			if attack.ProfitTxRecord != nil {
				l = l.Str("profitTx", attack.ProfitTxRecord.Tx.Hash().Hex())
			}
			l.Msg("Failed to insert attack to database")
			return
		}

		// save analysis to database
		// for _, analysis := range attack.Analysis {
		// 	_, err = db.Collection("analysis").InsertOne(global.Ctx(), analysis.AsAttackAnalysisBSON())
		// 	if err != nil {
		// 		log.Error().Err(err).
		// 			Str("attackID", analysis.Attack.AsAttackBSON().ID.Hex()).
		// 			Str("analysisID", analysis.AsAttackAnalysisBSON().ID.Hex()).
		// 			Msg("Failed to insert analysis to database")
		// 		continue
		// 	}
		// }
	})
	go searcher.PrefetchBlocks(global.Ctx(), historyWorkerPool, prefetch)

	var searchPivot = from
	var windowSemaphore = semaphore.NewWeighted(windowParallel)
	for b := from; b < to; b += uint64(step) {
		pivot := searchPivot
		bn := b
		if global.Ctx().Err() != nil {
			break
		}

		if err = windowSemaphore.Acquire(global.Ctx(), 1); err != nil {
			log.Error().Err(err).Msg("Failed to acquire window semaphore")
			break
		}
		searchWindow := searcher.OpenSearchWindow(global.Ctx(), bn, window)
		searchWindow.SetSearchPivot(pivot, 0)
		log.Info().Uint64("from", bn).Msg("Searching window")
		go func() {
			defer func() {
				// wait some time before release resources so that other windows can reuse them.
				history.ForgetBlockRange(bn, bn+uint64(step))
				time.Sleep(windowSearchTimeout * time.Duration(windowParallel))
				searchWindow.Close()
			}()
			defer windowSemaphore.Release(1)
			windowSearchDDLCtx, cancel := context.WithTimeout(global.Ctx(), windowSearchTimeout)
			defer cancel()
			log.Info().
				Uint64("from", bn).
				Int("window", window).
				Str("pivot", fmt.Sprintf("%d:%d", pivot, 0)).
				Msg("Searching for attack cases")
			searchWindow.Search(windowSearchDDLCtx)
			timeElapsed := time.Since(startTime)
			log.Info().
				Uint64("from", bn).
				Int("window", window).
				Str("pivot", fmt.Sprintf("%d:%d", pivot, 0)).
				Str("timeElapsed", timeElapsed.String()).
				Str("speed", fmt.Sprintf("%.2f s/w", timeElapsed.Seconds()/float64(bn-from))).
				Msg("Searching for attack cases done")
		}()

		searchPivot = bn + uint64(window)
	}

	return nil
}
