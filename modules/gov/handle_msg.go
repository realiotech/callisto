package gov

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"

	"github.com/forbole/callisto/v4/types"
	"github.com/forbole/callisto/v4/utils"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	juno "github.com/forbole/juno/v6/types"
)

var msgFilter = map[string]bool{
	"/cosmos.gov.v1.MsgSubmitProposal": true,
	"/cosmos.gov.v1.MsgDeposit":        true,
	"/cosmos.gov.v1.MsgVote":           true,
	"/cosmos.gov.v1.MsgVoteWeighted":   true,

	"/cosmos.gov.v1beta1.MsgSubmitProposal": true,
	"/cosmos.gov.v1beta1.MsgDeposit":        true,
	"/cosmos.gov.v1beta1.MsgVote":           true,
	"/cosmos.gov.v1beta1.MsgVoteWeighted":   true,
}

// HandleMsgExec implements modules.AuthzMessageModule
func (m *Module) HandleMsgExec(index int, _ int, executedMsg juno.Message, tx *juno.Transaction) error {
	return m.HandleMsg(index, executedMsg, tx)
}

// HandleMsg implements modules.MessageModule
func (m *Module) HandleMsg(index int, msg juno.Message, tx *juno.Transaction) error {
	if _, ok := msgFilter[msg.GetType()]; !ok {
		return nil
	}

	log.Debug().Str("module", "gov").Str("hash", tx.TxHash).Uint64("height", tx.Height).Msg(fmt.Sprintf("handling gov message %s", msg.GetType()))

	switch msg.GetType() {
	case "/cosmos.gov.v1.MsgSubmitProposal":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &govtypesv1.MsgSubmitProposal{})
		return m.handleSubmitProposalEvent(tx, cosmosMsg.Proposer, cosmosMsg.InitialDeposit, sdk.StringifyEvents(tx.Events))
	case "/cosmos.gov.v1beta1.MsgSubmitProposal":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &govtypesv1beta1.MsgSubmitProposal{})
		return m.handleSubmitProposalEvent(tx, cosmosMsg.Proposer, cosmosMsg.InitialDeposit, sdk.StringifyEvents(tx.Events))

	case "/cosmos.gov.v1.MsgDeposit":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &govtypesv1.MsgDeposit{})
		return m.handleDepositEvent(tx, cosmosMsg.Depositor, cosmosMsg.Amount, sdk.StringifyEvents(tx.Events))
	case "/cosmos.gov.v1beta1.MsgDeposit":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &govtypesv1beta1.MsgDeposit{})
		return m.handleDepositEvent(tx, cosmosMsg.Depositor, cosmosMsg.Amount, sdk.StringifyEvents(tx.Events))

	case "/cosmos.gov.v1.MsgVote":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &govtypesv1.MsgVote{})
		return m.handleVoteEvent(tx, cosmosMsg.Voter, sdk.StringifyEvents(tx.Events))
	case "/cosmos.gov.v1beta1.MsgVote":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &govtypesv1beta1.MsgVote{})
		return m.handleVoteEvent(tx, cosmosMsg.Voter, sdk.StringifyEvents(tx.Events))

	case "/cosmos.gov.v1.MsgVoteWeighted":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &govtypesv1.MsgVoteWeighted{})
		return m.handleVoteEvent(tx, cosmosMsg.Voter, sdk.StringifyEvents(tx.Events))
	case "/cosmos.gov.v1beta1.MsgVoteWeighted":
		cosmosMsg := utils.UnpackMessage(m.cdc, msg.GetBytes(), &govtypesv1beta1.MsgVoteWeighted{})

		return m.handleVoteEvent(tx, cosmosMsg.Voter, sdk.StringifyEvents(tx.Events))
	}

	return nil
}

// handleSubmitProposalEvent allows to properly handle a handleSubmitProposalEvent
func (m *Module) handleSubmitProposalEvent(tx *juno.Transaction, proposer string, initialDeposit sdk.Coins, events sdk.StringEvents) error {
	// Get the proposal id
	proposalID, err := ProposalIDFromEvents(events)
	if err != nil {
		return fmt.Errorf("error while getting proposal id: %s", err)
	}

	if err := m.fetchAndSaveProposal(int64(tx.Height), proposalID); err != nil {
		return err
	}

	// Submit proposal must have a deposit event with depositor equal to the proposer
	return m.handleDepositEvent(tx, proposer, initialDeposit, events)
}

