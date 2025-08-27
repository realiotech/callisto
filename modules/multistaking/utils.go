package multistaking

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/forbole/callisto/v4/types"

	multistakingtypes "github.com/realio-tech/multi-staking-module/x/multi-staking/types"
)

func (m *Module) CompleteUnbonding(height int64, stakerAddr string, valAddr string) error {
	msunlock, err := m.source.GetMultiStakingUnlock(height, stakerAddr, valAddr)
	if err != nil {
		return err
	}

	err = m.UpdateUnlockToken(height, stakerAddr, valAddr, msunlock)
	if err != nil {
		return err
	}

	// Now drop the old unlock data and save the new data
	err = m.db.DropMultiStakingUnlock(stakerAddr, valAddr)
	if err != nil {
		return err
	}

	if msunlock != nil {
		err = m.db.SaveMultiStakingUnlock(height, msunlock)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Module) convertValidatorInfo(info *multistakingtypes.ValidatorInfo) (types.MSValidatorInfo, error) {
	var pubKey cryptotypes.PubKey
	err := m.cdc.UnpackAny(info.ConsensusPubkey, &pubKey)
	if err != nil {
		return types.MSValidatorInfo{}, err
	}
	return types.MSValidatorInfo{
		ConsensusAddress: convertPubkeyToAddr(pubKey),
		Denom:            info.BondDenom,
	}, nil
}

func convertPubkeyToAddr(pubkey cryptotypes.PubKey) string {
	return sdk.ConsAddress(pubkey.Address()).String()
}
