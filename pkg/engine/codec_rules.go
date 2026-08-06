package engine

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/mirkobrombin/nan-core/pkg/rules"
)

func encodeRule(r rules.Rule) ([]byte, error) {
	name := []byte(r.Name)
	if len(name) > 0xffff {
		return nil, errors.New("name too large")
	}
	if len(r.If) > 0xffff {
		return nil, errors.New("too many premises")
	}

	buf := bytes.NewBuffer(make([]byte, 0, 64))
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(name)))
	buf.Write(name)
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(r.If)))
	for _, p := range r.If {
		if err := writePattern(buf, p); err != nil {
			return nil, err
		}
	}
	if err := writePattern(buf, r.Then); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeRule(p []byte) (rules.Rule, error) {
	r := bytes.NewReader(p)

	var nameLen uint16
	if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
		return rules.Rule{}, err
	}
	name := make([]byte, nameLen)
	if _, err := r.Read(name); err != nil {
		return rules.Rule{}, err
	}

	var premCount uint16
	if err := binary.Read(r, binary.LittleEndian, &premCount); err != nil {
		return rules.Rule{}, err
	}

	prem := make([]rules.AtomPattern, 0, premCount)
	for i := 0; i < int(premCount); i++ {
		ap, err := readPattern(r)
		if err != nil {
			return rules.Rule{}, err
		}
		prem = append(prem, ap)
	}

	then, err := readPattern(r)
	if err != nil {
		return rules.Rule{}, err
	}

	if r.Len() != 0 {
		return rules.Rule{}, errors.New("trailing bytes")
	}

	return rules.Rule{Name: string(name), If: prem, Then: then}, nil
}

func writePattern(buf *bytes.Buffer, p rules.AtomPattern) error {
	from := []byte(p.From)
	pred := []byte(p.Predicate)
	to := []byte(p.To)
	if len(from) > 0xffff || len(pred) > 0xffff || len(to) > 0xffff {
		return errors.New("field too large")
	}
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(from)))
	buf.Write(from)
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(pred)))
	buf.Write(pred)
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(to)))
	buf.Write(to)
	return nil
}

func readPattern(r *bytes.Reader) (rules.AtomPattern, error) {
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
		return rules.AtomPattern{}, err
	}
	pred, err := readBytes()
	if err != nil {
		return rules.AtomPattern{}, err
	}
	to, err := readBytes()
	if err != nil {
		return rules.AtomPattern{}, err
	}
	return rules.AtomPattern{From: string(from), Predicate: string(pred), To: string(to)}, nil
}
