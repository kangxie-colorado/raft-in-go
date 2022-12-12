package lib

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type nodeRole uint

const raftTimeUnit time.Duration = 10 * time.Millisecond

type RaftServer struct {
	raftnet   *RaftNet
	raftstate *RaftState

	myID        int
	followerIDs []int

	electionTimer   int
	electionTimeOut int

	// gurad around commitIdx read/write by
	// this way, I can use raftserver.lock() like https://kaviraj.me/understanding-condition-variable-in-go/
	// entering wait() will release the lock iirc
	lock sync.Mutex
	cond *sync.Cond

	// callback to application
	raftcallback func([]RaftLogEntry)
}

func CreateARaftServer(id int, callback func([]RaftLogEntry)) *RaftServer {
	var raftserver = RaftServer{}
	raftserver.myID = id

	raftserver.raftnet = CreateARaftNet(id)

	raftserver.lock = sync.Mutex{}
	raftserver.cond = sync.NewCond(&raftserver.lock)

	var followerIDs = []int{}
	for k := range RaftNetConfig {
		if k != id {
			followerIDs = append(followerIDs, k)
		}

	}
	raftserver.followerIDs = followerIDs

	// used for playing logs for client e.g. kv server
	raftserver.raftcallback = callback

	// election timer, random value between 80-100 time units
	// should be much bigger than heartbeat, which currently used as 20 time units to not go so fraze
	raftserver.electionTimeOut = rand.Intn(20) + 100
	raftserver.electionTimer = 0

	var raftstate = RaftState{}
	raftstate.initRaftState(raftserver.myID)
	raftserver.raftstate = &raftstate

	return &raftserver
}

func (raftserver *RaftServer) Net() *RaftNet {
	return raftserver.raftnet
}

// set term?
// leader election happened, new leader will update its own term
// follower will update on messages
// that is much later
func (raftserver *RaftServer) currentLeaderTerm() int {
	return raftserver.raftstate.currentTerm
}

// prototype of watiForCommit()
func (raftserver *RaftServer) watiForCommit(writtenIdx int, commited chan bool) {
	raftserver.lock.Lock()
	log.Debugf("lock: %v is locked, waiting on cond var: %v", raftserver.lock, raftserver.cond)

	//log.Debugf("commitIdx now %v, and wirttenIdx: %v", raftserver.commitIdx, writtenIdx)
	log.Infoln("Waiting Raft to commit")
	for raftserver.raftstate.commitIdx < writtenIdx {
		raftserver.cond.Wait()
	}
	log.Infof("Raft Committed: commitIdx now %v, and wirttenIdx: %v", raftserver.raftstate.commitIdx, writtenIdx)
	commited <- true

	raftserver.lock.Unlock()
	log.Debugf("lock: %v is unlocked", raftserver.lock)

}

func (raftserver *RaftServer) AppendNewEntry(msg string, commited chan bool) {
	for raftserver.raftstate.myRole == Candidate {
		// waiting until it becomes a leader of follower
		// sleep 20 timeUnits is sensible enough, which is the heartbeat frequency as well
		log.Infoln("I'm a candidate! I don't know who is leader yet!")
		time.Sleep(120 * raftTimeUnit)
	}

	if raftserver.raftstate.myRole == Follower {
		log.Errorln("I am a follower, and received a client write request, ignore")
		commited <- false
	}

	// append to leader
	// from the respective of leader, the prevTermFromLog before appending, is the prevTerm parameter to the function
	// currentLeaderTerm() >= prevTermFromLog()
	writeIdx := len(raftserver.raftstate.raftlog.items)
	success := raftserver.raftstate.LeaderAppendEntries(writeIdx, raftserver.raftstate.prevTermFromLog(), []RaftLogEntry{RaftLogEntry{raftserver.currentLeaderTerm(), msg}}, raftserver)

	if success {
		// for now, not blocked waiting for commited
		// need to think how to wait for commit ... better in a go routine actuall
		go raftserver.watiForCommit(writeIdx, commited)

		// commited <- true
	}
}

func (raftserver *RaftServer) LeaderNoop() {
	// first thing in my term, write a "noop" to celebrate the election
	// this should be in the winning election moment, between the state transition
	// but at this moment, I just want to make sure I write a noop
	// and not to block the main event loop, this will be done in go-routine
	if raftserver.raftstate.myRole == Leader {

		commitWait := make(chan bool)
		raftserver.AppendNewEntry("NOP", commitWait)
		<-commitWait
	}
}

func (raftserver *RaftServer) Start() {
	go raftserver.stateMachineRun()
	go raftserver.raftnet.Start()
	go raftserver.heartbeatGenerator()
	go raftserver.electionTimerRun()
}

