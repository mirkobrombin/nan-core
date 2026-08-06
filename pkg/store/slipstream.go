package store

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/mirkobrombin/go-slipstream/pkg/engine"
	sswal "github.com/mirkobrombin/go-slipstream/pkg/wal"
)

const slipstreamSeqKey = "__nan_seq"

type SlipstreamLog struct {
	mu sync.Mutex

	w  *sswal.Manager
	db *engine.Bitcask[[]byte]
}

func OpenSlipstreamLog(dir string) (*SlipstreamLog, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	w, err := sswal.NewManager(dir)
	if err != nil {
		return nil, err
	}

	codec := func(v []byte) ([]byte, error) { return v, nil }
	decoder := func(b []byte) ([]byte, error) { return b, nil }
	db := engine.NewBitcask[[]byte](w, codec, decoder)
	if err := db.Engine().Recover(); err != nil {
		_ = db.Close()
		_ = w.Close()
		return nil, err
	}

	return &SlipstreamLog{w: w, db: db}, nil
}

func (s *SlipstreamLog) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err1, err2 error
	if s.db != nil {
		err1 = s.db.Close()
		s.db = nil
	}
	if s.w != nil {
		err2 = s.w.Close()
		s.w = nil
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func (s *SlipstreamLog) Append(rec Record) error {
	payload, err := encodeRecord(rec)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("slipstream log closed")
	}

	ctx := context.Background()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	seq := uint64(0)
	seqBytes, err := tx.Get(ctx, slipstreamSeqKey)
	if err == nil {
		if len(seqBytes) != 8 {
			_ = tx.Rollback()
			return errors.New("invalid seq value")
		}
		seq = binary.LittleEndian.Uint64(seqBytes)
	} else {
		// Slipstream currently returns a plain "engine: not found" error for missing keys.
		if !errors.Is(err, engine.ErrKeyNotFound) && err.Error() != "engine: not found" {
			_ = tx.Rollback()
			return err
		}
	}

	seq++
	key := fmt.Sprintf("%020d", seq)

	newSeq := make([]byte, 8)
	binary.LittleEndian.PutUint64(newSeq, seq)

	if err := tx.Put(ctx, key, payload, 0*time.Second); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Put(ctx, slipstreamSeqKey, newSeq, 0*time.Second); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return nil
}

func (s *SlipstreamLog) Replay(fn ReplayFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return errors.New("slipstream log closed")
	}

	keys, err := s.db.Engine().Keys()
	if err != nil {
		return err
	}
	sort.Strings(keys)

	ctx := context.Background()
	for _, k := range keys {
		if k == slipstreamSeqKey {
			continue
		}
		val, err := s.db.Get(ctx, k)
		if err != nil {
			return err
		}
		rec, err := decodeRecord(val)
		if err != nil {
			return err
		}
		if fn != nil {
			if err := fn(rec); err != nil {
				return err
			}
		}
	}
	return nil
}
