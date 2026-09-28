package utils

import (
	"sync"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	cosmosevmcryptocodec "github.com/cosmos/evm/crypto/codec"
	legacyevmtypes "github.com/cosmos/evm/x/vm/types/legacy"
	"github.com/cosmos/gogoproto/proto"
	realioapp "github.com/realiotech/realio-network/app"
	ethcryptocodec "github.com/realiotech/realio-network/crypto/codec"

	ibcclientv10types "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
)

var once sync.Once
var cdc *codec.ProtoCodec

func GetCodec() codec.Codec {
	once.Do(func() {
		interfaceRegistry := codectypes.NewInterfaceRegistry()
		getBasicManagers().RegisterInterfaces(interfaceRegistry)
		ethcryptocodec.RegisterInterfaces(interfaceRegistry)
		cosmosevmcryptocodec.RegisterInterfaces(interfaceRegistry)
		ibcclientv10types.RegisterInterfaces(interfaceRegistry)
		std.RegisterInterfaces(interfaceRegistry)

		// Pre-migration (evmos/os) chain history still needs to decode: old MsgEthereumTx/gov
		// content used the "os.evm.v1.*" type URLs, which the current "cosmos.evm.vm.v1.*" ones
		// (registered above via ModuleBasics) don't cover. Rather than depending on evmos/os itself
		// (its x/evm/types pulls in x/evm/core/vm, which needs go-ethereum/crypto/bls12381 - a
		// package the go-ethereum fork this repo replaces it with doesn't have), cosmos/evm ships a
		// small self-contained "legacy" package with just the old wire types under the same
		// "os.evm.v1.*" names, with no such dependency. modules/evm/handle_msg.go already decodes
		// old MsgEthereumTx via this package; this registration is what makes that decode succeed
		// instead of panicking in UnpackMessage. Mirrors legacy.initLegacyTxConfig(), which the real
		// chain app's own legacy tx decoder uses for the same purpose.
		interfaceRegistry.RegisterImplementations(
			(*sdktx.TxExtensionOptionI)(nil),
			&legacyevmtypes.ExtensionOptionsEthereumTx{},
		)
		interfaceRegistry.RegisterImplementations(
			(*sdk.Msg)(nil),
			&legacyevmtypes.MsgEthereumTx{},
			&legacyevmtypes.MsgUpdateParams{},
		)
		interfaceRegistry.RegisterInterface(
			"os.evm.v1.TxData",
			(*legacyevmtypes.TxData)(nil),
			&legacyevmtypes.DynamicFeeTx{},
			&legacyevmtypes.AccessListTx{},
			&legacyevmtypes.LegacyTx{},
		)

		cdc = codec.NewProtoCodec(interfaceRegistry)
	})
	return cdc
}

// getBasicManagers returns the basic manager used to register message/type encoding for every
// module the chain actually runs. It reuses realio-network's own app.ModuleBasics - the same list
// the chain's app.go registers - instead of a hand-picked subset kept in sync by hand here: a
// module added to the chain (staking, IBC, EVM, or a custom one) is registered here automatically,
// with no risk of someone forgetting to also add it to this file. app.go marks its own list with a
// scaffolding comment ("stargate/app/moduleBasic") precisely because new modules are expected to be
// appended there - that guarantee is lost if we hand-copy only a subset of it here.
func getBasicManagers() module.BasicManager {
	return realioapp.ModuleBasics
}

// UnpackMessage unpacks a message from a byte slice
func UnpackMessage[T proto.Message](cdc codec.Codec, bz []byte, ptr T) T {
	var any codectypes.Any
	cdc.MustUnmarshalJSON(bz, &any)
	var cosmosMsg sdk.Msg
	if err := cdc.UnpackAny(&any, &cosmosMsg); err != nil {
		panic(err)
	}
	return cosmosMsg.(T)
}
