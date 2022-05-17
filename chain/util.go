package chain

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/minio/blake2b-simd"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/types/wallet"
)

//var log = logging.Logger("API")

type WinningPoStProver interface {
	GenerateCandidates(context.Context, abi.PoStRandomness, uint64) ([]uint64, error)
	ComputeProof(context.Context, []builtin.ExtendedSectorInfo, abi.PoStRandomness, abi.ChainEpoch, network.Version) ([]builtin.PoStProof, error)
}

type SignFunc func(ctx context.Context, account string, signer address.Address, toSign []byte, meta types.MsgMeta) (*crypto.Signature, error)

// type SignFunc func(context.Context, address.Address, []byte) (*crypto.Signature, error)

func DrawRandomness(rbase []byte, pers crypto.DomainSeparationTag, round abi.ChainEpoch, entropy []byte) ([]byte, error) {
	h := blake2b.New256()
	if err := binary.Write(h, binary.BigEndian, int64(pers)); err != nil {
		return nil, fmt.Errorf("deriving randomness: %w", err)
	}
	VRFDigest := blake2b.Sum256(rbase)
	_, err := h.Write(VRFDigest[:])
	if err != nil {
		return nil, fmt.Errorf("hashing VRFDigest: %w", err)
	}
	if err := binary.Write(h, binary.BigEndian, round); err != nil {
		return nil, fmt.Errorf("deriving randomness: %w", err)
	}
	_, err = h.Write(entropy)
	if err != nil {
		return nil, fmt.Errorf("hashing entropy: %w", err)
	}

	return h.Sum(nil), nil
}

func ComputeVRF(ctx context.Context, sign SignFunc, account string, worker address.Address, sigInput []byte) ([]byte, error) {
	// log.Infof("sigInput: %s", hex.EncodeToString(sigInput))
	sig, err := sign(ctx, account, worker, sigInput, types.MsgMeta{Type: types.MTDrawRandomParam})
	if err != nil {
		return nil, err
	}

	if sig.Type != crypto.SigTypeBLS {
		return nil, fmt.Errorf("miner worker address was not a BLS key")
	}

	return sig.Data, nil
}

func IsRoundWinner(ctx context.Context, round abi.ChainEpoch, account string,
	miner address.Address, brand types.BeaconEntry, mbi *types.MiningBaseInfo, sign SignFunc) (*types.ElectionProof, error) {

	buf := new(bytes.Buffer)
	if err := miner.MarshalCBOR(buf); err != nil {
		return nil, fmt.Errorf("failed to cbor marshal address: %w", err)
	}

	electionRand := new(bytes.Buffer)
	drp := &wallet.DrawRandomParams{
		Rbase:   brand.Data,
		Pers:    crypto.DomainSeparationTag_ElectionProofProduction,
		Round:   round,
		Entropy: buf.Bytes(),
	}
	err := drp.MarshalCBOR(electionRand)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal randomness: %w", err)
	}
	//electionRand, err := DrawRandomness(brand.Data, crypto.DomainSeparationTag_ElectionProofProduction, round, buf.Bytes())
	//if err != nil {
	//	return nil, fmt.Errorf("failed to draw randomness: %w", err)
	//}

	vrfout, err := ComputeVRF(ctx, sign, account, mbi.WorkerKey, electionRand.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to compute VRF: %w", err)
	}

	ep := &types.ElectionProof{VRFProof: vrfout}
	j := ep.ComputeWinCount(mbi.MinerPower, mbi.NetworkPower)
	ep.WinCount = j
	if j < 1 {
		return nil, nil
	}

	return ep, nil
}
