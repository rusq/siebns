package siebns

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	testfileEmpty = `Siebel Name Server Backing File
16.0.0.0 [23057] ENU
1.2
DAMAAAAAAAA=             

[/]
	Persistence=partial
	Type=empty
`
	testfileCorrupt = "Siebel Name Server Backing File\n16.0.0.0 [23057] ENU\n1.2\nDA=     \n\n[/]\n	Persistence=partial\n	Type=empty"
)

/*
	Fixtures
	--------
*/
type fakeDisker struct {
	WantRead []byte

	WantStatFileInfo os.FileInfo
	WantStatError    error

	buf  *bytes.Buffer
	name string
}

func (d *fakeDisker) Close() error {
	d.buf.Reset()
	d.buf = nil
	return nil
}

func (d *fakeDisker) Read(p []byte) (int, error) {
	if d.buf == nil {
		if d.WantRead == nil {
			panic("set fakeDisker.WantRead the the []byte you want to 'Read'")
		}
		d.buf = bytes.NewBuffer(d.WantRead)
	}
	return d.buf.Read(p)
}

func (d *fakeDisker) Write(p []byte) (int, error) {
	if d.buf == nil {
		d.buf = &bytes.Buffer{}
	}
	return d.buf.Write(p)
}

func (d *fakeDisker) Name() string {
	return d.name
}

func (d *fakeDisker) Stat() (os.FileInfo, error) {
	return d.WantStatFileInfo, d.WantStatError
}

func (d *fakeDisker) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

type fakeFileInfo struct {
	WantName    string
	WantSize    int64
	WantMode    os.FileMode
	WantModTime time.Time
	WantIsDir   bool
	WantSys     interface{}
}

func (i *fakeFileInfo) Name() string       { return i.WantName }
func (i *fakeFileInfo) Size() int64        { return i.WantSize }
func (i *fakeFileInfo) Mode() os.FileMode  { return i.WantMode }
func (i *fakeFileInfo) ModTime() time.Time { return i.WantModTime }
func (i *fakeFileInfo) IsDir() bool        { return i.WantIsDir }
func (i *fakeFileInfo) Sys() interface{}   { return i.WantSys }

/*
	Other things
	------------
*/

type testVals struct {
	size           int64
	base64data     string
	isLittleEndian bool
}

var testSet = []testVals{
	{size: 780, base64data: "DAMAAAAAAAA=", isLittleEndian: true},
	{size: 780, base64data: "AAAAAAAAAww=", isLittleEndian: false},
}

/*
	Utility functions
	-----------------
*/

func CreateTestNSFile(contents string) *NSFile {
	f, err := ioutil.TempFile("", "siebns")
	if err != nil {
		panic("Unable to create test file")
	}
	f.Write([]byte(contents))
	f.Seek(0, io.SeekStart)
	ns := NSFile{
		nsDisker: f}
	return &ns
}

func CloseTestNSFile(ns *NSFile) {
	name := ns.Name()
	ns.Close()
	if err := os.Remove(name); err != nil {
		panic(err)
	}
}

/*
	Tests
	-----
*/

func TestParseHeader(t *testing.T) {
	ns := CreateTestNSFile(testfileEmpty)
	defer CloseTestNSFile(ns)

	hdr, err := readHeader(ns)
	if err != nil {
		t.Fatal(err)
	}
	ns.header = hdr

	if ns.IsHeaderCorrect() {
		t.Error("CorrectionNeeded invalid")
	}
	if ns.header.format.dos {
		t.Error("ns.format.dos invalid")
	}
	if ns.header.format.unicode {
		t.Error("ns.format.unicode invalid")
	}
	if ns.header.offsets.checksum != 57 {
		t.Errorf("offsets.checksum invalid: want: %d, got: %d.", 57, ns.header.offsets.checksum)
	}
}

func TestParseHeaderCorrupt(t *testing.T) {
	ns := CreateTestNSFile(testfileCorrupt)
	defer CloseTestNSFile(ns)

	if _, err := readHeader(ns); err != nil {
		if strings.Index("Checksum part is corrupt", err.Error()) == 0 {
			t.Error("Did not detect checksum line corruption.")
		}
	}
}

func TestFixSize(t *testing.T) {
	ns := CreateTestNSFile(testfileEmpty)
	defer CloseTestNSFile(ns)

	hdr, err := readHeader(ns)
	if err != nil {
		t.Fatal(err)
	}
	ns.header = hdr

	if ns.IsHeaderCorrect() {
		t.Error("IsHeaderCorrect() invalid")
	}

	bytes, err := ns.FixSize()
	if err != nil {
		t.Fatal(err)
	}
	if bytes != 12 {
		t.Errorf("Wrong bytes count.  Want: %d, Got: %d.", 12, bytes)
	}

}

func TestLoadClose(t *testing.T) {
	ns := CreateTestNSFile(testfileEmpty)
	if nsOpened, err := Open(ns.Name()); err != nil || nsOpened == nil {
		t.Fatal(err)
	}
	defer ns.Close()
	if ns.Size() != int64(len(testfileEmpty)) {
		t.Errorf("Wrong Size.  Want: %d, Got: %d.", len(testfileEmpty), ns.Size())
	}

}

func TestNSFile_IsHeaderCorrect(t *testing.T) {
	type fields struct {
		header   *nsHeader
		nsDisker nsDisker
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"header and actual match",
			fields{
				header: &nsHeader{
					byteOrder: binary.LittleEndian,
				},
				nsDisker: &fakeDisker{
					WantRead:         []byte{0x39, 0x41, 0x45, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x3d},
					WantStatFileInfo: &fakeFileInfo{WantSize: 500},
				},
			},
			true},
		{"header and actual mismatch",
			fields{
				header: &nsHeader{
					byteOrder: binary.LittleEndian,
				},
				nsDisker: &fakeDisker{
					WantRead:         []byte{0x39, 0x41, 0x45, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x3d},
					WantStatFileInfo: &fakeFileInfo{WantSize: 100500},
				},
			},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns := &NSFile{
				header:   tt.fields.header,
				nsDisker: tt.fields.nsDisker,
			}
			if got := ns.IsHeaderCorrect(); got != tt.want {
				t.Errorf("NSFile.IsHeaderCorrect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNSFile_Size(t *testing.T) {
	type fields struct {
		header   *nsHeader
		nsDisker nsDisker
	}
	tests := []struct {
		name      string
		fields    fields
		want      int64
		wantPanic bool
	}{
		{"ok",
			fields{nsDisker: &fakeDisker{
				WantStatFileInfo: &fakeFileInfo{WantSize: 0xFADE},
			}},
			0xFADE, false,
		},
		{"stat err",
			fields{nsDisker: &fakeDisker{
				WantStatError: errors.New("stat error"),
			}},
			0, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var panicked bool
			defer func() {
				if r := recover(); r != nil {
					panicked = true
					if tt.wantPanic != panicked {
						t.Errorf("panicked = %v, wantPanic = %v", panicked, tt.wantPanic)
					}
				}
			}()
			ns := &NSFile{
				header:   tt.fields.header,
				nsDisker: tt.fields.nsDisker,
			}
			if got := ns.Size(); got != tt.want {
				t.Errorf("NSFile.Size() = %v, want %v", got, tt.want)
			}
		})
	}
}
