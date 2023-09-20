package readers

import (
	"context"
	"fmt"

	"github.com/ledgerwatch/erigon-lib/gointerfaces"
	"github.com/ledgerwatch/erigon-lib/gointerfaces/grpcutil"
	"github.com/ledgerwatch/erigon-lib/gointerfaces/remote"
	"github.com/ledgerwatch/erigon-lib/kv/remotedb"
	"github.com/ledgerwatch/erigon-lib/kv/remotedbserver"
	erigonLog "github.com/ledgerwatch/log/v3"
)

type erigonRpcReader struct {
	*erigonDBReader
}

func NewErigonRpcReader(ctx context.Context, rpcAddr string) (*erigonRpcReader, error) {
	logger := erigonLog.New()
	logger.SetHandler(erigonZeroLogHandler())
	rpcConn, err := grpcutil.Connect(nil, rpcAddr)
	if err != nil {
		return nil, fmt.Errorf("can't connect to erigon rpc: %w", err)
	}
	kvClient := remote.NewKVClient(rpcConn)
	remoteKv, err := remotedb.NewRemote(
		gointerfaces.VersionFromProto(remotedbserver.KvServiceAPIVersion),
		logger, kvClient,
	).ReadOnly().Open()
	if err != nil {
		return nil, fmt.Errorf("can't open remote db: %w", err)
	}

	if compatErr := checkDbCompatibility(ctx, remoteKv); compatErr != nil {
		return nil, fmt.Errorf("erigon db incompatible: %w", compatErr)
	}
	return &erigonRpcReader{erigonDBReader: &erigonDBReader{db: remoteKv}}, nil
}
