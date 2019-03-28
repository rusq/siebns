// Copyrigh 2018 Rustam Gilyazov. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package siebns currently only allows fixing the encoded file size
// in Siebel Gateway Naming file after making manual modifications to it.
package siebns

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// other constants
const (
	headerSize = 82
	checksumSz = 12
	signature  = "Siebel Name Server Backing File"
)

type format struct {
	unicode bool
	dos     bool
}
type offsets struct {
	checksum int64
}
type version struct {
	siebel string
	nsfile string
}

//nsHeader contains the information from NS file header
type nsHeader struct {
	byteOrder binary.ByteOrder

	format  format
	offsets offsets
	version version
}

type nsDisker interface {
	io.Reader
	io.Writer
	io.Seeker
	io.Closer

	Name() string
	Stat() (os.FileInfo, error)
}

// NSFile Name Server File descriptor
type NSFile struct {
	header *nsHeader // ns information

	nsDisker // file handle
}

var (
	// bom is Unicode BOM
	bom = [3]byte{0xef, 0xbb, 0xbf}
	// crlf is dos carriage return and line feed
	crlf = []byte{0x0d, 0x0a}
	// coding is the encoding used
	coding = base64.StdEncoding
)

var (
	errNotInitialised  = errors.New("internal error: structure not initialised")
	errNotSiebns       = errors.New("not a siebel gateway file")
	errByteOrderNil    = errors.New("internal error: byteOrder is nil")
	errZeroFilesize    = errors.New("zero file size")
	errChecksumCorrupt = errors.New("Checksum part is corrupt.  Please fix it " +
		"manually by\nopening the file in the editor and deleting " +
		"data from line 4 (leaving the\nline 4 empty).")
)

// FixSize fixes the size in header regardless of whether
// NSFile.CorrectionNeeded is true or false.  Sets NSFile.CorrectionNeeded to
// false
func (ns *NSFile) FixSize() (int, error) {
	return ns.header.writeEncodedSize(ns, ns.Size())
}

// IsHeaderCorrect returns true if the file header doesn't need adjustment
func (ns *NSFile) IsHeaderCorrect() bool {
	size, _ := ns.header.readEncodedSize(ns)
	return (ns.Size() == size)
}

// Open opens existing nsfile
func Open(path string) (*NSFile, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.Size() < headerSize {
		return nil, errNotSiebns
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	hdr, err := readHeader(f)

	ns := &NSFile{
		nsDisker: f,
		header:   hdr,
	}

	return ns, err
}

// Size returns the file size
func (ns *NSFile) Size() int64 {
	fi, err := ns.Stat()
	if err != nil {
		panic(err)
	}
	return fi.Size()
}
