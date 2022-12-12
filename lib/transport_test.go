package lib

import (
	"reflect"
	"testing"
)

func TestSizeTo12Bytes(t *testing.T) {
	type args struct {
		sz int
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
		{"", args{1}, []byte{48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 49}},
		{"", args{100}, []byte{48, 48, 48, 48, 48, 48, 48, 48, 48, 49, 48, 48}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SizeTo12Bytes(tt.args.sz); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SizeTo12Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSizeFrom12Bytes(t *testing.T) {
	type args struct {
		sz []byte
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
		{"", args{[]byte{48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 49}}, 1},
		{"", args{[]byte{48, 48, 48, 48, 48, 48, 48, 48, 48, 49, 48, 49}}, 101},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSizeFrom12Bytes(tt.args.sz); got != tt.want {
				t.Errorf("GetSizeFrom12Bytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
