package store

import (
	"bufio"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"sync"
)

const (
	walMagic   = "NAN1"
	walVersion = uint8(1)

	walHeaderSize = 16
)

type EventType uint8

const (
	EventUnknown       EventType = 0
	EventFactAdded     EventType = 1
	EventResolutionSet EventType = 2
	EventRuleAdded     EventType = 3
)

type Record struct {
	Type    EventType
	Payload []byte
}

type WAL struct {
	mu        sync.Mutex
	path      string
	f         *os.File
	deferSync bool
}

func OpenWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &WAL{path: path, f: f}, nil
}

func CreateWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &WAL{path: path, f: f}, nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return nil
	}
	err := w.f.Close()
	w.f = nil
	return err
}

// SetDeferSync enables/disables deferred sync mode for bulk imports.
// When enabled, Append skips fsync per record. Call Flush() when done.
func (w *WAL) SetDeferSync(on bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.deferSync = on
}

// Flush forces an fsync on the WAL file.
func (w *WAL) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.f == nil {
		return nil
	}
	return w.f.Sync()
}

func (w *WAL) Append(rec Record) error {
	if rec.Type == EventUnknown {
		return errors.New("invalid event type")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.f == nil {
		return errors.New("wal closed")
	}
	if len(rec.Payload) > int(^uint32(0)) {
		return errors.New("payload too large")
	}

	header := make([]byte, walHeaderSize)
	copy(header[0:4], []byte(walMagic))
	header[4] = walVersion
	header[5] = byte(rec.Type)
	binary.LittleEndian.PutUint16(header[6:8], 0)
	binary.LittleEndian.PutUint32(header[8:12], uint32(len(rec.Payload)))
	binary.LittleEndian.PutUint32(header[12:16], crc32.ChecksumIEEE(rec.Payload))

	if _, err := w.f.Write(header); err != nil {
		return err
	}
	if _, err := w.f.Write(rec.Payload); err != nil {
		return err
	}
	if w.deferSync {
		return nil
	}
	return w.f.Sync()
}

type ReplayFunc func(rec Record) error

func (w *WAL) Replay(fn ReplayFunc) error {
	f, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 1<<20)
	for {
		header := make([]byte, walHeaderSize)
		_, err := io.ReadFull(r, header)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}

		if string(header[0:4]) != walMagic {
			return errors.New("invalid magic")
		}
		if header[4] != walVersion {
			return errors.New("unsupported version")
		}
		et := EventType(header[5])
		ln := binary.LittleEndian.Uint32(header[8:12])
		wantCRC := binary.LittleEndian.Uint32(header[12:16])

		payload := make([]byte, ln)
		_, err = io.ReadFull(r, payload)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}
		if crc32.ChecksumIEEE(payload) != wantCRC {
			return errors.New("crc mismatch")
		}
		if fn != nil {
			if err := fn(Record{Type: et, Payload: payload}); err != nil {
				return err
			}
		}
	}
}
