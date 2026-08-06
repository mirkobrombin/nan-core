package vsa

import (
	"errors"
	"math/bits"
)

func BindXOR(a, b Vector) (Vector, error) {
	if a.bits != b.bits || len(a.words) != len(b.words) {
		return Vector{}, errors.New("dimension mismatch")
	}
	out := a.clone()
	for i := range out.words {
		out.words[i] ^= b.words[i]
	}
	out.maskUnusedBits()
	return out, nil
}

// BundleMajority performs a per-bit majority vote over an odd number of vectors.
func BundleMajority(vs []Vector) (Vector, error) {
	if len(vs) == 0 {
		return Vector{}, errors.New("empty input")
	}
	if len(vs)%2 == 0 {
		return Vector{}, errors.New("requires odd number of vectors")
	}

	bitsCount := vs[0].bits
	wordsCount := len(vs[0].words)
	for _, v := range vs {
		if v.bits != bitsCount || len(v.words) != wordsCount {
			return Vector{}, errors.New("dimension mismatch")
		}
	}

	out, _ := NewZero(bitsCount)
	threshold := len(vs)/2 + 1

	for wi := 0; wi < wordsCount; wi++ {
		var word uint64
		for bi := 0; bi < 64; bi++ {
			bitMask := uint64(1) << uint(bi)
			count := 0
			for _, v := range vs {
				if (v.words[wi] & bitMask) != 0 {
					count++
				}
			}
			if count >= threshold {
				word |= bitMask
			}
		}
		out.words[wi] = word
	}

	out.maskUnusedBits()
	return out, nil
}

// Similarity returns normalized similarity in [0,1] based on Hamming distance.
func Similarity(a, b Vector) (float64, error) {
	if a.bits != b.bits || len(a.words) != len(b.words) {
		return 0, errors.New("dimension mismatch")
	}
	if a.bits == 0 {
		return 0, errors.New("invalid dimension")
	}

	diff := 0
	for i := range a.words {
		diff += bits.OnesCount64(a.words[i] ^ b.words[i])
	}

	return 1.0 - (float64(diff) / float64(a.bits)), nil
}
