package utils

import (
	"sync"

	"cosmossdk.io/x/evidence"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	consensus "github.com/cosmos/cosmos-sdk/x/consensus"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	cosmosevmcryptocodec "github.com/cosmos/evm/crypto/codec"
	cosmosfeemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	cosmosevmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	multistakingtypes "github.com/realio-tech/multi-staking-module/x/multi-staking/types"
	ethcryptocodec "github.com/realiotech/realio-network/crypto/codec"
	bridgemoduletypes "github.com/realiotech/realio-network/x/bridge/types"
	assetmoduletypes "github.com/realiotech/realio-network/x/asset/types"
)

var once sync.Once
var cdc *codec.ProtoCodec

func GetCodec() codec.Codec {
	once.Do(func() {
		interfaceRegistry := codectypes.NewInterfaceRegistry()
		getBasicManagers().RegisterInterfaces(interfaceRegistry)
		// ostypes.RegisterInterfaces(interfaceRegistry)
		ethcryptocodec.RegisterInterfaces(interfaceRegistry)
		interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil),
			&MsgEthereumTx{},
			&MsgUpdateParams{},
		)
		cosmosevmcryptocodec.RegisterInterfaces(interfaceRegistry)
		cosmosevmtypes.RegisterInterfaces(interfaceRegistry)
		ibcclienttypes.RegisterInterfaces(interfaceRegistry)
		// evmtypes.RegisterInterfaces(interfaceRegistry)
		cosmosfeemarkettypes.RegisterInterfaces(interfaceRegistry)
		// cryptocodec.RegisterInterfaces(interfaceRegistry)
		multistakingtypes.RegisterInterfaces(interfaceRegistry)
		bridgemoduletypes.RegisterInterfaces(interfaceRegistry)
		assetmoduletypes.RegisterInterfaces(interfaceRegistry)
		std.RegisterInterfaces(interfaceRegistry)
		cdc = codec.NewProtoCodec(interfaceRegistry)
	})
	return cdc
}

// getBasicManagers returns the various basic managers that are used to register the encoding to
// support custom messages.
// This should be edited by custom implementations if needed.
func getBasicManagers() module.BasicManager {
	return module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
			},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		groupmodule.AppModuleBasic{},
		vesting.AppModuleBasic{},
		consensus.AppModuleBasic{},
	)
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
