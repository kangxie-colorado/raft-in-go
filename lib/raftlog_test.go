package lib

import (
	"reflect"
	"testing"
)

var emptyEntries = []RaftLogEntry{}

var fig7ToAppendEntry = RaftLogEntry{8, 811} // 811, term 8, pos 11, index 10
var fig7ToAppendIndex = 10
var fig7ToAppendPrevTerm = 6
var fig7ItemToBecome = []RaftLogEntry{RaftLogEntry{1, 11}, RaftLogEntry{1, 12}, RaftLogEntry{1, 13}, RaftLogEntry{4, 44}, RaftLogEntry{4, 45},
	RaftLogEntry{5, 56}, RaftLogEntry{5, 57}, RaftLogEntry{6, 68}, RaftLogEntry{6, 69}, RaftLogEntry{6, 610}, RaftLogEntry{8, 811}}

// some stupid scheme here
// 610: term 6, pos 10 index 9
// 69: term 6, pos 9 index 8
// lets keep term under 10 hahaha
var fig7Scenarios = [][]RaftLogEntry{
	[]RaftLogEntry{RaftLogEntry{1, 11}, RaftLogEntry{1, 12}, RaftLogEntry{1, 13}, RaftLogEntry{4, 44}, RaftLogEntry{4, 45},
		RaftLogEntry{5, 56}, RaftLogEntry{5, 57}, RaftLogEntry{6, 68}, RaftLogEntry{6, 69}},

	[]RaftLogEntry{RaftLogEntry{1, 11}, RaftLogEntry{1, 12}, RaftLogEntry{1, 13}, RaftLogEntry{4, 44}},
	[]RaftLogEntry{RaftLogEntry{1, 11}, RaftLogEntry{1, 12}, RaftLogEntry{1, 13}, RaftLogEntry{4, 44}, RaftLogEntry{4, 45},
		RaftLogEntry{5, 56}, RaftLogEntry{5, 57}, RaftLogEntry{6, 68}, RaftLogEntry{6, 69}, RaftLogEntry{6, 610}, RaftLogEntry{6, 611}},
	[]RaftLogEntry{RaftLogEntry{1, 11}, RaftLogEntry{1, 12}, RaftLogEntry{1, 13}, RaftLogEntry{4, 44}, RaftLogEntry{4, 45},
		RaftLogEntry{5, 56}, RaftLogEntry{5, 57}, RaftLogEntry{6, 68}, RaftLogEntry{6, 69}, RaftLogEntry{6, 610}, RaftLogEntry{7, 711}, RaftLogEntry{7, 712}},
	[]RaftLogEntry{RaftLogEntry{1, 11}, RaftLogEntry{1, 12}, RaftLogEntry{1, 13}, RaftLogEntry{4, 44}, RaftLogEntry{4, 45},
		RaftLogEntry{5, 46}, RaftLogEntry{5, 47}},
	[]RaftLogEntry{RaftLogEntry{1, 11}, RaftLogEntry{1, 12}, RaftLogEntry{1, 13}, RaftLogEntry{2, 24}, RaftLogEntry{2, 25},
		RaftLogEntry{2, 26}, RaftLogEntry{3, 47}, RaftLogEntry{3, 38}, RaftLogEntry{3, 39}, RaftLogEntry{3, 310}, RaftLogEntry{3, 311}},
}

func DeepEqual(x interface{}, y interface{}) bool {
	// handle special emtpy list to empty list comparison otherwise this non-sense
	//     raftlog_test.go:69: RaftLog.AfterAppendEntries() = [], want []

	//	fmt.Println(reflect.TypeOf(x)) -> this reveals the type to be []lib.RaftLogEntry

	switch x.(type) {
	case []RaftLogEntry:
		xAsList := x.([]RaftLogEntry)
		yAsList := y.([]RaftLogEntry)
		if len(xAsList) == 0 && len(yAsList) == 0 {
			return true
		} else {
			return reflect.DeepEqual(x, y)
		}
	default:
		return reflect.DeepEqual(x, y)
	}
}

