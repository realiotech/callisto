package staking

import (
	"encoding/hex"
	"fmt"

	"github.com/forbole/callisto/v4/types"

	juno "github.com/forbole/juno/v6/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/rs/zerolog/log"
)

// HandleBlock implements BlockModule
func (m *Module) HandleBlock(
	block *tmctypes.ResultBlock, res *tmctypes.ResultBlockResults, _ []*juno.Transaction, vals *tmctypes.ResultValidators,
) error {

	// Handle case create val by precompile
	events := res.FinalizeBlockEvents
	for _, tx := range res.TxsResults {
		events = append(events, tx.Events...)
	}
	err := m.updateValsByEvent(block.Block.Height, events)
	if err != nil {
		fmt.Printf("Error when updateValsByEvent, error: %s", err)
	}

	// Update the validators
	_, err = m.updateValidators(block.Block.Height)
	if err != nil {
		return fmt.Errorf("error while updating validators: %s", err)
	}

	// Updated the double sign evidences
	go m.updateDoubleSignEvidence(block.Block.Height, block.Block.Evidence.Evidence)

	return nil
}

func (m *Module) updateValsByEvent(height int64, events []abci.Event) error {
	for _, event := range events {
		switch event.Type {
		case stakingtypes.EventTypeCreateValidator:
			m.RefreshAllValidatorInfos(height)
		}
	}
	return nil
}

// updateDoubleSignEvidence updates the double sign evidence of all validators
func (m *Module) updateDoubleSignEvidence(height int64, evidenceList tmtypes.EvidenceList) {
	log.Debug().Str("module", "staking").Int64("height", height).
		Msg("updating double sign evidence")

	var evidences []types.DoubleSignEvidence
	for _, ev := range evidenceList {
		dve, ok := ev.(*tmtypes.DuplicateVoteEvidence)
		if !ok {
			continue
		}

		evidences = append(evidences, types.NewDoubleSignEvidence(
			height,
			types.NewDoubleSignVote(
				int(dve.VoteA.Type),
				dve.VoteA.Height,
				dve.VoteA.Round,
				dve.VoteA.BlockID.String(),
				juno.ConvertValidatorAddressToBech32String(dve.VoteA.ValidatorAddress),
				dve.VoteA.ValidatorIndex,
				hex.EncodeToString(dve.VoteA.Signature),
			),
			types.NewDoubleSignVote(
				int(dve.VoteB.Type),
				dve.VoteB.Height,
				dve.VoteB.Round,
				dve.VoteB.BlockID.String(),
				juno.ConvertValidatorAddressToBech32String(dve.VoteB.ValidatorAddress),
				dve.VoteB.ValidatorIndex,
				hex.EncodeToString(dve.VoteB.Signature),
			),
		),
		)
	}

	err := m.db.SaveDoubleSignEvidences(evidences)
	if err != nil {
		log.Error().Str("module", "staking").Err(err).Int64("height", height).
			Msg("error while saving double sign evidence")
		return
	}

}
