package database_test

import (
	"encoding/json"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	dbtypes "github.com/forbole/callisto/v4/database/types"

	"github.com/forbole/callisto/v4/types"

	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func (suite *DbTestSuite) TestBigDipperDb_SaveCommunityPool() {
	// Save data
	original := sdk.NewDecCoins(sdk.NewDecCoin("uatom", math.NewInt(100)))
	err := suite.database.SaveCommunityPool(original, 10)
	suite.Require().NoError(err)

	// Verify data
	expected := dbtypes.NewCommunityPoolRow(dbtypes.NewDbDecCoins(original), 10)
	var rows []dbtypes.CommunityPoolRow
	err = suite.database.Sqlx.Select(&rows, `SELECT * FROM community_pool`)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 1, "community_pool table should contain only one row")
	suite.Require().True(expected.Equals(rows[0]))

	// ---------------------------------------------------------------------------------------------------------------

	// Try updating with lower height
	coins := sdk.NewDecCoins(sdk.NewDecCoin("uatom", math.NewInt(50)))
	err = suite.database.SaveCommunityPool(coins, 5)
	suite.Require().NoError(err)

	// Verify data
	expected = dbtypes.NewCommunityPoolRow(dbtypes.NewDbDecCoins(original), 10)
	rows = []dbtypes.CommunityPoolRow{}
	err = suite.database.Sqlx.Select(&rows, `SELECT * FROM community_pool`)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 1, "community_pool table should contain only one row")
	suite.Require().True(expected.Equals(rows[0]), "updating with lower height should not modify the data")

	// ---------------------------------------------------------------------------------------------------------------

	// Try updating with equal height
	coins = sdk.NewDecCoins(sdk.NewDecCoin("uatom", math.NewInt(120)))
	err = suite.database.SaveCommunityPool(coins, 10)
	suite.Require().NoError(err)

	// Verify data
	expected = dbtypes.NewCommunityPoolRow(dbtypes.NewDbDecCoins(coins), 10)
	rows = []dbtypes.CommunityPoolRow{}
	err = suite.database.Sqlx.Select(&rows, `SELECT * FROM community_pool`)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 1, "community_pool table should contain only one row")
	suite.Require().True(expected.Equals(rows[0]), "updating with same height should modify the data")

	// ---------------------------------------------------------------------------------------------------------------

	// Try updating with higher height
	coins = sdk.NewDecCoins(sdk.NewDecCoin("uatom", math.NewInt(200)))
	err = suite.database.SaveCommunityPool(coins, 11)
	suite.Require().NoError(err)

	// Verify data
	expected = dbtypes.NewCommunityPoolRow(dbtypes.NewDbDecCoins(coins), 11)
	rows = []dbtypes.CommunityPoolRow{}
	err = suite.database.Sqlx.Select(&rows, `SELECT * FROM community_pool`)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 1, "community_pool table should contain only one row")
	suite.Require().True(expected.Equals(rows[0]), "updating with higher height should modify the data")
}

func (suite *DbTestSuite) TestBigDipperDb_SaveDistributionParams() {
	distrParams := distrtypes.Params{
		CommunityTax:        math.LegacyNewDecWithPrec(2, 2),
		BaseProposerReward:  math.LegacyNewDecWithPrec(1, 2),
		BonusProposerReward: math.LegacyNewDecWithPrec(4, 2),
		WithdrawAddrEnabled: true,
	}
	err := suite.database.SaveDistributionParams(types.NewDistributionParams(distrParams, 10))
	suite.Require().NoError(err)

	var rows []dbtypes.DistributionParamsRow
	err = suite.database.Sqlx.Select(&rows, `SELECT * FROM distribution_params`)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 1)

	var stored distrtypes.Params
	err = json.Unmarshal([]byte(rows[0].Params), &stored)
	suite.Require().NoError(err)
	suite.Require().Equal(distrParams, stored)
	suite.Require().Equal(int64(10), rows[0].Height)
}

