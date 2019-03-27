package siebns

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
)

// decodeSize determines the endianness and returns the valid file size
func decodeSize(b []byte) (int64, binary.ByteOrder, error) {
	base64Size, err := coding.DecodeString(strings.Trim(string(b[0:]), " "+string(crlf)))
	if err != nil {
		return 0, nil, err
	}

	size := sizeFromBinary(base64Size, binary.LittleEndian)
	if size > int64(math.MaxInt32<<1) {
		// assuming that size of the siebNS file can't be more than 4TB...
		return sizeFromBinary(base64Size, binary.BigEndian), binary.BigEndian, nil
	}
	return size, binary.LittleEndian, nil
}

func encodeSize(size int64, byteOder binary.ByteOrder) ([]byte, error) {
	if size == 0 {
		return nil, errZeroFilesize
	}

	sizeBytes := sizeToBinary(size, byteOder)
	out := make([]byte, coding.EncodedLen(len(sizeBytes)))
	coding.Encode(out, sizeBytes)

	return out, nil
}

// sizeFromBinary translates the byte sequence into the int64.  It is a
// wrapper around the binary.Read()
func sizeFromBinary(b []byte, byteOrder binary.ByteOrder) (size int64) {
	if byteOrder == nil {
		panic(errByteOrderNil)
	}
	if err := binary.Read(bytes.NewReader(b), byteOrder, &size); err != nil {
		panic(err)
	}
	return
}

// sizeFromBinary translates the byte sequence into the int64.  It is a
// wrapper around the binary.Read()
func sizeToBinary(size int64, byteOrder binary.ByteOrder) []byte {
	if byteOrder == nil {
		panic(errByteOrderNil)
	}
	var buf bytes.Buffer
	_ = binary.Write(&buf, byteOrder, size)
	return buf.Bytes()
}
