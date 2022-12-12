package lib

import (
	"fmt"
	"math/rand"

	log "github.com/sirupsen/logrus"
)

const (
	Undefined nodeRole = iota
	Follower
	Candidate
	Leader
)

func (noderole nodeRole) asString() string {
	switch noderole {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Undefined"
	}
}

type RaftState struct {
	raftlog *RaftLog // share the reference with raftserver

	myId        int
	whoIsLeader int // only need to store in follower side

	myRole      nodeRole
	currentTerm int // maybe not the same as the term from log
	votedFor    int
	votes       [NetWorkSize]bool

	// book keeping
	commitIdx     int
	replicatedIdx [NetWorkSize]int

	nextIdx [NetWorkSize]int // not in use yet
}

func (raftstate *RaftState) Repr() string {
	return fmt.Sprintf("RaftState{myId=%v, myRole=%v, currentTerm=%v, votedFor=%v}",
		raftstate.myId, raftstate.myRole.asString(), raftstate.currentTerm, raftstate.votedFor)
}

func (raftstate *RaftState) initRaftState(id int) {
	raftstate.myId = id
	raftstate.whoIsLeader = -1

	raftstate.myRole = Follower

	raftstate.currentTerm = 0
	if raftstate.myId == 3 {
		// rogue server test
		// test if it joins a established cluster and takes over
		raftstate.currentTerm = 100000
	}
	raftstate.votedFor = -1
	raftstate.votes = [NetWorkSize]bool{false}

	raftstate.raftlog = &RaftLog{}

	raftstate.commitIdx = -1
	raftstate.replicatedIdx = [NetWorkSize]int{
		-1, -1, -1, -1, -1,
	}

	raftlogLen := 0
	raftstate.nextIdx = [NetWorkSize]int{
		raftlogLen, raftlogLen, raftlogLen, raftlogLen, raftlogLen,
	}

}

func (raftstate *RaftState) LeaderAppendEntries(index, prevTerm int, entries []RaftLogEntry, raftserver *RaftServer) bool {
	if raftstate.myRole != Leader {
		return false
	}

	success := raftstate.raftlog.AppendEntries(index, prevTerm, entries)

	if success {
		// book keeping leader itself
		raftstate.replicatedIdx[raftserver.myID] = index

		// send AppendEntries Msg to followers
		for _, f := range raftserver.followerIDs {
			m := CreateAppendEntriesMsg(raftstate.myId, raftstate.currentTerm, raftstate.commitIdx, index, prevTerm, entries)
			// simple fixed length type field for now, of course can wrap this with even one more layer.
			// not the major focus
			log.Debugf("Sending raftnode:%v, msg: %v", f, m.Repr())
			raftserver.raftnet.Send(f, m.Encoding())
		}

	}
	return success
}

func (raftstate *RaftState) handleAppendEntriesMsg(msg *AppendEntriesMsg, raftserver *RaftServer) {
	// if msg.term > my term, convert to follower
	// if I am candidation, unconditionally conver to follower
	log.Debugf("Current I am %s, received %s", raftstate.Repr(), msg.Repr())

	if raftstate.myRole == Leader && msg.LeaderTerm == raftstate.currentTerm {
		panic("Edge Case Really Happened! Two leaders of same term really happened!")
		// may need to compare who logs are more up-to-date if this really happenes
	}

	// If AppendEntries RPC received from new leader: convert to follower
	// this is unconitionally; this is separate of returning false or true
	if raftstate.myRole == Candidate {
		log.Infoln("Change from a candidate to a follower unconditionally!")
		raftstate.myRole = Follower
		raftstate.currentTerm = msg.LeaderTerm
	}
	// it should always reset timer when receiving an AppendEntriesMsg
	// because there is a leader in play
	// If election timeout elapses without receiving AppendEntries RPC from current leader or granting vote to candidate: convert to candidate
	// above means, it there is AppendEntriesRPC incoming, it should never become a candidate
	raftserver.electionTimer = 0

	if msg.LeaderTerm >= raftstate.currentTerm {
		log.Debugln("Becoming/Staying A Follower")
		// 		raftstate.myRole = Follower <--- this is implicitly true here
		raftstate.currentTerm = msg.LeaderTerm
		raftstate.whoIsLeader = msg.SenderId
	}

	success := raftstate.raftlog.AppendEntries(msg.Index, msg.PrevTerm, msg.Entries)
	if success {
		// corner case: when a rogue candidate/follower join the network
		// it needs to be equalized to current leader term if the appendEntry was good
		raftstate.currentTerm = msg.LeaderTerm
	}

	resp := AppendEntriesResp{raftstate.myId, success, msg.Index, len(msg.Entries), raftstate.prevTermFromLog()}
	raftserver.raftnet.Send(msg.SenderId, resp.Encoding())

	// if the commitIdx is newer than myself, I might want to look into update my commit
	// especially when it is a heartbeat message
	if len(msg.Entries) == 0 {
		raftserver.ProcessCommitUpdate(msg.LeaderCommitIdx)
	}
}

