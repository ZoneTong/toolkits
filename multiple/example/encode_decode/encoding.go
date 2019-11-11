package main

import (
	"bytes"
	"encoding/binary"

	"v2ray.com/core/common/serial"
)

const (
	PROTOCOL   = "newudp"
	HEADER_LEN = uint32(1 + len(PROTOCOL) + 4 + 4)
)

func Uint32ToBytes(v uint32) (bs []byte) {
	bs = make([]byte, 4)
	binary.BigEndian.PutUint32(bs, v)
	return
}

func BytesToUint32(bs []byte) uint32 {
	return binary.BigEndian.Uint32(bs)
}

type MultipleEncoder struct {
	sequence uint32
}

// \0xmess(6) reqid(4) seq(4) length(4) data(length)
func (c *MultipleEncoder) Encode(m []byte) []byte {
	// return m

	c.sequence++
	if c.sequence == 0 {
		c.sequence = 1
	}
	header := bytes.NewBuffer(nil)
	header.WriteByte(0x0)
	header.Write([]byte(PROTOCOL))

	header.Write(Uint32ToBytes(c.sequence))
	header.Write(Uint32ToBytes(uint32(len(m))))

	newbuf := bytes.NewBuffer(header.Bytes())
	newbuf.Write(m)
	return newbuf.Bytes()
}

type MultipleDecoder struct {
	m               []byte
	recved_sequence uint32
}

func (c *MultipleDecoder) Decode(m []byte) []byte {
	// return m

	c.m = append(c.m, m...)
	if len(c.m) < int(HEADER_LEN) {
		return nil
	}

	header := make([]byte, HEADER_LEN)
	copy(header, c.m)
	if header[0] != 0 || string(header[1:1+len(PROTOCOL)]) != PROTOCOL {
		c.m = nil
		return nil
	}

	seq := serial.BytesToUint32(header[HEADER_LEN-8 : HEADER_LEN-4])
	length := serial.BytesToUint32(header[HEADER_LEN-4:])
	size := (HEADER_LEN) + (length)
	if size > uint32(len(c.m)) {
		return nil
	}

	newbuf := c.m[:size]
	c.m = c.m[size:]
	if seq <= c.recved_sequence {
		return nil
	}

	c.recved_sequence = seq
	if c.recved_sequence == 0xFFFFFFFF {
		c.recved_sequence = 0
	}
	newbuf = newbuf[HEADER_LEN:]
	return newbuf
}