func (suite *DbTestSuite) TestBigDipperDb_SaveRewardEarned() {
	// Save the data
	reward1 := types.NewRewardEarned("cosmos1address1", sdk.NewCoin("uatom", math.NewInt(1000)), 10)
	err := suite.database.SaveRewardEarned(reward1)
	suite.Require().NoError(err)

	reward2 := types.NewRewardEarned("cosmos1address2", sdk.NewCoin("uatom", math.NewInt(2000)), 10)
	err = suite.database.SaveRewardEarned(reward2)
	suite.Require().NoError(err)

	reward3 := types.NewRewardEarned("cosmos1address1", sdk.NewCoin("stake", math.NewInt(500)), 10)
	err = suite.database.SaveRewardEarned(reward3)
	suite.Require().NoError(err)

	// Verify the data
	var rows []dbtypes.RewardEarnedRow
	err = suite.database.Sqlx.Select(&rows, `SELECT * FROM reward_earned ORDER BY delegator_address, denom`)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 3, "reward_earned table should contain three rows")

	// Check first row
	expected1 := dbtypes.NewRewardEarnedRow("cosmos1address1", "stake", "500", 10)
	suite.Require().True(expected1.Equals(rows[0]))

	// Check second row
	expected2 := dbtypes.NewRewardEarnedRow("cosmos1address1", "uatom", "1000", 10)
	suite.Require().True(expected2.Equals(rows[1]))

	// Check third row
	expected3 := dbtypes.NewRewardEarnedRow("cosmos1address2", "uatom", "2000", 10)
	suite.Require().True(expected3.Equals(rows[2]))

	// ----------------------------------------------------------------------------------------------------------------

	// Test updating existing reward (same delegator, denom, and height)
	updatedReward := types.NewRewardEarned("cosmos1address1", sdk.NewCoin("uatom", math.NewInt(1500)), 10)
	err = suite.database.SaveRewardEarned(updatedReward)
	suite.Require().NoError(err)

	// Verify the data was updated
	rows = []dbtypes.RewardEarnedRow{}
	err = suite.database.Sqlx.Select(&rows, `SELECT * FROM reward_earned WHERE delegator_address = 'cosmos1address1' AND denom = 'uatom'`)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 1, "should have one updated row")

	expectedUpdated := dbtypes.NewRewardEarnedRow("cosmos1address1", "uatom", "1500", 10)
	suite.Require().True(expectedUpdated.Equals(rows[0]))

	// ----------------------------------------------------------------------------------------------------------------

	// Test saving additional reward at different height
	additionalReward := types.NewRewardEarned("cosmos1address3", sdk.NewCoin("uatom", math.NewInt(3000)), 20)
	err = suite.database.SaveRewardEarned(additionalReward)
	suite.Require().NoError(err)

	// Verify the data
	rows = []dbtypes.RewardEarnedRow{}
	err = suite.database.Sqlx.Select(&rows, `SELECT * FROM reward_earned ORDER BY height, delegator_address, denom`)
	suite.Require().NoError(err)
	suite.Require().Len(rows, 4, "reward_earned table should contain four rows")

	// Check the new row
	expected4 := dbtypes.NewRewardEarnedRow("cosmos1address3", "uatom", "3000", 20)
	suite.Require().True(expected4.Equals(rows[3]))
}

func (suite *DbTestSuite) TestBigDipperDb_GetRewardEarned() {
	// First, save some test data
	reward1 := types.NewRewardEarned("cosmos1address1", sdk.NewCoin("uatom", math.NewInt(1000)), 10)
	err := suite.database.SaveRewardEarned(reward1)
	suite.Require().NoError(err)

	reward2 := types.NewRewardEarned("cosmos1address1", sdk.NewCoin("stake", math.NewInt(500)), 10)
	err = suite.database.SaveRewardEarned(reward2)
	suite.Require().NoError(err)

	reward3 := types.NewRewardEarned("cosmos1address2", sdk.NewCoin("uatom", math.NewInt(2000)), 20)
	err = suite.database.SaveRewardEarned(reward3)
	suite.Require().NoError(err)

	// Test GetRewardEarned - get specific reward
	retrievedReward, err := suite.database.GetRewardEarned("cosmos1address1", "uatom", 10)
	suite.Require().NoError(err)
	suite.Require().NotNil(retrievedReward)
	suite.Require().Equal("cosmos1address1", retrievedReward.DelegatorAddress)
	suite.Require().Equal("uatom", retrievedReward.Coin.Denom)
	suite.Require().Equal("1000", retrievedReward.Coin.Amount.String())
	suite.Require().Equal(int64(10), retrievedReward.Height)

	// Test GetRewardEarnedByDelegator - get reward for a delegator
	delegatorReward, err := suite.database.GetRewardEarnedByDelegator("cosmos1address1")
	suite.Require().NoError(err)
	suite.Require().NotNil(delegatorReward)
	suite.Require().Equal("cosmos1address1", delegatorReward.DelegatorAddress)
	// Should return one of the rewards for this delegator
	suite.Require().True(delegatorReward.Coin.Denom == "uatom" || delegatorReward.Coin.Denom == "stake")
}
