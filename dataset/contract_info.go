package dataset

import (
	"context"
	"sync"

	"github.com/Troublor/erebus/troubeth"

	"github.com/ethereum/go-ethereum/event"

	"github.com/ethereum/go-ethereum/common"
)

type ContractInfoProvider struct {
	service         *troubeth.TroubEth
	isVerifiedTasks *sync.Map // map[common.Address]*event.Feed
}

func NewContractInfoProvider(service *troubeth.TroubEth) *ContractInfoProvider {
	return &ContractInfoProvider{
		service:         service,
		isVerifiedTasks: &sync.Map{},
	}
}

type isVerifiedEvent struct {
	isVerified bool
	err        error
}

// IsVerified returns true if the contract is verified on Etherscan.
// Data is obtained from TroubEth service.
// This function is thread safe.
func (p *ContractInfoProvider) IsVerified(ctx context.Context, contract common.Address) (bool, error) {
	feed, loaded := p.isVerifiedTasks.LoadOrStore(contract, &event.Feed{})
	if !loaded {
		go func() {
			verified, err := p.service.IsVerified(ctx, contract)
			if err != nil {
				feed.(*event.Feed).Send(isVerifiedEvent{err: err})
			} else {
				feed.(*event.Feed).Send(isVerifiedEvent{isVerified: verified})
			}
		}()
	}
	resultFeed := feed.(*event.Feed)
	ch := make(chan isVerifiedEvent)
	sub := resultFeed.Subscribe(ch)
	defer sub.Unsubscribe()
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case result := <-ch:
		return result.isVerified, result.err
	}
}
