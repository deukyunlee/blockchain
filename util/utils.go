package util

import "encoding/binary"

func IntToHex(num int64) []byte {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, uint64(num))

	return bs
}

func UintToHex(num uint64) []byte {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, num)

	return bs
}
