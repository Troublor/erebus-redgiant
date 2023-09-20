package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/Troublor/erebus-redgiant/dataset"
	"github.com/Troublor/erebus-redgiant/global"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
)

var migrateCmd = &cobra.Command{
	Use: "migrate",
	Run: func(cmd *cobra.Command, args []string) {
		migrate()
	},
}

func migrate() {
	db := global.MongoDbClient().Database("erebus-redgiant")
	attackCollection := db.Collection("attacks")

	dir := "./legacy-dataset/attacks"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to read dir")
	}

	ctx := global.Ctx()
	chainReader := global.BlockchainReader()
	pool := global.GoroutinePool()

	var wg sync.WaitGroup
	for _, f := range files {
		content, err := os.ReadFile(path.Join(dir, f.Name()))
		if err != nil {
			log.Error().Err(err).Str("file", f.Name()).Msg("failed to read file")
			continue
		}
		var data map[string]interface{}
		err = json.Unmarshal(content, &data)
		if err != nil {
			log.Error().Err(err).Str("file", f.Name()).Msg("failed to unmarshal json")
			continue
		}

		attack := data["attack"].(map[string]interface{})
		aTx := common.HexToHash(attack["attackTx"].(string))
		vTx := common.HexToHash(attack["victimTx"].(string))
		var pTx *common.Hash
		if attack["profitTx"] != nil {
			pTx = lo.ToPtr(common.HexToHash(attack["profitTx"].(string)))
		}

		hash := dataset.ComputeAttackHash(aTx, vTx, pTx)
		r, err := attackCollection.CountDocuments(ctx, bson.D{{"hash", hash.Hex()}})
		if err != nil {
			log.Error().Err(err).Str("hash", hash.Hex()).Msg("failed to count document")
			continue
		}
		if r > 0 {
			log.Info().Str("hash", hash.Hex()).Msg("attack already exists")
			continue
		}

		wg.Add(1)
		err = pool.Submit(func() {
			defer wg.Done()
			defer func() {
				err := recover()
				if err != nil {
					log.Error().Err(err.(error)).Str("hash", hash.Hex()).Msg("panic")
				}
			}()

			attack, err := dataset.ConstructAttack(ctx, chainReader, aTx, vTx, pTx)
			if err != nil {
				log.Error().
					Err(err).
					Str("hash", hash.Hex()).
					Msg("failed to construct attack")
				return
			}

			attackBson := attack.AsAttackBSON()
			_, err = attackCollection.InsertOne(ctx, attackBson)
			if err != nil {
				log.Error().Err(err).Str("hash", hash.Hex()).Msg("failed to insert attack")
				return
			} else {
				log.Info().Str("hash", hash.Hex()).Msg("inserted attack")
			}
		})

		if err != nil {
			wg.Done()
			log.Error().Err(err).Str("file", f.Name()).Msg("failed to submit task")
			continue
		}
	}

	wg.Wait()
}
