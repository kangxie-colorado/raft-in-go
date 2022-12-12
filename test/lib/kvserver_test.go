package lib

import "testing"

func Test_parseKeyValue(t *testing.T) {
	type args struct {
		msg string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		// TODO: Add test cases.
		{"", args{"SET00003FOO00003BAR"}, "FOO", "BAR"},
		{"", args{"SET00005AFOOD00005ABARK"}, "AFOOD", "ABARK"},
		{"foo:bar2", args{"SET00003foo00004bar2"}, "foo", "bar2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := parseKeyValue(tt.args.msg)
			if got != tt.want {
				t.Errorf("parseKeyValue() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseKeyValue() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
