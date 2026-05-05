package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Packet struct {
	Cmd         uint64
	Seq         uint64
	Payload     []byte
	TimestampMs uint64
}

const (
	fieldCmd         = 1
	fieldSeq         = 2
	fieldPayload     = 3
	fieldTimestampMs = 4

	wireVarint          = 0
	wireLengthDelimited = 2
)

func Encode(packet Packet) []byte {
	out := make([]byte, 0, len(packet.Payload)+32)
	out = appendVarintField(out, fieldCmd, packet.Cmd)
	out = appendVarintField(out, fieldSeq, packet.Seq)
	out = appendBytesField(out, fieldPayload, packet.Payload)
	out = appendVarintField(out, fieldTimestampMs, packet.TimestampMs)
	return out
}

func Decode(data []byte) (Packet, error) {
	var packet Packet
	offset := 0

	for offset < len(data) {
		tag, next, err := readVarint(data, offset)
		if err != nil {
			return Packet{}, err
		}
		offset = next

		fieldNo := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch fieldNo {
		case fieldCmd:
			value, next, err := readVarint(data, offset)
			if err != nil {
				return Packet{}, err
			}
			packet.Cmd = value
			offset = next
		case fieldSeq:
			value, next, err := readVarint(data, offset)
			if err != nil {
				return Packet{}, err
			}
			packet.Seq = value
			offset = next
		case fieldPayload:
			length, next, err := readVarint(data, offset)
			if err != nil {
				return Packet{}, err
			}
			end := next + int(length)
			if end > len(data) {
				return Packet{}, errors.New("packet payload exceeds frame length")
			}
			packet.Payload = append([]byte(nil), data[next:end]...)
			offset = end
		case fieldTimestampMs:
			value, next, err := readVarint(data, offset)
			if err != nil {
				return Packet{}, err
			}
			packet.TimestampMs = value
			offset = next
		default:
			next, err := skipUnknown(data, wireType, offset)
			if err != nil {
				return Packet{}, err
			}
			offset = next
		}
	}

	return packet, nil
}

func appendVarintField(out []byte, fieldNo int, value uint64) []byte {
	out = binary.AppendUvarint(out, uint64(fieldNo<<3|wireVarint))
	out = binary.AppendUvarint(out, value)
	return out
}

func appendBytesField(out []byte, fieldNo int, value []byte) []byte {
	out = binary.AppendUvarint(out, uint64(fieldNo<<3|wireLengthDelimited))
	out = binary.AppendUvarint(out, uint64(len(value)))
	out = append(out, value...)
	return out
}

func readVarint(data []byte, offset int) (uint64, int, error) {
	value, size := binary.Uvarint(data[offset:])
	if size <= 0 {
		return 0, offset, errors.New("invalid varint")
	}
	return value, offset + size, nil
}

func skipUnknown(data []byte, wireType int, offset int) (int, error) {
	switch wireType {
	case wireVarint:
		_, next, err := readVarint(data, offset)
		return next, err
	case wireLengthDelimited:
		length, next, err := readVarint(data, offset)
		if err != nil {
			return offset, err
		}
		end := next + int(length)
		if end > len(data) {
			return offset, errors.New("unknown length-delimited field exceeds frame length")
		}
		return end, nil
	default:
		return offset, fmt.Errorf("unsupported wire type %d", wireType)
	}
}
