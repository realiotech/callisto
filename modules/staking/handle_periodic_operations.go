package staking

import (
	"fmt"

	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"

	"github.com/forbole/callisto/v4/modules/utils"
)

// RunAdditionalOperations implements modules.AdditionalOperationsModule. It runs once at
// startup and backfills state that's otherwise only ever written by a specific message/event
// (MsgCreateValidator, MsgEditValidator, a staking param-change proposal, genesis) - so a
// validator/param that already existed before the indexer started syncing (or before a genesis
// parse succeeded) would otherwise never get this data.
func (m *Module) RunAdditionalOperations() error {
	if err := m.UpdateValidatorsData(); err != nil {
		return err
	}

	block, err := m.db.GetLastBlockHeightAndTimestamp()
	if err != nil {
		return fmt.Errorf("error while getting latest block height from db: %s", err)
	}

	// validator_description/validator_commission for every current validator
	if err := m.RefreshAllValidatorInfos(block.Height); err != nil {
		return err
	}

	// staking_params
	return m.UpdateParams(block.Height)
}

// RegisterPeriodicOperations implements modules.PeriodicOperationsModule
func (m *Module) RegisterPeriodicOperations(scheduler *gocron.Scheduler) error {
	log.Debug().Str("module", "staking").Msg("setting up periodic tasks")

	// Update the staking pool every 5 mins
	if _, err := scheduler.Every(5).Minutes().Do(func() {
		utils.WatchMethod(m.UpdateStakingPool)
	}); err != nil {
		return fmt.Errorf("error while scheduling staking pool periodic operation: %s", err)
	}

	// refresh proposal validators status snapshots every 5 mins
	if _, err := scheduler.Every(5).Minutes().Do(func() {
		utils.WatchMethod(m.UpdateValidatorStatuses)
	}); err != nil {
		return fmt.Errorf("error while setting up gov period operations: %s", err)
	}

	return nil
}

// UpdateStakingPool reads from the LCD the current staking pool and stores its value inside the database
func (m *Module) UpdateStakingPool() error {
	block, err := m.db.GetLastBlockHeightAndTimestamp()
	if err != nil {
		return fmt.Errorf("error while getting latest block height: %s", err)
	}
	log.Debug().Str("module", "staking").Int64("height", block.Height).
		Msg("updating staking pool")

	pool, err := m.GetStakingPool(block.Height)
	if err != nil {
		return fmt.Errorf("error while getting staking pool: %s", err)

	}

	err = m.db.SaveStakingPool(pool)
	if err != nil {
		return fmt.Errorf("error while saving staking pool: %s", err)

	}

	return nil
}
