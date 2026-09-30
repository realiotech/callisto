package bank

import (
	"fmt"

	"github.com/forbole/callisto/v4/types"

	"github.com/rs/zerolog/log"
)

// RunAdditionalOperations implements modules.AdditionalOperationsModule
func (m *Module) RunAdditionalOperations() error {

	return m.initAccountBalances()
}

func (m *Module) initAccountBalances() error {
	log.Trace().Str("module", "bank").Str("operation", "account balance").
		Msg("init account balance")

	block, err := m.db.GetLastBlockHeightAndTimestamp()
	if err != nil {
		return fmt.Errorf("error while getting latest block height: %s", err)
	}

	return m.refreshAllAccountBalances(block.Height)
}

// refreshAllAccountBalances rebuilds the entire balance table from the current chain
// state (every denom owner of every denom in the total supply) at the given height.
// Unlike the block-by-block event driven update, this doesn't depend on any bank
// event being emitted/captured, so it acts as a safety net that self-heals any
// address (eg. module accounts like bonded_tokens_pool/not_bonded_tokens_pool,
// whose balance only ever changes via internal keeper calls rather than user
// messages) whose balance drifted out of sync from missed or unemitted events.
func (m *Module) refreshAllAccountBalances(height int64) error {
	tokens, err := m.keeper.GetSupply(height)
	if err != nil {
		return err
	}

	var accountBalances []types.AccountBalance
	for _, tokenUnit := range tokens {
		denom := tokenUnit.Denom
		holders, err := m.keeper.GetDenomOwners(height, denom)
		if err != nil {
			return fmt.Errorf("error while get denom holder: %s", err)
		}

		for _, holder := range holders {
			addr := holder.GetAddress()
			if addr == "" {
				continue
			}
			accountBalances = append(accountBalances, types.NewAccountBalance(addr, holder.Balance, height))
		}
	}

	return m.db.SaveAccountBalances(accountBalances, height)
}
