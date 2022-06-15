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

// Blocks (e)
var BlocksPerEpoch = uint64(builtin2.ExpectedLeadersPerEpoch)

// Epochs
const Finality = policy.ChainFinality

// /////
// Mining

// Epochs
const TicketRandomnessLookback = abi.ChainEpoch(1)

// /////
// Address

const AddressMainnetEnvVar = "_mainnet_"

// the 'f' prefix doesn't matter
var ZeroAddress = MustParseAddress("f3yaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaby2smx7a")

// /////
// Devnet settings

var Devnet = true

const FilBase = uint64(2_000_000_000)

const FilecoinPrecision = uint64(1_000_000_000_000_000_000)

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

// ///////
// Limits

const BlockGasLimit = 10_000_000_000
