package dyengine

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Tx is a wrapper of Ethereum's types.Transaction.
// It provides some additional features.
type Tx struct {
	*types.Transaction

	isPseudo bool
	// PseudoExec is specific for transaction with type PseudoTxType.
	// Pseudo tx is allowed to directly modify State, without changing VMContext.
	// It will panic if the tx.Type() is not PseudoTxType.
	PseudoExec func(State) ([]byte, error)

	signed bool
	from   common.Address
}

func TxFromTransactionWithSigner(tx *types.Transaction, signer types.Signer) (*Tx, error) {
	var err error
	var from common.Address
	if signer == nil {
		from = common.Address{}
	} else {
		from, err = signer.Sender(tx)
		if err != nil {
			return nil, err
		}
	}

	return &Tx{
		Transaction: tx,

		signed: true,
		from:   from,
	}, nil
}

func NewTx(from common.Address, inner types.TxData) *Tx {
	tx := &Tx{
		Transaction: types.NewTx(inner),

		signed: false,
		from:   from,
	}
	return tx
}

func NewPseudoTx(data []byte, exec func(State) ([]byte, error)) *Tx {
	return &Tx{
		Transaction: types.NewTx(&types.LegacyTx{Data: data}),
		isPseudo:    true,
		PseudoExec:  exec,
	}
}

func (tx *Tx) IsPseudo() bool {
	return tx.isPseudo
}

func (tx *Tx) Signed() bool {
	return tx.signed
}

func (tx *Tx) From() common.Address {
	return tx.from
}
