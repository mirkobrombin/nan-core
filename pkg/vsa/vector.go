package vsa

import (
	"encoding/binary"
	"errors"
)

type Vector struct {
	words []uint64
	bits  int
}

func NewZero(bits int) (Vector, error) {
	if bits <= 0 {
		return Vector{}, errors.New("bits must be > 0")
	}
	words := (bits + 63) / 64
	return Vector{words: make([]uint64, words), bits: bits}, nil
}

// NewRandom returns a deterministic pseudo-random vector for a given seed.
// It is NOT cryptographically secure.
func NewRandom(bits int, seed uint64) (Vector, error) {
	v, err := NewZero(bits)
	if err != nil {
		return Vector{}, err
	}

	x := seed
	for i := range v.words {
		x = xorshift64star(x)
		v.words[i] = x
	}

	v.maskUnusedBits()
	return v, nil
}

func (v Vector) Bits() int { return v.bits }

func (v Vector) Words() int { return len(v.words) }

func (v Vector) clone() Vector {
	out := Vector{words: make([]uint64, len(v.words)), bits: v.bits}
	copy(out.words, v.words)
	return out
}

func (v *Vector) maskUnusedBits() {
	unused := (len(v.words) * 64) - v.bits
	if unused <= 0 {
		return
	}
	mask := uint64(0xffffffffffffffff) >> uint(unused)
	v.words[len(v.words)-1] &= mask
}

func xorshift64star(x uint64) uint64 {
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	return x * 2685821657736338717
}

func (v Vector) MarshalBinary() ([]byte, error) {
	out := make([]byte, 8+8+8*len(v.words))
	binary.LittleEndian.PutUint64(out[0:8], uint64(v.bits))
	binary.LittleEndian.PutUint64(out[8:16], uint64(len(v.words)))
	off := 16
	for _, w := range v.words {
		binary.LittleEndian.PutUint64(out[off:off+8], w)
		off += 8
	}
	return out, nil
}

func UnmarshalBinary(b []byte) (Vector, error) {
	if len(b) < 16 {
		return Vector{}, errors.New("buffer too small")
	}
	bits := int(binary.LittleEndian.Uint64(b[0:8]))
	words := int(binary.LittleEndian.Uint64(b[8:16]))
	if bits <= 0 || words <= 0 {
		return Vector{}, errors.New("invalid header")
	}
	expected := 16 + 8*words
	if len(b) != expected {
		return Vector{}, errors.New("invalid length")
	}
	v := Vector{words: make([]uint64, words), bits: bits}
	off := 16
	for i := 0; i < words; i++ {
		v.words[i] = binary.LittleEndian.Uint64(b[off : off+8])
		off += 8
	}
	v.maskUnusedBits()
	return v, nil
}
