package vsa

import "testing"

func TestNewRandomDeterministic(t *testing.T) {
	a, err := NewRandom(1024, 123)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewRandom(1024, 123)
	if err != nil {
		t.Fatal(err)
	}

	sa, _ := a.MarshalBinary()
	sb, _ := b.MarshalBinary()
	if string(sa) != string(sb) {
		t.Fatalf("expected deterministic output for same seed")
	}
}

func TestBindXORInvertible(t *testing.T) {
	a, _ := NewRandom(1024, 1)
	b, _ := NewRandom(1024, 2)

	ab, err := BindXOR(a, b)
	if err != nil {
		t.Fatal(err)
	}
	abb, err := BindXOR(ab, b)
	if err != nil {
		t.Fatal(err)
	}

	sa, _ := a.MarshalBinary()
	sabb, _ := abb.MarshalBinary()
	if string(sa) != string(sabb) {
		t.Fatalf("expected XOR bind to be invertible")
	}
}

func TestSimilarityRangeAndSymmetry(t *testing.T) {
	a, _ := NewRandom(512, 10)
	b, _ := NewRandom(512, 11)

	s1, err := Similarity(a, b)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := Similarity(b, a)
	if err != nil {
		t.Fatal(err)
	}

	if s1 < 0 || s1 > 1 {
		t.Fatalf("similarity out of range: %v", s1)
	}
	if s1 != s2 {
		t.Fatalf("expected symmetry: %v vs %v", s1, s2)
	}
}

func TestBundleMajorityOdd(t *testing.T) {
	v1, _ := NewZero(128)
	v2, _ := NewZero(128)
	v3, _ := NewZero(128)

	v1.words[0] = 0b1
	v2.words[0] = 0b1
	v3.words[0] = 0b0

	out, err := BundleMajority([]Vector{v1, v2, v3})
	if err != nil {
		t.Fatal(err)
	}

	if (out.words[0] & 0b1) == 0 {
		t.Fatalf("expected majority bit to be set")
	}
}
