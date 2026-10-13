package multistaking

import (
	"fmt"

	"github.com/go-co-op/gocron"
	"github.com/rs/zerolog/log"

	"github.com/forbole/callisto/v4/modules/utils"
)

// RegisterPeriodicOperations implements modules.PeriodicOperationsModule
func (m *Module) RegisterPeriodicOperations(scheduler *gocron.Scheduler) error {
	log.Debug().Str("module", "multistaking").Msg("setting up periodic tasks")

	if _, err := scheduler.Every(1).Hour().Do(func() {
		utils.WatchMethod(m.UpdateMultiStaking)
	}); err != nil {
		return fmt.Errorf("error while setting up multistaking periodic operation: %s", err)
	}

	return nil
}
