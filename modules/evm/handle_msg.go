package evm

import (
	"fmt"

	"github.com/rs/zerolog/log"
	juno "github.com/forbole/juno/v6/types"

	"github.com/forbole/callisto/v4/utils"

	// evmtypes "github.com/evmos/os/x/evm/types"
	cosmosevmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/evm/x/vm/types/legacy"
)

var msgFilter = map[string]bool{
	"/os.evm.v1.MsgEthereumTx":  true,
	"/cosmos.evm.vm.v1.MsgEthereumTx": true,
}

// HandleMsgExec implements modules.AuthzMessageModule
func (m *Module) HandleMsgExec(index int, _ int, executedMsg juno.Message, tx *juno.Transaction) error {
	return m.HandleMsg(index, executedMsg, tx)
}

// HandleMsg implements MessageModule
func (m *Module) HandleMsg(_ int, msg juno.Message, tx *juno.Transaction) error {
	if _, ok := msgFilter[msg.GetType()]; !ok {
		return nil
	}

	log.Debug().Str("module", "evm").Str("hash", tx.TxHash).Uint64("height", tx.Height).Msg(fmt.Sprintf("handling evm message %s", msg.GetType()))

	switch msg.GetType() {
	case "/os.evm.v1.MsgEthereumTx":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &legacy.MsgEthereumTx{})
		return m.db.SaveEvmTx(int64(tx.Height), tx.TxHash, cosmosMsg.Hash)
	case "/cosmos.evm.vm.v1.MsgEthereumTx":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &cosmosevmtypes.MsgEthereumTx{})
		return m.db.SaveEvmTx(int64(tx.Height), tx.TxHash, cosmosMsg.GetHash().String())
	}
	return nil
}
