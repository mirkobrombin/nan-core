package store

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

func encodeRecord(rec Record) ([]byte, error) {
	if rec.Type == EventUnknown {
		return nil, errors.New("invalid event type")
	}
	out := make([]byte, 1+4+len(rec.Payload))
	out[0] = byte(rec.Type)
	binary.LittleEndian.PutUint32(out[1:5], crc32.ChecksumIEEE(rec.Payload))
	copy(out[5:], rec.Payload)
	return out, nil
}

func decodeRecord(b []byte) (Record, error) {
	if len(b) < 5 {
		return Record{}, errors.New("buffer too small")
	}
	rec := Record{Type: EventType(b[0]), Payload: append([]byte(nil), b[5:]...)}
	want := binary.LittleEndian.Uint32(b[1:5])
	if crc32.ChecksumIEEE(rec.Payload) != want {
		return Record{}, errors.New("crc mismatch")
	}
	if rec.Type == EventUnknown {
		return Record{}, errors.New("invalid event type")
	}
	return rec, nil
}