func (raftstate *RaftState) handleAppendEntriesResp(msg *AppendEntriesResp, raftserver *RaftServer) {
	log.Debugf("Current I am %s, received %s", raftstate.Repr(), msg.Repr())

	// only learder is possible to receive this message
	// in network partiotion and minority side, the fake leader will continue to receieve from the together partioned follower
	// but in that sense, it is still leader and won't be able to commit at all
	// because we ignore the msg term < my term, so message with a bigger term won't come back
	// this simplifies things; this fake leader can become follower when it receives the new leader's heartbeat
	if raftstate.myRole != Leader {
		panic("Not A Leader! But Received AppendEntriesResp " + msg.Repr())
	}

	// deal as normal
	raftstate.ProcessAppendEntriesResp(msg, raftserver)

}

func (raftstate *RaftState) handleRequestVoteMsg(msg *RequestVoteMsg, raftserver *RaftServer) {
	log.Debugf("Current I am %s, received %s", raftstate.Repr(), msg.Repr())
	raftstate.whoIsLeader = -1 // voting, I don't know a leder? yeah, logicll okay

	// 1. Reply false if term < currentTerm (§5.1)
	voteGranted := false

	// • If RPC request or response contains term T > currentTerm: set currentTerm = T, convert to follower (§5.1)
	// this is unconditional and it does NOT suggest more than this
	if msg.Term > raftstate.currentTerm {
		log.Infof("Becoming A Follower Because RequestVoteMsg msgTerm is newer than my own: %s", msg.Repr())
		raftstate.currentTerm = msg.Term
		raftstate.myRole = Follower
	}

	// 2. If votedFor is null or candidateId, and candidate’s log is at least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
	// if msg.Term > raftstate.currentTerm, it will become equal by above block
	if msg.Term == raftstate.currentTerm && msg.LastLogTerm >= raftstate.prevTermFromLog() &&
		msg.LastLogIdx >= len(raftstate.raftlog.items)-1 &&
		(raftstate.votedFor == -1 || raftstate.votedFor == msg.SenderId) {
		raftstate.votedFor = msg.SenderId
		raftstate.votes[raftserver.myID] = false
		voteGranted = true

		raftserver.electionTimer = 0
	}

	// if not granted, should reset the votedFor
	// if myself is also a candidate it will become myself on next timeout
	if !voteGranted {
		raftstate.votedFor = -1
	}

	resp := RequestVoteResp{raftstate.myId, raftstate.currentTerm, voteGranted}
	raftserver.SendRequestVoteResp(msg.SenderId, &resp)

}

func (raftstate *RaftState) handleRequestVoteResp(msg *RequestVoteResp, raftserver *RaftServer) {
	log.Infof("Current I am %s, received %s", raftstate.Repr(), msg.Repr())

	if raftstate.myRole != Candidate {
		// this can happen in the transition
		// will leader receive it? in transtiion, yes, slow response
		log.Infof("I am not a candidate: %s, yet I received RequestVoteResp: %s, ignore", raftstate.Repr(), msg.Repr())
		return

	}

	if msg.Term < raftstate.currentTerm {
		log.Infof("Vote for my early term? I am %s, received %s", raftstate.Repr(), msg.Repr())
		return
	}

	// If RPC request or response contains term T > currentTerm: set currentTerm = T, convert to follower (§5.1)
	// then per 1. Reply false if term < currentTerm (§5.1)
	// if this happens, the reply must be false
	if msg.Term > raftstate.currentTerm {
		log.Infof("Becoming A Follower Because RequestVoteResp msgTerm is newer than my own: %v", msg.Repr())
		raftstate.myRole = Follower
		raftstate.currentTerm = msg.Term
	}

	raftstate.votes[msg.SenderId] = msg.VoteGranted
	log.Infof("Votes Received %v", raftstate.votes)

	howManyVoteForMe := 0
	for _, v := range raftstate.votes {
		if v {
			howManyVoteForMe += 1
		}
	}

	if howManyVoteForMe >= NetWorkSize/2+1 {

		log.Infoln("Becoming The Leader!")
		raftstate.myRole = Leader
		go raftserver.LeaderNoop()
	}

}

