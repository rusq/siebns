package siebns

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

type testVals struct {
	size           int64
	base64data     string
	isLittleEndian bool
}

var testSet = []testVals{
	{size: 780, base64data: "DAMAAAAAAAA=", isLittleEndian: true},
	{size: 780, base64data: "AAAAAAAAAww=", isLittleEndian: false},
}

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

func CreateTestNSFile(contents string) *NSFile {
	f, err := ioutil.TempFile("", "siebns")
	if err != nil {
		panic("Unable to create test file")
	}
	f.Write([]byte(contents))
	ns := NSFile{
		Name: f.Name(),
		Size: int64(len(contents)),
		f:    f}
	return &ns
}

func CloseTestNSFile(ns *NSFile) {
	name := ns.f.Name()
	ns.f.Close()
	if err := os.Remove(name); err != nil {
		panic(err)
	}
}

func TestDecode(t *testing.T) {

	for i, v := range testSet {
		size, isLittleEndian, err := decodeSize([]byte(v.base64data))
		if err != nil {
			t.Errorf("Item %d: Got error: %v", i, err)
		}
		if size != v.size {
			t.Errorf("Item %d: Size invalid: got: %d, want: %d.",
				i, size, v.size)
		}
		if isLittleEndian != v.isLittleEndian {
			t.Errorf("Item %d: isLittleEndian invalid: got: %v, want: %v.",
				i, isLittleEndian, v.isLittleEndian)
		}
	}

}

func TestEncode(t *testing.T) {

	ns := NSFile{}
	for i, v := range testSet {
		ns.Size = v.size
		if v.isLittleEndian {
			ns.formatFlags |= FmtIsLittleEndian
		} else {
			ns.formatFlags = ns.formatFlags &^ FmtIsLittleEndian
		}
		encoded, err := ns.encodeSize()
		if err != nil {
			t.Errorf("Item %d: Got error: %v", i, err)
		}
		if string(encoded) != v.base64data {
			t.Errorf("Item %d:  Invalid data:  got: %s, want: %s.", i,
				encoded, v.base64data)
		}
	}
}

func TestNSFileError(t *testing.T) {
	if testErr := NSFileError("Big mistake"); strings.Index("Big mistake", testErr.Error()) > 0 {
		t.Fail()
	}
}

func TestParseHeader(t *testing.T) {
	ns := CreateTestNSFile(testfileEmpty)
	defer CloseTestNSFile(ns)

	if err := ns.parseHeader(); err != nil {
		t.Fatal(err)
	}
	if !ns.CorrectionNeeded {
		t.Error("CorrectionNeeded invalid")
	}
	if ns.formatFlags&FmtIsDOS == 1 {
		t.Error("fmtDos invalid")
	}
	if ns.formatFlags&FmtIsUnicode == 1 {
		t.Error("fmtUnicode invalid")
	}
	if ns.offsetHeader != 0 {
		t.Errorf("offsetHeader invalid: want: %d, got: %d.", 0, ns.offsetHeader)
	}
	if ns.offsetChecksum != 57 {
		t.Errorf("offsetChecksum invalid: want: %d, got: %d.", 57, ns.offsetChecksum)
	}
}

func TestParseHeaderCorrupt(t *testing.T) {
	ns := CreateTestNSFile(testfileCorrupt)
	defer CloseTestNSFile(ns)

	if err := ns.parseHeader(); err != nil {
		if strings.Index("Checksum part is corrupt", err.Error()) == 0 {
			t.Error("Did not detect checksum line corruption.")
		}
	}
}

func TestFixSize(t *testing.T) {
	ns := CreateTestNSFile(testfileEmpty)
	defer CloseTestNSFile(ns)

	if err := ns.parseHeader(); err != nil {
		t.Fatal(err)
	}
	if !ns.CorrectionNeeded {
		t.Error("CorrectionNeeded invalid")
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
	if err := ns.Load(ns.Name); err != nil {
		t.Fatal(err)
	}
	defer ns.Close()
	if ns.Size != int64(len(testfileEmpty)) {
		t.Errorf("Wrong Size.  Want: %d, Got: %d.", len(testfileEmpty), ns.Size)
	}

}
