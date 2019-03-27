package siebns

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func Test_decodeSize(t *testing.T) {
	little01020304 := []byte{0x04, 0x03, 0x02, 0x01, 0, 0, 0, 0}
	b64little01020304 := make([]byte, coding.EncodedLen(len(little01020304)))
	coding.Encode(b64little01020304, little01020304)

	big01020304 := []byte{0, 0, 0, 0, 0x01, 0x02, 0x03, 0x04}
	b64big01020304 := make([]byte, coding.EncodedLen(len(big01020304)))
	coding.Encode(b64big01020304, big01020304)

	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		args    args
		want    int64
		want1   binary.ByteOrder
		wantErr bool
	}{
		{"little endian", args{b64little01020304}, 0x0000000001020304, binary.LittleEndian, false},
		{"big endian", args{b64big01020304}, 0x0000000001020304, binary.BigEndian, false},
		{"invalid b64", args{big01020304}, 0, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := decodeSize(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("decodeSize() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("decodeSize() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_encodeSize(t *testing.T) {
	type args struct {
		size     int64
		byteOder binary.ByteOrder
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{"LittleEndian",
			args{0x01020304050607, binary.LittleEndian},
			[]byte{66, 119, 89, 70, 66, 65, 77, 67, 65, 81, 65, 61},
			false,
		},
		{"BigEndian",
			args{0x01020304050607, binary.BigEndian},
			[]byte{65, 65, 69, 67, 65, 119, 81, 70, 66, 103, 99, 61},
			false,
		},
		{"zero bytes", args{0, binary.BigEndian}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeSize(tt.args.size, tt.args.byteOder)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeSize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("encodeSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sizeFromBinary(t *testing.T) {
	type args struct {
		b          []byte
		endianness binary.ByteOrder
	}
	tests := []struct {
		name      string
		args      args
		wantSize  int64
		wantPanic bool
	}{
		{"little endian",
			args{[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
				binary.LittleEndian},
			0x0706050403020100,
			false},
		{"big endian",
			args{[]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
				binary.BigEndian},
			0x0001020304050607,
			false},
		{"byteorder nil", args{[]byte{}, nil}, 0, true},
		{"bytes nil", args{nil, binary.BigEndian}, 0, true},
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
			if gotSize := sizeFromBinary(tt.args.b, tt.args.endianness); gotSize != tt.wantSize {
				t.Errorf("sizeFromBinary() = %v, want %v", gotSize, tt.wantSize)
			}
		})

	}
}

func Test_sizeToBinary(t *testing.T) {
	type args struct {
		size      int64
		byteOrder binary.ByteOrder
	}
	tests := []struct {
		name      string
		args      args
		want      []byte
		wantPanic bool
	}{
		{"little endian", args{0x0102030405060708, binary.LittleEndian},
			[]byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01}, false},
		{"big endian", args{0x0102030405060708, binary.BigEndian},
			[]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, false},
		{"byteorder nil", args{0, nil}, []byte{}, true},
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
			if got := sizeToBinary(tt.args.size, tt.args.byteOrder); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sizeToBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}
