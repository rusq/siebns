package siebns

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type fakeWriteSeeker struct {
	b *bytes.Buffer

	WantSeekError  error
	WantWriteError error
}

func (f *fakeWriteSeeker) Write(p []byte) (n int, err error) {
	if f.b == nil {
		f.b = &bytes.Buffer{}
	}
	if f.WantWriteError != nil {
		return 0, f.WantWriteError
	}
	return f.b.Write(p)
}

func (f *fakeWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	if f.b == nil {
		f.b = &bytes.Buffer{}
	}
	if f.WantSeekError != nil {
		return 0, f.WantSeekError
	}
	return offset, nil
}

func (f *fakeWriteSeeker) Bytes() []byte {
	return f.b.Bytes()
}

func Test_hasBOM(t *testing.T) {
	type args struct {
		line []byte
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 int
	}{
		{"has bom",
			args{[]byte{0xef, 0xbb, 0xbf, 'S', 'i', 'e', 'b', 'e', 'l'}},
			true, 3},
		{"no bom",
			args{[]byte("Siebel no BOM")},
			false, 0},
		{"empty line",
			args{[]byte{}},
			false, 0},
		{"nil line",
			args{nil},
			false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := hasBOM(tt.args.line)
			if got != tt.want {
				t.Errorf("hasBOM() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("hasBOM() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
func Test_hasDOSlineEnding(t *testing.T) {
	type args struct {
		line []byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"dos", args{[]byte("C:\\>command.com\r\n")}, true},
		{"unix", args{[]byte("/bin/bash\n")}, false},
		{"no line ending", args{[]byte("/bin/bash")}, false},
		{"empty", args{nil}, false},
		{"empty", args{[]byte{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasDOSlineEndings(tt.args.line); got != tt.want {
				t.Errorf("hasDOSlineEndings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_nsHeader_readEncodedSize(t *testing.T) {
	type fields struct {
		byteOrder binary.ByteOrder
		format    format
		offsets   offsets
		version   version
	}
	type args struct {
		r io.ReadSeeker
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{"valid data",
			fields{
				byteOrder: binary.LittleEndian,
				offsets:   offsets{checksum: 3},
			},
			args{bytes.NewReader([]byte{00, 00, 00, 66, 119, 89, 70, 66, 65, 77, 67, 65, 81, 65, 61})},
			0x0706050403020100, false,
		},
		{"invalid base64 data",
			fields{
				byteOrder: binary.LittleEndian,
				offsets:   offsets{checksum: 3},
			},
			args{bytes.NewReader([]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00})},
			0, true,
		},
		{"seek error",
			fields{
				byteOrder: binary.LittleEndian,
				offsets:   offsets{checksum: 3},
			},
			args{bytes.NewReader([]byte{00})},
			0, true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := &nsHeader{
				byteOrder: tt.fields.byteOrder,
				format:    tt.fields.format,
				offsets:   tt.fields.offsets,
				version:   tt.fields.version,
			}
			got, err := hdr.readEncodedSize(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("nsHeader.readEncodedSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("nsHeader.readEncodedSize() = %x, want %x", got, tt.want)
			}
		})
	}
}

func Test_nsHeader_writeEncodedSize(t *testing.T) {
	type fields struct {
		byteOrder binary.ByteOrder
		format    format
		offsets   offsets
		version   version
	}
	type args struct {
		w    io.WriteSeeker
		size int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantW   []byte
		wantErr bool
	}{
		{"ok",
			fields{
				byteOrder: binary.LittleEndian,
				offsets:   offsets{checksum: 2},
			},
			args{&fakeWriteSeeker{}, 500},
			12,
			[]byte{0x39, 0x41, 0x45, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x41, 0x3d},
			false},
		{"seek error",
			fields{
				byteOrder: binary.LittleEndian,
				offsets:   offsets{checksum: 2},
			},
			args{&fakeWriteSeeker{WantSeekError: errors.New("seek err")}, 500},
			0,
			nil,
			true},
		{"write error",
			fields{
				byteOrder: binary.LittleEndian,
				offsets:   offsets{checksum: 2},
			},
			args{&fakeWriteSeeker{WantWriteError: errors.New("write err")}, 500},
			0,
			nil,
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := &nsHeader{
				byteOrder: tt.fields.byteOrder,
				format:    tt.fields.format,
				offsets:   tt.fields.offsets,
				version:   tt.fields.version,
			}
			got, err := hdr.writeEncodedSize(tt.args.w, tt.args.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("nsHeader.writeEncodedSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("nsHeader.writeEncodedSize() = %v, want %v", got, tt.want)
			}
			f := tt.args.w.(*fakeWriteSeeker)
			if diff := cmp.Diff(tt.wantW, f.Bytes()); diff != "" {
				t.Errorf("writeEncodedSize() output mismatch, (-want,+got):\n%s", diff)
			}
		})
	}
}

func Test_readHeader(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *nsHeader
		wantErr bool
	}{
		{"ok",
			args{bytes.NewReader([]byte(testfileEmpty))},
			&nsHeader{
				byteOrder: binary.LittleEndian,
				offsets:   offsets{checksum: 57},
				format:    format{unicode: false, dos: false},
				version: version{
					siebel: "16.0.0.0 [23057] ENU",
					nsfile: "1.2",
				},
			},
			false},
		{"corrupt file",
			args{bytes.NewReader([]byte(testfileCorrupt))},
			nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readHeader(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var opts cmp.Option
			if tt.want != nil {
				opts = cmp.AllowUnexported(*tt.want, (*tt.want).format, (*tt.want).offsets, (*tt.want).version)
			}
			if diff := cmp.Diff(tt.want, got, opts); diff != "" {
				t.Errorf("readHeader() mismatch, (-want,+got):\n%s", diff)
			}
		})
	}
}
