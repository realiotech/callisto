package bank

import (
	"fmt"

	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/forbole/callisto/v4/modules/utils"
)

// RegisterPeriodicOperations implements modules.Module
func (m *Module) RegisterPeriodicOperations(scheduler *gocron.Scheduler) error {
	log.Debug().Str("module", "bank").Msg("setting up periodic tasks")

	if _, err := scheduler.Every(1).Hour().Do(func() {
		utils.WatchMethod(m.UpdateSupply)
	}); err != nil {
		return fmt.Errorf("error while setting up bank periodic operation: %s", err)
	}

	if _, err := scheduler.Every(1).Hour().Do(func() {
		utils.WatchMethod(m.RunAdditionalOperations)
	}); err != nil {
		return fmt.Errorf("error while setting up bank periodic operation: %s", err)
	}

	return nil
}

// UpdateSupply updates the supply of all the tokens
func (m *Module) UpdateSupply() error {
	log.Trace().Str("module", "bank").Str("operation", "total supply").
		Msg("updating total supply")

	block, err := m.db.GetLastBlockHeightAndTimestamp()
	if err != nil {
		return fmt.Errorf("error while getting latest block height: %s", err)
	}

	supply, err := m.keeper.GetSupply(block.Height)
	if err != nil {
		return err
	}

	err = m.db.SaveSupply(supply, block.Height)
	if err != nil {
		return err
	}

	err = m.updateTokenHolder(block.Height, supply)
	return err
}

// updateTokenHolder updates num holder of all the tokens
func (m *Module) updateTokenHolder(height int64, tokens sdk.Coins) error {
	log.Trace().Str("module", "bank").Str("operation", "total holder").
		Msg("updating token holder")

	total := make(map[string]int)
	for _, tokenUnit := range tokens {
		denom := tokenUnit.Denom
		holders, err := m.keeper.GetDenomOwners(height, denom)
		if err != nil {
			return fmt.Errorf("error while updating holder: %s", err)
		}

		total[denom] = len(holders)
	}

	if len(total) == 0 {
		return nil
	}
	return m.db.SaveTokenHolder(total, height)
}
