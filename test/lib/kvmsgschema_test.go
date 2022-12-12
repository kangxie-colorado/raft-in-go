package lib

import "testing"

func Test_encodeKeyValue(t *testing.T) {
	type args struct {
		key   string
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{"foo:bar", args{"foo", "bar"}, "00003foo00003bar"},
		{"foo2:bar2", args{"foo2", "bar2"}, "00004foo200004bar2"},
		{"foo:bar2", args{"foo", "bar2"}, "00003foo00004bar2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeKeyValue(tt.args.key, tt.args.value); got != tt.want {
				t.Errorf("encodeKeyValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
