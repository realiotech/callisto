package bank

import (
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"

	juno "github.com/forbole/juno/v6/types"

	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/rs/zerolog/log"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// HandleBlock implements BlockModule
func (m *Module) HandleBlock(
	block *tmctypes.ResultBlock, res *tmctypes.ResultBlockResults, _ []*juno.Transaction, _ *tmctypes.ResultValidators,
) error {
	events := res.FinalizeBlockEvents
	for _, tx := range res.TxsResults {
		events = append(events, tx.Events...)
	}
	err := m.updateBalanceByEvent(block.Block.Height, events)
	if err != nil {
		fmt.Printf("Error when update balance by end block: %s", err)
	}

	return nil
}

// removeExpiredFeeGrantAllowances removes fee grant allowances in database that have expired
func (m *Module) updateBalanceByEvent(height int64, events []abci.Event) error {
	log.Debug().Str("module", "bank").Int64("height", height).
		Msg("updating balance by event")

	setAddr := make(map[string]struct{})
	for _, event := range events {
		switch event.Type {
		case banktypes.EventTypeTransfer:
			address, err := juno.FindAttributeByKey(event, banktypes.AttributeKeyRecipient)
			if err == nil && address.Value != "" {
				setAddr[address.Value] = struct{}{}
			}

			address, err = juno.FindAttributeByKey(event, banktypes.AttributeKeySender)
			if err == nil && address.Value != "" {
				setAddr[address.Value] = struct{}{}
			}

		case banktypes.EventTypeCoinSpent:
			address, err := juno.FindAttributeByKey(event, banktypes.AttributeKeySpender)
			if err == nil && address.Value != "" {
				setAddr[address.Value] = struct{}{}
			}

		case banktypes.EventTypeCoinReceived:
			address, err := juno.FindAttributeByKey(event, banktypes.AttributeKeyReceiver)
			if err == nil && address.Value != "" {
				setAddr[address.Value] = struct{}{}
			}

		case banktypes.EventTypeCoinMint:
			address, err := juno.FindAttributeByKey(event, banktypes.AttributeKeyMinter)
			if err == nil && address.Value != "" {
				setAddr[address.Value] = struct{}{}
			}

		case banktypes.EventTypeCoinBurn:
			address, err := juno.FindAttributeByKey(event, banktypes.AttributeKeyBurner)
			if err == nil && address.Value != "" {
				setAddr[address.Value] = struct{}{}
			}

		}
	}

	var addresses []string
	for address := range setAddr {
		addresses = append(addresses, address)
	}
	if len(addresses) == 0 {
		return nil
	}
	return m.UpdateBalance(addresses, height)
}

func (m *Module) UpdateBalance(addresses []string, height int64) error {
	log.Trace().Str("module", "bank").Str("operation", "account balance").
		Msg("updating account balance")

	accountBalances, err := m.keeper.GetBalances(addresses, height)
	if err != nil {
		return err
	}

	err = m.db.SaveAccountBalances(accountBalances, height)
	return err
}
