//go:build !testground
// +build !testground

package build

import (
	"os"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/filecoin-project/venus/venus-shared/actors/policy"
)

// Epochs
const Finality = policy.ChainFinality

// Epochs
const TicketRandomnessLookback = abi.ChainEpoch(1)

// /////
// Address

const AddressMainnetEnvVar = "_mainnet_"

// Build Settings
var (
	UpgradeSmokeHeight  abi.ChainEpoch = -1
	UpgradeOrangeHeight abi.ChainEpoch = 336458

	BlockDelaySecs = uint64(builtin2.EpochDurationSeconds)

	PropagationDelaySecs = uint64(12)
)

func init() {
	if os.Getenv("MINING_USE_TEST_ADDRESSES") != "1" || os.Getenv("VENUS_ADDRESS_TYPE") == AddressMainnetEnvVar {
		SetAddressNetwork(address.Mainnet)
	}
}