// fetchAndSaveProposal queries the given proposal from the chain and stores it, unless it's
// already stored. votes/deposits/tallies all reference proposal(id) via foreign key, and a
// MsgSubmitProposal is not the only way a proposal ID first becomes relevant to us: a proposal
// submitted before the indexer's tracked window began (eg. it started syncing mid-chain, or
// missed a block) never gets a row from handleSubmitProposalEvent, but a later MsgDeposit/MsgVote
// against that same still-open proposal will still arrive - so this is called defensively before
// storing any of those, to backfill the proposal on first reference instead of failing the FK.
func (m *Module) fetchAndSaveProposal(height int64, proposalID uint64) error {
	has, err := m.db.HasProposal(proposalID)
	if err != nil {
		return fmt.Errorf("error while checking proposal existence: %s", err)
	}
	if has {
		return nil
	}

	// Get the proposal
	proposal, err := m.source.Proposal(height, proposalID)
	if err != nil {
		if strings.Contains(err.Error(), codes.NotFound.String()) {
			// query the proposal details using the latest height stored in db
			// to fix the rpc error returning code = NotFound desc = proposal x doesn't exist
			block, err := m.db.GetLastBlockHeightAndTimestamp()
			if err != nil {
				return fmt.Errorf("error while getting latest block height: %s", err)
			}
			proposal, err = m.source.Proposal(block.Height, proposalID)
			if err != nil {
				return fmt.Errorf("error while getting proposal: %s", err)
			}
		} else {
			return fmt.Errorf("error while getting proposal: %s", err)
		}
	}

	var addresses []types.Account
	for _, msg := range proposal.Messages {
		var sdkMsg sdk.Msg
		err := m.cdc.UnpackAny(msg, &sdkMsg)
		if err != nil {
			return fmt.Errorf("error while unpacking proposal message: %s", err)
		}

		switch msg := sdkMsg.(type) {
		case *distrtypes.MsgCommunityPoolSpend:
			addresses = append(addresses, types.NewAccount(msg.Recipient))
		case *govtypesv1.MsgExecLegacyContent:
			content, ok := msg.Content.GetCachedValue().(*distrtypes.CommunityPoolSpendProposal)
			if ok {
				addresses = append(addresses, types.NewAccount(content.Recipient))
			}
		}
	}

	err = m.db.SaveAccounts(addresses)
	if err != nil {
		return fmt.Errorf("error while storing proposal recipient: %s", err)
	}

	// Unpack the proposal interfaces
	err = proposal.UnpackInterfaces(m.cdc)
	if err != nil {
		return fmt.Errorf("error while unpacking proposal interfaces: %s", err)
	}

	// Store the proposal
	proposalObj := types.NewProposal(
		proposal.Id,
		proposal.Title,
		proposal.Summary,
		proposal.Metadata,
		proposal.Messages,
		proposal.Status.String(),
		*proposal.SubmitTime,
		*proposal.DepositEndTime,
		proposal.VotingStartTime,
		proposal.VotingEndTime,
		proposal.Proposer,
	)

	err = m.db.SaveProposals([]types.Proposal{proposalObj})
	if err != nil {
		return fmt.Errorf("error while saving proposal: %s", err)
	}

	return nil
}

// handleDepositEvent allows to properly handle a handleDepositEvent. amount is what this
// specific message deposited - not the depositor's cumulative on-chain deposit for the proposal,
// which is what m.source.ProposalDeposit returns. Each MsgDeposit gets its own row keyed by
// (proposal_id, depositor_address, transaction_hash), so storing the cumulative total there would
// make every earlier deposit's row overcount once the same depositor deposits again.
func (m *Module) handleDepositEvent(tx *juno.Transaction, depositor string, amount sdk.Coins, events sdk.StringEvents) error {
	// Get the proposal id
	proposalID, err := ProposalIDFromEvents(events)
	if err != nil {
		return fmt.Errorf("error while getting proposal id: %s", err)
	}

	// The proposal might predate the indexer's tracked window (eg. it started syncing mid-chain),
	// in which case this deposit is the first time we ever see this proposal ID - back it fill in
	// first so the insert below doesn't fail its foreign key.
	if err := m.fetchAndSaveProposal(int64(tx.Height), proposalID); err != nil {
		return err
	}

	txTimestamp, err := time.Parse(time.RFC3339, tx.Timestamp)
	if err != nil {
		return fmt.Errorf("error while parsing time: %s", err)
	}

	return m.db.SaveDeposits([]types.Deposit{
		types.NewDeposit(proposalID, depositor, amount, txTimestamp, tx.TxHash, int64(tx.Height)),
	})
}

// handleVoteEvent allows to properly handle a handleVoteEvent
func (m *Module) handleVoteEvent(tx *juno.Transaction, voter string, events sdk.StringEvents) error {
	// Get the proposal id
	proposalID, err := ProposalIDFromEvents(events)
	if err != nil {
		return fmt.Errorf("error while getting proposal id: %s", err)
	}

	// See the same call in handleDepositEvent: the proposal might predate the indexer's tracked
	// window, in which case this vote is the first time we ever see this proposal ID.
	if err := m.fetchAndSaveProposal(int64(tx.Height), proposalID); err != nil {
		return err
	}

	txTimestamp, err := time.Parse(time.RFC3339, tx.Timestamp)
	if err != nil {
		return fmt.Errorf("error while parsing time: %s", err)
	}

	// Get the vote option
	weightVoteOption, err := WeightVoteOptionFromEvents(events)
	if err != nil {
		return fmt.Errorf("error while getting vote option: %s", err)
	}

	// A vote fully replaces whatever the voter previously cast for this proposal on-chain, so
	// clear out any option from a prior vote before storing the new one (eg. switching a straight
	// Yes to a straight No must not leave the old Yes row sitting next to the new No one).
	err = m.db.DeleteVotesByProposalAndVoter(proposalID, voter)
	if err != nil {
		return fmt.Errorf("error while deleting previous vote: %s", err)
	}

	for _, weightVote := range weightVoteOption {
		vote := types.NewVote(proposalID, voter, weightVote.Option, weightVote.Weight, txTimestamp, int64(tx.Height))

		err = m.db.SaveVote(vote)
		if err != nil {
			return fmt.Errorf("error while saving vote: %s", err)
		}
	}

	// update tally result for given proposal
	return m.UpdateProposalTallyResult(proposalID, int64(tx.Height))
}
