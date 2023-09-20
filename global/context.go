package global

import (
	"context"

	"github.com/panjf2000/ants/v2"

	"github.com/Troublor/erebus-redgiant/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var (
	ctx       context.Context
	ctxCancel context.CancelFunc
	pool      *ants.Pool
)

func Ctx() context.Context {
	if ctx != nil {
		return ctx
	}

	ctx, ctxCancel = context.WithCancel(context.Background())
	RegisterCleanupTask(ctxCancel)
	return ctx
}

func CtxCancel() context.CancelFunc {
	if ctxCancel != nil {
		return ctxCancel
	}

	ctx, ctxCancel = context.WithCancel(context.Background())
	return ctxCancel
}

func GoroutinePool() *ants.Pool {
	if pool != nil {
		return pool
	}

	var err error
	pool, err = ants.NewPool(viper.GetInt(config.CConcurrency.Key))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create goroutine pool")
	}
	RegisterCleanupTask(pool.Release)

	return pool
}
