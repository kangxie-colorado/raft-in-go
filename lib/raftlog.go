package lib

type RaftLogEntry struct {
	Term int
	Item interface{}
}

type RaftLog struct {
	items []RaftLogEntry
}

// a helper to create RaftLogEntry
func CreateRaftLogEntry(term int, item interface{}) RaftLogEntry {
	return RaftLogEntry{term, item}
}

// a helper function to test the AppendEntries, test the entries after appending
func (raftlog *RaftLog) _afterAppendEntries(index, prevTerm int, entries []RaftLogEntry) []RaftLogEntry {

	raftlog.AppendEntries(index, prevTerm, entries)
	return raftlog.items
}

func (raftlog *RaftLog) AppendEntries(index, prevTerm int, entries []RaftLogEntry) bool {
	if index < 0 {
		return false
	}

	if index > len(raftlog.items) {
		// leaving holes is not allowed
		return false
	} else {
		if index > 0 && prevTerm != raftlog.items[index-1].Term {
			// log continuity must be maintained
			// log continuity is a recursive concept
			// prevTerm and the one prior forms a chain
			return false
		}

		if len(entries) != 0 {
			// this was to deal with special case, but actually this is also generally true
			// no effect to append an empty list, so lets skip it and this way index==0 is dealt too
			// and this is indeed necessary, say, having 5, testing append at index 3 with empty entries
			// the result should be true: yes, allow such things depending on other parameters but we cannot just
			// replace elem 4 and 5 with empty...!!!
			// need a test case for this
			raftlog.items = append(raftlog.items[:index], entries...)
		}
		return true

	}

	return false
}
