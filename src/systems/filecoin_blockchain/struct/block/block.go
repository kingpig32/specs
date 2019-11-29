package block

import (
	filcrypto "github.com/filecoin-project/specs/algorithms/crypto"
	util "github.com/filecoin-project/specs/util"
)

func SmallerBytes(a, b util.Bytes) util.Bytes {
	if util.CompareBytesStrict(a, b) > 0 {
		return b
	}
	return a
}

// TODO: add SHA256 to filcrypto
// TODO: import SHA256 from filcrypto
func SHA256(input util.Bytes) util.Bytes {
	ret := make([]byte, 0)
	return ret
}

func sliceEqual(a util.Bytes, b util.Bytes) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// will return tipset from closest prior (or equal) epoch with a tipset
// return epoch should be checked accordingly
func (chain *Chain_I) TipsetAtEpoch(epoch ChainEpoch) Tipset {

	dist := chain.HeadEpoch() - epoch
	current := chain.HeadTipset()
	parents := current.Parents()
	for i := 0; i < int(dist); i++ {
		current = parents
		parents = current.Parents()
	}

	return current
}

func (chain *Chain_I) RandomnessAtEpoch(epoch ChainEpoch) util.Bytes {
	ts := chain.TipsetAtEpoch(epoch)
	priorRand := ts.MinTicket().DrawRandomness(epoch)
	// doesn't matter if ts.Epoch() != epoch
	// since we generate new ticket from prior one in any case
	// else we use ticket from that epoch and derive new randomness from it

	var input []byte
	input = append(input, priorRand...)
	input = append(input, byte(filcrypto.InputDelimeter_Case_Bytes))
	input = append(input, byte(epoch))
	return SHA256(input)
}

func (chain *Chain_I) HeadEpoch() ChainEpoch {
	panic("")
}

func (chain *Chain_I) HeadTipset() Tipset {
	panic("")
}
