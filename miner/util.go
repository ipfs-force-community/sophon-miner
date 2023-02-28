package miner

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/minio/blake2b-simd"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/types/wallet"
)

type WinningPoStProver interface {
	GenerateCandidates(context.Context, abi.PoStRandomness, uint64) ([]uint64, error)
	ComputeProof(context.Context, []builtin.ExtendedSectorInfo, abi.PoStRandomness, abi.ChainEpoch, network.Version) ([]builtin.PoStProof, error)
}

type SignFunc func(ctx context.Context, signer address.Address, accounts []string, toSign []byte, meta types.MsgMeta) (*crypto.Signature, error)

func ComputeVRF(ctx context.Context, sign SignFunc, account string, worker address.Address, sigInput []byte) ([]byte, error) {
	sig, err := sign(ctx, worker, []string{account}, sigInput, types.MsgMeta{Type: types.MTDrawRandomParam})
	if err != nil {
		return nil, err
	}

	if sig.Type != crypto.SigTypeBLS {
		return nil, fmt.Errorf("miner worker address was not a BLS key")
	}

	return sig.Data, nil
}

func IsRoundWinner(
	ctx context.Context,
	round abi.ChainEpoch,
	account string,
	miner address.Address,
	brand types.BeaconEntry,
	mbi *types.MiningBaseInfo,
	sign SignFunc) (*types.ElectionProof, error) {

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
	//electionRand, err := chain.DrawRandomness(brand.Data, crypto.DomainSeparationTag_ElectionProofProduction, round, buf.Bytes())
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

// DrawRandomness todo 在venus处理好后,这里的删除,用venus中的函数
func DrawRandomness(rbase []byte, pers crypto.DomainSeparationTag, round abi.ChainEpoch, entropy []byte) ([]byte, error) {
	h := blake2b.New256()
	if err := binary.Write(h, binary.BigEndian, int64(pers)); err != nil {
		return nil, fmt.Errorf("deriving randomness: %s", err)
	}
	VRFDigest := blake2b.Sum256(rbase)
	_, err := h.Write(VRFDigest[:])
	if err != nil {
		return nil, fmt.Errorf("hashing VRFDigest: %s", err)
	}
	if err := binary.Write(h, binary.BigEndian, round); err != nil {
		return nil, fmt.Errorf("deriving randomness: %s", err)
	}
	_, err = h.Write(entropy)
	if err != nil {
		return nil, fmt.Errorf("hashing entropy: %s", err)
	}

	return h.Sum(nil), nil
}

// ReorgOps takes two tipsets (which can be at different heights), and walks
// their corresponding chains backwards one step at a time until we find
// a common ancestor. It then returns the respective chain segments that fork
// from the identified ancestor, in reverse order, where the first element of
// each slice is the supplied tipset, and the last element is the common
// ancestor.
//
// If an error happens along the way, we return the error with nil slices.
// todo should move this code into store.ReorgOps. anywhere use this function should invoke store.ReorgOps
// todo 因依赖filecoin-ffi,暂时从venus复制的,venus处理好依赖后删除,用venus中的
func ReorgOps(lts func(context.Context, types.TipSetKey) (*types.TipSet, error), a, b *types.TipSet) ([]*types.TipSet, []*types.TipSet, error) {
	left := a
	right := b

	var leftChain, rightChain []*types.TipSet
	for !left.Equals(right) {
		if left.Height() > right.Height() {
			leftChain = append(leftChain, left)
			par, err := lts(context.TODO(), left.Parents())
			if err != nil {
				return nil, nil, err
			}

			left = par
		} else {
			rightChain = append(rightChain, right)
			par, err := lts(context.TODO(), right.Parents())
			if err != nil {
				log.Infof("failed to fetch right.Parents: %s", err)
				return nil, nil, err
			}

			right = par
		}
	}

	return leftChain, rightChain, nil
}