func (raftstate *RaftState) prevTermFromLog() int {
	if len(raftstate.raftlog.items) == 0 {
		// doesn't matter, append at index 0 always succeeds
		return -1
	}

	return raftstate.raftlog.items[len(raftstate.raftlog.items)-1].Term
}

func (raftstate *RaftState) FollowerAppendEntries(msg *AppendEntriesMsg, raftserver *RaftServer) bool {
	success := raftstate.raftlog.AppendEntries(msg.Index, msg.PrevTerm, msg.Entries)
	resp := AppendEntriesResp{raftstate.myId, success, msg.Index, len(msg.Entries), raftstate.prevTermFromLog()}

	raftserver.raftnet.Send(msg.SenderId, resp.Encoding())

	// if the commitIdx is newer than myself, I might want to look into update my commit
	// especially when it is a heartbeat message
	if len(msg.Entries) == 0 && success {
		raftserver.ProcessCommitUpdate(msg.LeaderCommitIdx)
	}

	return success
}

func (raftstate *RaftState) ProcessAppendEntriesResp(msg *AppendEntriesResp, raftserver *RaftServer) {
	if msg.Success {
		log.Debugf("Follower %v able to append to its own %v\n", msg.SenderId, msg.Repr())
		// establish the consensus here
		raftstate.replicatedIdx[msg.SenderId] = msg.Index + msg.NumOfEntries - 1

		// determine the highest replicatedIDX which has 3 or more shows, including leader itself
		// bookkeeping of leader itself happens in LeaderAppendEntries()
		newCommitIdx := DetermineCommitIdx(raftstate.replicatedIdx)
		raftserver.lock.Lock()
		log.Debugf("lock: %v is locked, raftserver.commitIdx is %v, newCommitIdx is %v", raftserver.lock, raftserver.raftstate.commitIdx, newCommitIdx)
		if raftstate.commitIdx < newCommitIdx && raftstate.raftlog.items[newCommitIdx].Term == raftserver.raftstate.currentTerm {
			raftstate.commitIdx = newCommitIdx
		}
		raftserver.lock.Unlock()
		log.Debugf("lock: %v is unlocked", raftserver.lock)

		// we should now signla checkForCommit channel
		// any client/goroutine waiting for its entry to be committed can now move on
		raftserver.cond.Broadcast()

	} else {
		log.Errorf("Follower %v not able to append to its own %v, back tracking\n", msg.SenderId, msg.Repr())
		prevTerm := 0
		newIndex := Max(msg.Index-1, raftstate.replicatedIdx[msg.SenderId])
		if newIndex > 0 {
			prevTerm = raftstate.raftlog.items[newIndex-1].Term
		}
		backoffMsg := AppendEntriesMsg{raftserver.myID, raftstate.currentTerm, raftstate.commitIdx, newIndex, prevTerm, raftstate.raftlog.items[newIndex:]}
		// two places sending to follower, would there be some refactoring? maybe no need
		// conditional send to follower: condition being if index<nextIdx[follower], no need to send more than one time... but it is just an optimization
		// do that later
		raftserver.raftnet.Send(msg.SenderId, backoffMsg.Encoding())
	}

}

func (raftstate *RaftState) handleElectionTimout(raftserver *RaftServer) {
	log.Debugf("Current I am %s", raftstate.Repr())

	if raftstate.myRole == Leader {
		panic("As a Leader, I received election timeout!!!")
	}

	if raftstate.myRole == Follower {
		log.Infoln("Becoming A Candidate!")
		// become a candidate then the candidate block will be executed on originally follower as well
		raftstate.myRole = Candidate
	}

	if raftstate.myRole == Candidate {
		log.Infoln("Now I am A Candidate!")

		raftstate.currentTerm += 1
		for i := range raftstate.votes {
			// start new election, all previous votes should reset
			// but who I voted for last time can stay,
			// but here I vote for myself again
			raftstate.votes[i] = false
		}
		raftstate.votedFor = raftstate.myId
		raftstate.votes[raftstate.myId] = true

		raftserver.electionTimer = 0
		raftserver.electionTimeOut = rand.Intn(20) + 100

		for _, f := range raftserver.followerIDs {
			rfv := RequestVoteMsg{raftstate.myId, raftstate.currentTerm, len(raftstate.raftlog.items) - 1, raftstate.prevTermFromLog()}
			log.Debugf("Sending RequestVote %s", rfv.Repr())

			raftserver.raftnet.Send(f, rfv.Encoding())
		}
	}

}
