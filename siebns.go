// Copyrigh 2018 Rustam Gilyazov. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package siebns currently only allows fixing the encoded file size
// in Siebel Gateway Naming file after making manual modifications to it.
package siebns

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// NSFileError used to return errors
type NSFileError string

// NSFixer interface for fixing Files
type NSFixer interface {
	FixSize()
	Load(path string) error
	Close()
}

// NSFile Name Server File descriptor
type NSFile struct {
	Name             string   // filename and path
	Size             int64    // real file size
	CorrectionNeeded bool     // true if checksum is invalid
	fmtUnicode       bool     // true if file has BOM
	fmtDos           bool     // true if file has dos line endings
	fmtLittleEndian  bool     // true if file is little-endian (x86)
	offsetHeader     byte     // header offset
	offsetChecksum   int64    // checksum value offset
	offsetData       int      // actual data offset
	f                *os.File // file handle
}

// Error interface to output errors.
func (e NSFileError) Error() string {
	return fmt.Sprintf("Error: %s", string(e))
}

// FixSize fixes the size in header regardless of whether
// NSFile.CorrectionNeeded is true or false.  Sets NSFile.CorrectionNeeded to
// false
func (ns *NSFile) FixSize() (int, error) {
	buffer := make([]byte, 12)
	ns.f.ReadAt(buffer, ns.offsetChecksum)
	encoded, err := ns.encodeSize()
	if err != nil {
		return 0, err
	}
	ns.f.Seek(ns.offsetChecksum, os.SEEK_SET)
	wrote, err := ns.f.Write(encoded)
	if err != nil {
		return 0, err
	}
	ns.CorrectionNeeded = false
	return wrote, nil
}

// Close the file
func (ns *NSFile) Close() {
	ns.f.Close()
}

// Load loads an ns file and verifies that it's valid
func (ns *NSFile) Load(path string) error {
	var err error
	fileInfo, serr := os.Stat(path)
	if serr != nil {
		return (serr)
	}
	if fileInfo.Size() <= 0x3e {
		return NSFileError("Not a Siebel Gateway file.")
	}
	ns.Size = fileInfo.Size()

	ns.f, err = os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	ns.Name = path

	err = ns.parseHeader()
	if err != nil {
		return err
	}

	return nil
}

// decodeSize determines the endianness and returns the valid file size
func decodeSize(b []byte) (int64, bool, error) {
	var size int64
	base64Size, err := base64.StdEncoding.DecodeString(
		strings.Trim(string(b[0:]), " \r\n"))
	if err != nil {
		return 0, false, err
	}
	buf := bytes.NewReader(base64Size)
	err = binary.Read(buf, binary.LittleEndian, &size)
	if err != nil {
		return 0, false, err
	}
	littleEndian := !(size > int64(4294967296))
	if !littleEndian {
		buf.Seek(0, os.SEEK_SET)
		err = binary.Read(buf, binary.BigEndian, &size)
		if err != nil {
			return 0, false, err
		}
	}
	return size, littleEndian, err
}

func (ns *NSFile) parseHeader() error {
	ns.offsetHeader = 0
	ns.fmtDos = false
	ns.fmtUnicode = false
	ns.CorrectionNeeded = false
	currentPos := 0

	// read format line
	ns.f.Seek(0, os.SEEK_SET)
	reader := bufio.NewReader(ns.f)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return err
	}
	currentPos += len(line)

	// unicode detection
	if line[0] == 0xef && line[1] == 0xbb && line[2] == 0xbf {
		ns.fmtUnicode = true
		ns.offsetHeader = 3
	}
	// signature
	if string(line[ns.offsetHeader:6+ns.offsetHeader]) != "Siebel" {
		return NSFileError("Not a Siebel Gateway file.")
	}
	// line endings
	if line[len(line)-2] == '\r' {
		ns.fmtDos = true
	}

	// skip 2 lines and read the base64 value
	for i := 0; i < 2; i++ {
		line, err = reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		currentPos += len(line)
	}

	ns.offsetChecksum = int64(currentPos)
	line, err = reader.ReadBytes('\n')
	if (ns.fmtDos && len(line) != 27) || (!ns.fmtDos && len(line) != 26) {
		return NSFileError("Checksum part is corrupt.  Please fix it manually by\nopening the file in the editor and deleting data from line 4 (leaving the\nline 4 empty).")
	}

	var base64Size int64
	base64Size, ns.fmtLittleEndian, err = decodeSize(line)
	if base64Size != ns.Size || err != nil {
		ns.CorrectionNeeded = true
	}

	return nil
}

func (ns *NSFile) encodeSize() ([]byte, error) {
	var err error
	if ns.Size == 0 {
		panic("Zero file size")
	}
	buf := new(bytes.Buffer)

	if ns.fmtLittleEndian {
		err = binary.Write(buf, binary.LittleEndian, ns.Size)
	} else {
		err = binary.Write(buf, binary.BigEndian, ns.Size)
	}
	if err != nil {
		return nil, err
	}

	output := base64.StdEncoding.EncodeToString(buf.Bytes())

	return []byte(output), nil
}