func TestRaftLog_AppendEntries(t *testing.T) {
	type fields struct {
		items []RaftLogEntry
	}
	type args struct {
		index    int
		prevTerm int
		entries  []RaftLogEntry
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
		{"append at index 0", fields{}, args{0, 0, []RaftLogEntry{RaftLogEntry{0, 1}}}, true},
		{"append at index 1 - same term, should pass", fields{[]RaftLogEntry{RaftLogEntry{0, 1}}}, args{1, 0, []RaftLogEntry{RaftLogEntry{0, 2}, RaftLogEntry{0, 3}}}, true},

		{"append at index 0 - empty", fields{}, args{0, 0, emptyEntries}, true},
		{"append at index 0 - empty entries to append", fields{[]RaftLogEntry{RaftLogEntry{0, 1}}}, args{0, 0, emptyEntries}, true},

		{"append at index 5 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{5, 0, []RaftLogEntry{RaftLogEntry{0, 6}}}, true},
		{"empty test - append ok? at index 5 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{5, 0, emptyEntries}, true},

		{"idempotent test - append at index 4 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{4, 0, []RaftLogEntry{RaftLogEntry{0, 5}}}, true},

		{"replace test - append at index 4 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{4, 0, []RaftLogEntry{RaftLogEntry{0, 6}}}, true},

		{"hole test - append at index 9 - while only having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{9, 0, []RaftLogEntry{RaftLogEntry{0, 9}}}, false},

		{"non-matching pre-term - append at index 5 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{5, 1, []RaftLogEntry{RaftLogEntry{0, 6}}}, false},

		{"empty test - append ok? at index 3 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{3, 0, emptyEntries}, true},

		{"fig7-a", fields{fig7Scenarios[0]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, false},
		{"fig7-b", fields{fig7Scenarios[1]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, false},
		{"fig7-c", fields{fig7Scenarios[2]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, true},
		{"fig7-d", fields{fig7Scenarios[3]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, true},
		{"fig7-e", fields{fig7Scenarios[4]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, false},
		{"fig7-f", fields{fig7Scenarios[5]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raftlog := &RaftLog{
				items: tt.fields.items,
			}
			if got := raftlog.AppendEntries(tt.args.index, tt.args.prevTerm, tt.args.entries); got != tt.want {
				t.Errorf("RaftLog.AppendEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRaftLog__afterAppendEntries(t *testing.T) {
	type fields struct {
		items []RaftLogEntry
	}
	type args struct {
		index    int
		prevTerm int
		entries  []RaftLogEntry
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []RaftLogEntry
	}{
		// TODO: Add test cases.
		{"append at index 0 - 1 item", fields{}, args{0, 0, []RaftLogEntry{RaftLogEntry{0, 1}}}, []RaftLogEntry{RaftLogEntry{0, 1}}},
		{"append at index 0 - 2 items", fields{}, args{0, 0, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}}}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}}},
		{"append at index 0 - replace", fields{[]RaftLogEntry{RaftLogEntry{0, 1}}}, args{0, 0, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}}}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}}},
		{"append at index 0 - empty", fields{}, args{0, 0, emptyEntries}, emptyEntries},
		{"append at index 0 - empty entries to append", fields{[]RaftLogEntry{RaftLogEntry{0, 1}}}, args{0, 0, emptyEntries}, []RaftLogEntry{RaftLogEntry{0, 1}}},

		{"append at index 5 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{5, 0, []RaftLogEntry{RaftLogEntry{0, 6}}}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}, RaftLogEntry{0, 6}}},

		{"empty test - append ok? at index 5 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{5, 0, []RaftLogEntry{}}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},

		{"idempotent test - append at index 4 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{4, 0, []RaftLogEntry{RaftLogEntry{0, 5}}}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},

		{"replace test - append at index 4 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{4, 0, []RaftLogEntry{RaftLogEntry{0, 6}}}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 6}}},

		{"non-matching pre-term - append at index 5 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{5, 1, []RaftLogEntry{RaftLogEntry{0, 6}}}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},

		{"hole test - append at index 9 - while only having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{9, 0, []RaftLogEntry{RaftLogEntry{0, 9}}}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},

		{"empty test - append ok? at index 3 - while having 5", fields{[]RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},
			args{3, 0, emptyEntries}, []RaftLogEntry{RaftLogEntry{0, 1}, RaftLogEntry{0, 2}, RaftLogEntry{0, 3}, RaftLogEntry{0, 4}, RaftLogEntry{0, 5}}},

		{"fig7-a", fields{fig7Scenarios[0]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, fig7Scenarios[0]},
		{"fig7-b", fields{fig7Scenarios[1]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, fig7Scenarios[1]},
		{"fig7-c", fields{fig7Scenarios[2]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, fig7ItemToBecome},
		{"fig7-d", fields{fig7Scenarios[3]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, fig7ItemToBecome},
		{"fig7-e", fields{fig7Scenarios[4]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, fig7Scenarios[4]},
		{"fig7-f", fields{fig7Scenarios[5]}, args{fig7ToAppendIndex, fig7ToAppendPrevTerm, []RaftLogEntry{fig7ToAppendEntry}}, fig7Scenarios[5]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raftlog := &RaftLog{
				items: tt.fields.items,
			}
			if got := raftlog._afterAppendEntries(tt.args.index, tt.args.prevTerm, tt.args.entries); !DeepEqual(got, tt.want) {
				t.Errorf("RaftLog._afterAppendEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}
