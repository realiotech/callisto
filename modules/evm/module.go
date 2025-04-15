package evm

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/forbole/juno/v6/modules"

	"github.com/forbole/callisto/v4/database"
)

var (
	_ modules.Module                     = &Module{}
	_ modules.MessageModule              = &Module{}
)

// Module represents the x/staking module
type Module struct {
	cdc    codec.Codec
	db     *database.Db
}

// NewModule returns a new Module instance
func NewModule(
	cdc codec.Codec, db *database.Db,
) *Module {
	return &Module{
		cdc:    cdc,
		db:     db,
	}
}

// Name implements modules.Module
func (m *Module) Name() string {
	return "evm"
}
