package vsa

import "testing"

func TestSpaceEncodeDeterministic(t *testing.T) {
	s := NewSpace(1024)
	v1 := s.Encode("alice")
	v2 := s.Encode("alice")
	sim, _ := Similarity(v1, v2)
	if sim != 1.0 {
		t.Fatalf("same symbol must produce identical vector, sim=%f", sim)
	}
}

func TestSpaceEncodeDifferentSymbols(t *testing.T) {
	s := NewSpace(10000)
	v1 := s.Encode("alice")
	v2 := s.Encode("bob")
	sim, _ := Similarity(v1, v2)
	// Random vectors in high-dim space should be ~0.5 similar
	if sim < 0.45 || sim > 0.55 {
		t.Fatalf("unrelated symbols similarity should be ~0.5, got %f", sim)
	}
}

func TestSpaceEncodeTupleDeterministic(t *testing.T) {
	s := NewSpace(10000)
	v1 := s.EncodeTuple("alice", "likes", "bob")
	v2 := s.EncodeTuple("alice", "likes", "bob")
	sim, _ := Similarity(v1, v2)
	if sim != 1.0 {
		t.Fatalf("same tuple must produce identical vector, sim=%f", sim)
	}
}

func TestSpaceEncodeTupleDifferent(t *testing.T) {
	s := NewSpace(10000)
	v1 := s.EncodeTuple("alice", "likes", "bob")
	v2 := s.EncodeTuple("alice", "hates", "bob")
	sim, _ := Similarity(v1, v2)
	// Different predicates should be distinguishable
	if sim > 0.7 {
		t.Fatalf("different tuples should be distinguishable, sim=%f", sim)
	}
}

func TestSpaceNearest(t *testing.T) {
	s := NewSpace(10000)
	s.Encode("cat")
	s.Encode("dog")
	s.Encode("car")

	query := s.Encode("cat")
	matches := s.Nearest(query, 3)
	if len(matches) == 0 {
		t.Fatal("no matches")
	}
	if matches[0].Symbol != "cat" || matches[0].Similarity != 1.0 {
		t.Fatalf("first match should be exact, got %+v", matches[0])
	}
}

func BenchmarkEncodeTuple(b *testing.B) {
	s := NewSpace(10000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.EncodeTuple("alice", "likes", "bob")
	}
}
