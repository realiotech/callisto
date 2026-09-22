package utils

import (
	"context"
	"strconv"

	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"google.golang.org/grpc/metadata"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RemoveDuplicateValues removes the duplicated values from the given slice. It always returns a
// non-nil slice (empty when the input has no entries) because the result is written straight into
// the NOT NULL message.involved_accounts_addresses column - a nil []string binds as SQL NULL via
// the pq driver, while an empty []string{} correctly binds as '{}'. Messages whose events carry no
// bech32 addresses (e.g. MsgEthereumTx, which uses hex addresses) used to hit this and fail to save.
func RemoveDuplicateValues(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// GetHeightRequestContext adds the height to the context for queries
func GetHeightRequestContext(context context.Context, height int64) context.Context {
	return metadata.AppendToOutgoingContext(
		context,
		grpctypes.GRPCBlockHeightHeader,
		strconv.FormatInt(height, 10),
	)
}

// ConvertValidatorPubKeyToBech32String converts the given pubKey to a Bech32 string
func ConvertValidatorPubKeyToBech32String(pubKey cryptotypes.PubKey) (string, error) {
	bech32Prefix := sdk.GetConfig().GetBech32ConsensusPubPrefix()
	return bech32.ConvertAndEncode(bech32Prefix, pubKey.Bytes())
}
