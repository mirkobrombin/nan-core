package store

import (
	"path/filepath"
	"testing"
)

func TestSlipstreamLogAppendReplay(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ss")

	l, err := OpenSlipstreamLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	if err := l.Append(Record{Type: EventFactAdded, Payload: []byte("one")}); err != nil {
		t.Fatal(err)
	}
	if err := l.Append(Record{Type: EventFactAdded, Payload: []byte("two")}); err != nil {
		t.Fatal(err)
	}

	var got []string
	if err := l.Replay(func(rec Record) error {
		got = append(got, string(rec.Payload))
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("got=%v", got)
	}
}
