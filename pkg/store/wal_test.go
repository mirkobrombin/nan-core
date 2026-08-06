package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWALAppendReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nan.wal")

	w, err := OpenWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if err := w.Append(Record{Type: EventFactAdded, Payload: []byte("one")}); err != nil {
		t.Fatal(err)
	}
	if err := w.Append(Record{Type: EventFactAdded, Payload: []byte("two")}); err != nil {
		t.Fatal(err)
	}

	var got []string
	if err := w.Replay(func(rec Record) error {
		got = append(got, string(rec.Payload))
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("got=%v", got)
	}
}

func TestWALPartialTailStopsCleanly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nan.wal")

	w, err := OpenWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Append(Record{Type: EventFactAdded, Payload: []byte("ok")}); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	// Corrupt by truncating last bytes.
	fi, _ := os.Stat(path)
	if err := os.Truncate(path, fi.Size()-2); err != nil {
		t.Fatal(err)
	}

	w2, err := OpenWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()

	count := 0
	if err := w2.Replay(func(rec Record) error {
		count++
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Depending on where truncation hits, replay may yield 0 records, but must not error.
	if count < 0 || count > 1 {
		t.Fatalf("unexpected count=%d", count)
	}
}
