package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type encoder struct {
	bytes.Buffer
}

func (e *encoder) uint8(value uint8) { _ = e.WriteByte(value) }
func (e *encoder) bool(value bool) {
	if value {
		e.uint8(1)
		return
	}
	e.uint8(0)
}
func (e *encoder) uint32(value uint32) { _ = binary.Write(&e.Buffer, binary.BigEndian, value) }
func (e *encoder) uint64(value uint64) { _ = binary.Write(&e.Buffer, binary.BigEndian, value) }
func (e *encoder) int64(value int64)   { _ = binary.Write(&e.Buffer, binary.BigEndian, value) }
func (e *encoder) bytes(value []byte) {
	e.uint64(uint64(len(value)))
	_, _ = e.Write(value)
}
func (e *encoder) string(value string) { e.bytes([]byte(value)) }

func hashHex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func decodeHash(value string) ([]byte, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return nil, fmt.Errorf("invalid SHA-256 hash %q", value)
	}
	return decoded, nil
}
