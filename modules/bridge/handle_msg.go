package bridge

import (
	"fmt"

	"github.com/rs/zerolog/log"
	juno "github.com/forbole/juno/v6/types"

	"github.com/forbole/callisto/v4/utils"

	bridgetypes "github.com/realiotech/realio-network/x/bridge/types"
)

var msgFilter = map[string]bool{
	"/realionetwork.bridge.v1.MsgBridgeIn":  true,
	"/realionetwork.bridge.v1.MsgBridgeOut": true,
}

// HandleMsgExec implements modules.AuthzMessageModule
func (m *Module) HandleMsgExec(index int, authzMsgIndex int, executedMsg juno.Message, tx *juno.Transaction) error {
	return m.handleMsg(index, authzMsgIndex, executedMsg, tx)
}

// HandleMsg implements MessageModule
func (m *Module) HandleMsg(index int, msg juno.Message, tx *juno.Transaction) error {
	return m.handleMsg(index, -1, msg, tx)
}

// handleMsg handles a single bridge message. msgIndex/authzMsgIndex identify the message's
// position within its transaction (authzMsgIndex is -1 outside of a MsgExec). They're only
// folded into the storage key when there's more than one bridge message to disambiguate (see
// bridgeMsgKey), so the common case of one bridge message per transaction keeps the plain tx
// hash as its key, unchanged from before.
func (m *Module) handleMsg(msgIndex int, authzMsgIndex int, msg juno.Message, tx *juno.Transaction) error {
	if _, ok := msgFilter[msg.GetType()]; !ok {
		return nil
	}

	log.Debug().Str("module", "bridge").Str("hash", tx.TxHash).Uint64("height", tx.Height).Msg(fmt.Sprintf("handling bridge message %s", msg.GetType()))

	key := bridgeMsgKey(tx.TxHash, msgIndex, authzMsgIndex)
	switch msg.GetType() {
	case "/realionetwork.bridge.v1.MsgBridgeIn":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &bridgetypes.MsgBridgeIn{})
		return m.db.SaveBridgeIn(key, cosmosMsg)

	case "/realionetwork.bridge.v1.MsgBridgeOut":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &bridgetypes.MsgBridgeOut{})
		return m.db.SaveBridgeOut(key, cosmosMsg)
	}
	return nil
}

// bridgeMsgKey returns the key used to store a bridge message. bridge_in/bridge_out are keyed
// by a single TEXT "hash" column holding the transaction hash, which silently dropped every
// bridge message but the first whenever a transaction batched more than one of them (relayer
// batching, or an authz-executed batch). Suffixing the index only in that case disambiguates
// them without needing a schema change: the overwhelmingly common single-message case still
// gets the plain, unmodified transaction hash.
func bridgeMsgKey(txHash string, msgIndex int, authzMsgIndex int) string {
	if msgIndex == 0 && authzMsgIndex == -1 {
		return txHash
	}
	return fmt.Sprintf("%s-%d-%d", txHash, msgIndex, authzMsgIndex)
}
