package mint

import (
	"github.com/forbole/callisto/v4/modules/utils"

	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"
)

// RegisterPeriodicOperations implements modules.PeriodicOperationsModule
func (m *Module) RegisterPeriodicOperations(scheduler *gocron.Scheduler) error {
	log.Debug().Str("module", "mint").Msg("setting up periodic tasks")

	return nil
}

// updateInflation fetches from the REST APIs the latest value for the
// inflation, and saves it inside the database.
func (m *Module) UpdateInflation() error {
	log.Debug().
		Str("module", "mint").
		Str("operation", "inflation").
		Msg("getting inflation data")

	block, err := m.db.GetLastBlockHeightAndTimestamp()
	if err != nil {
		return err
	}

	// Get the inflation
	inflation, err := m.source.GetInflation(block.Height)
	if err != nil {
		return err
	}

	return m.db.SaveInflation(inflation, block.Height)
}
