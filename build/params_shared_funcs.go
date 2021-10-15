package build

import (
	"fmt"
	"os"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/filecoin-project/venus/pkg/types/specactors/policy"
)

func SetAddressNetwork(n address.Network) {
	address.CurrentNetwork = n
}

func InitNetWorkParams(nettype string) error {
	fmt.Println("nettype: ", nettype)
	switch nettype {
	case "mainnet":
		policy.SetConsensusMinerMinPower(abi.NewStoragePower(10 << 40))

		if os.Getenv("MINING_USE_TEST_ADDRESSES") != "1" {
			SetAddressNetwork(address.Mainnet)
		}

		Devnet = false

		BuildType = BuildMainnet

		UpgradeSmokeHeight = 51000
		UpgradeOrangeHeight = 336458
		BlockDelaySecs = uint64(builtin2.EpochDurationSeconds)
		PropagationDelaySecs = uint64(12)
	case "debug", "2k":
		{
			policy.SetSupportedProofTypes(abi.RegisteredSealProof_StackedDrg2KiBV1)
			policy.SetConsensusMinerMinPower(abi.NewStoragePower(2048))
			policy.SetMinVerifiedDealSize(abi.NewStoragePower(256))

			SetAddressNetwork(address.Testnet)

			switch nettype {
			case "debug":
				BuildType |= BuildDebug
			case "2k":
				BuildType |= Build2k
			}

			UpgradeSmokeHeight = -1
			UpgradeOrangeHeight = -11
			BlockDelaySecs = uint64(4)
			PropagationDelaySecs = uint64(1)
		}
	case "calibnet":
		policy.SetConsensusMinerMinPower(abi.NewStoragePower(10 << 30))
		policy.SetSupportedProofTypes(
			abi.RegisteredSealProof_StackedDrg512MiBV1,
			abi.RegisteredSealProof_StackedDrg32GiBV1,
			abi.RegisteredSealProof_StackedDrg64GiBV1,
		)

		SetAddressNetwork(address.Testnet)

		Devnet = true

		BuildType = BuildCalibnet

		UpgradeSmokeHeight = -2
		UpgradeOrangeHeight = 300
		BlockDelaySecs = uint64(builtin2.EpochDurationSeconds)
		PropagationDelaySecs = uint64(12)
	default:
		return fmt.Errorf("unknown nettype %s", nettype)
	}

	return nil
}

func MustParseAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
