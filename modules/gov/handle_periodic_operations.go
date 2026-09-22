package gov

import (
	"fmt"

	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"

	"github.com/forbole/callisto/v4/modules/utils"
)

// RunAdditionalOperations implements modules.AdditionalOperationsModule. gov_params is
// otherwise only ever written at genesis or on a gov param-change proposal, so refresh it once
// at startup too - a chain whose genesis wasn't (fully) parsed, or a param set before the
// indexer ever ran, would otherwise leave this table empty/stale forever.
func (m *Module) RunAdditionalOperations() error {
	block, err := m.db.GetLastBlockHeightAndTimestamp()
	if err != nil {
		return fmt.Errorf("error while getting latest block height: %s", err)
	}

	return m.UpdateParams(block.Height)
}

// RegisterPeriodicOperations implements modules.PeriodicOperationsModule
func (m *Module) RegisterPeriodicOperations(scheduler *gocron.Scheduler) error {
	log.Debug().Str("module", "gov").Msg("setting up periodic tasks")

	// refresh proposal staking pool snapshots every 5 mins
	// (set the same interval as staking pool periodic ops)
	if _, err := scheduler.Every(5).Minutes().Do(func() {
		utils.WatchMethod(m.UpdateProposalsStakingPoolSnapshot)
	}); err != nil {
		return fmt.Errorf("error while setting up gov period operations: %s", err)
	}

	// refresh proposal tally results every 5 mins
	if _, err := scheduler.Every(5).Minutes().Do(func() {
		utils.WatchMethod(m.UpdateProposalsTallyResults)
	}); err != nil {
		return fmt.Errorf("error while setting up gov period operations: %s", err)
	}

	return nil
}
