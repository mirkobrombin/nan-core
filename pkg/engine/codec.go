package engine

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
)

// Binary encoding (little-endian):
// - fromLen u16 + from bytes
// - predLen u16 + pred bytes
// - toLen u16 + to bytes
// - polarity u8
// - sourceLen u16 + source bytes
// - unixNano i64
//
// Used for both beliefs and resolutions.

func encodeBelief(b belief.Belief) ([]byte, error) {
	from := []byte(b.Atom.From)
	pred := []byte(b.Atom.Predicate)
	to := []byte(b.Atom.To)
	src := []byte(b.Source)

	if len(from) > 0xffff || len(pred) > 0xffff || len(to) > 0xffff || len(src) > 0xffff {
		return nil, errors.New("field too large")
	}

	buf := bytes.NewBuffer(make([]byte, 0, 32+len(from)+len(pred)+len(to)+len(src)))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(from)))
	buf.Write(from)
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(pred)))
	buf.Write(pred)
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(to)))
	buf.Write(to)
	buf.WriteByte(byte(b.Polarity))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(src)))
	buf.Write(src)
	_ = binary.Write(buf, binary.LittleEndian, int64(b.UnixNano))
	return buf.Bytes(), nil
}

func decodeBelief(p []byte) (belief.Belief, error) {
	r := bytes.NewReader(p)

	readBytes := func() ([]byte, error) {
		var ln uint16
		if err := binary.Read(r, binary.LittleEndian, &ln); err != nil {
			return nil, err
		}
		b := make([]byte, ln)
		if _, err := r.Read(b); err != nil {
			return nil, err
		}
		return b, nil
	}

	from, err := readBytes()
	if err != nil {
		return belief.Belief{}, err
	}
	pred, err := readBytes()
	if err != nil {
		return belief.Belief{}, err
	}
	to, err := readBytes()
	if err != nil {
		return belief.Belief{}, err
	}

	pol, err := r.ReadByte()
	if err != nil {
		return belief.Belief{}, err
	}

	src, err := readBytes()
	if err != nil {
		return belief.Belief{}, err
	}

	var unixNano int64
	if err := binary.Read(r, binary.LittleEndian, &unixNano); err != nil {
		return belief.Belief{}, err
	}

	if r.Len() != 0 {
		return belief.Belief{}, errors.New("trailing bytes")
	}

	return belief.Belief{
		Atom: belief.Atom{
			From:      kg.NodeID(string(from)),
			Predicate: string(pred),
			To:        kg.NodeID(string(to)),
		},
		Polarity: belief.Polarity(pol),
		Source:   string(src),
		UnixNano: unixNano,
	}, nil
}

func encodeResolution(r belief.Resolution) ([]byte, error) {
	return encodeBelief(belief.Belief{
		Atom:     r.Atom,
		Polarity: r.Polarity,
		Source:   r.Source,
		UnixNano: r.UnixNano,
	})
}

func decodeResolution(p []byte) (belief.Resolution, error) {
	b, err := decodeBelief(p)
	if err != nil {
		return belief.Resolution{}, err
	}
	return belief.Resolution{Atom: b.Atom, Polarity: b.Polarity, Source: b.Source, UnixNano: b.UnixNano}, nil
}
