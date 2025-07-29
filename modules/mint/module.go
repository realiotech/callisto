package mint

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/forbole/juno/v6/modules"

	"github.com/forbole/callisto/v4/database"
	mintsource "github.com/forbole/callisto/v4/modules/mint/source"
	banksource "github.com/forbole/callisto/v4/modules/bank/source"
)

var (
	_ modules.Module                   = &Module{}
	_ modules.GenesisModule            = &Module{}
	_ modules.PeriodicOperationsModule = &Module{}
)

// Module represent database/mint module
type Module struct {
	cdc    codec.Codec
	db     *database.Db
	source mintsource.Source
	bankSource banksource.Source
}

// NewModule returns a new Module instance
func NewModule(source mintsource.Source, bankSource banksource.Source, cdc codec.Codec, db *database.Db) *Module {
	return &Module{
		cdc:    cdc,
		db:     db,
		source: source,
		bankSource: bankSource,
	}
}

// Name implements modules.Module
func (m *Module) Name() string {
	return "mint"
}