func (raftserver *RaftServer) electionTimerRun() {
	for {
		time.Sleep(raftTimeUnit)
		raftserver.electionTimer += 1
		if raftserver.electionTimer == raftserver.electionTimeOut {
			raftserver.raftnet.inbox.Enqueue("ELECTIMEOUT")
		}
	}
}

func (raftserver *RaftServer) heartbeatGenerator() {
	for {
		time.Sleep(10 * raftTimeUnit) // num * 10ms
		if raftserver.raftstate.myRole == Leader {
			// when heart beat is going on, no more election timeout for myself
			raftserver.electionTimer = 0

			hb := CreateAppendEntriesMsg(raftserver.myID, raftserver.currentLeaderTerm(), raftserver.raftstate.commitIdx,
				len(raftserver.raftstate.raftlog.items), raftserver.raftstate.prevTermFromLog(), []RaftLogEntry{})
			for _, f := range raftserver.followerIDs {
				log.Debugf("Sending heartbeat to %v", f)
				raftserver.raftnet.Send(f, hb.Encoding())
			}

		}
	}
}

func (raftserver *RaftServer) stateMachineRun() {
	for {
		log.Debugln("Waiting for a message")
		msgB64Encoding, _ := raftserver.raftnet.Receive()

		// route based on the message type
		// can refactor to a function if there is better solution
		switch msgB64Encoding[:MSGTYPEFIELDLEN] {
		case APPENDENTRYMSG:
			msg := AppendEntriesMsg{}
			msg.Decoding(msgB64Encoding[MSGTYPEFIELDLEN:])
			log.Debugf("AppendEntriesMsg received: %v\n", msg.Repr())

			raftserver.raftstate.handleAppendEntriesMsg(&msg, raftserver)
		case APPENDENTRYRSP:
			msg := AppendEntriesResp{}
			msg.Decoding(msgB64Encoding[MSGTYPEFIELDLEN:])
			log.Debugf("AppendEntriesResp received: %v\n", msg.Repr())

			raftserver.raftstate.handleAppendEntriesResp(&msg, raftserver)

		case REQUESTVOTEMSG:
			msg := RequestVoteMsg{}
			msg.Decoding(msgB64Encoding[MSGTYPEFIELDLEN:])
			log.Debugf("RequestVoteMsg received: %v\n", msg.Repr())

			raftserver.raftstate.handleRequestVoteMsg(&msg, raftserver)

		case REQUESTVOTERESP:
			msg := RequestVoteResp{}
			msg.Decoding(msgB64Encoding[MSGTYPEFIELDLEN:])
			log.Debugf("RequestVoteResp received: %v\n", msg.Repr())

			raftserver.raftstate.handleRequestVoteResp(&msg, raftserver)

		case ELECTIMEOUT:
			log.Debugf("Election Time Out Hppened for me, I am server", raftserver.myID)
			raftserver.raftstate.handleElectionTimout(raftserver)

		default:
			log.Errorln("Unknown Message Type! %s", msgB64Encoding)
		}

	}
}

func DetermineCommitIdx(replicatedIdxes [NetWorkSize]int) int {
	// [5,4,4,3,3] => 4
	// [5,4,3,3,3] => 3
	// looks like it is just to get the middle number?
	cpy := make([]int, NetWorkSize)
	copy(cpy, replicatedIdxes[:])
	sort.Ints(cpy)

	log.Debugf("the replication indexes are %v\n", cpy)

	return cpy[NetWorkSize/2]
}

func (raftserver *RaftServer) ProcessCommitUpdate(msgCommitIdx int) {
	// in case the replication has not caught up on the follower, just wait until next nudge
	// it could be a heartbeat or next message
	if len(raftserver.raftstate.raftlog.items) > msgCommitIdx && raftserver.raftstate.commitIdx < msgCommitIdx {
		// msg.commitIdx is already written in leader and should be written in follower, so should plus 1 to form the right bound
		// raftserver.commitIdx, 1)starting with -1, 2)also should be written already: to avoid rewriting it, plus 1 generalize both
		go raftserver.raftcallback(raftserver.raftstate.raftlog.items[raftserver.raftstate.commitIdx+1 : msgCommitIdx+1])
		raftserver.raftstate.commitIdx = msgCommitIdx
	}

}

func (raftserver *RaftServer) SendRequestVoteResp(sendto int, msg *RequestVoteResp) {
	raftserver.raftnet.Send(sendto, msg.Encoding())
}

func (raftserver *RaftServer) LeaderId() int {
	return raftserver.raftstate.whoIsLeader
}

func (raftserver *RaftServer) IsLeader() bool {
	return raftserver.raftstate.myRole == Leader
}
