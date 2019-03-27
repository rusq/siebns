package siebns

import (
	"bytes"
	"io"
)

// detectUnicode checks the provided line for BOM signature, and returns
// the length of the signature (offset of the text)
func hasBOM(line []byte) (bool, int) {
	if line == nil || len(line) < len(bom) {
		return false, 0
	}
	// unicode detection
	if bytes.Equal(bom[:], line[:len(bom)]) {
		return true, len(bom)
	}
	return false, 0
}

// hasDOSlineEndings returns true if the line ends with \r\n, otherwise returns false
func hasDOSlineEndings(line []byte) bool {
	if line == nil || len(line) < len(crlf) {
		return false
	}
	return bytes.Equal(line[len(line)-2:], crlf)
}

// readHeader parses the header in the reader returning the nsHeader
// structure
func readHeader(r io.Reader) (*nsHeader, error) {
	rd := newLineReader(r)

	sigLine := rd.readline()
	if rd.err() != nil {
		return nil, rd.err()
	}
	isUnicode, textOffset := hasBOM(sigLine)
	fileSignature := sigLine[textOffset : textOffset+len(signature)]
	if string(fileSignature) != signature {
		return nil, errNotSiebns
	}

	hdr := nsHeader{}

	// format
	hdr.format.dos = hasDOSlineEndings(sigLine)
	hdr.format.unicode = isUnicode

	// versions
	hdr.version.siebel = rd.readstring()
	hdr.version.nsfile = rd.readstring()

	// checksum business
	hdr.offsets.checksum = rd.position()
	checksumLine := rd.readline()
	if rd.err() != nil {
		return nil, rd.err()
	}
	if err := hdr.verifyCksumFmt(checksumLine); err != nil {
		return nil, err
	}
	_, byteOrder, err := decodeSize(checksumLine)
	if err != nil {
		return nil, err
	}
	hdr.byteOrder = byteOrder

	return &hdr, nil
}

// readEncodedSize reads the encoded string from header and attempts to
// decode it as size
func (hdr *nsHeader) readEncodedSize(r io.ReadSeeker) (int64, error) {
	_, err := r.Seek(hdr.offsets.checksum, io.SeekStart)
	if err != nil {
		return 0, err
	}
	data := make([]byte, checksumSz)
	if n, err := r.Read(data); err != nil || n == 0 {
		return 0, err
	}
	size, byteOrder, err := decodeSize(data)
	if err != nil {
		return 0, err
	}
	hdr.byteOrder = byteOrder
	return size, nil
}

// writeEncodedSize encodes and writes the size to provided io.WriterAt,
func (hdr *nsHeader) writeEncodedSize(w io.WriteSeeker, size int64) (int, error) {
	data, err := encodeSize(size, hdr.byteOrder)
	if err != nil {
		return 0, err
	}
	_, err = w.Seek(hdr.offsets.checksum, io.SeekStart)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// verifyCksumFmt verifies that the provided checksum []byte is valid for
// this header
func (hdr *nsHeader) verifyCksumFmt(checksum []byte) error {
	// expected line length depending on whether the line is in dos format
	var isDOScksumLen = map[bool]int{
		false: 26,
		true:  27,
	}
	if len(checksum) != isDOScksumLen[hdr.format.dos] {
		return errChecksumCorrupt
	}
	return nil
}
