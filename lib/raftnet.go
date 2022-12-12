package lib

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/enriquebris/goconcurrentqueue"
	log "github.com/sirupsen/logrus"
)

func (raftnet *RaftNet) HandleIncomingConn(cl net.Conn) {

	for {
		// just enqueue the message
		msg, err := RecvMessageStr(cl)
		if err != nil {
			log.Debugln("Error when receiving messages", err.Error())
			break
		}

		raftnet.inbox.Enqueue(msg)
	}

}

func (raftnet *RaftNet) connectTo(toID int) net.Conn {
	log.Debugln("RaftNet", raftnet.Id, "as a client, connecting to RaftNet", toID)

	sock := RaftNetConfig[toID]
	log.Debugln("Connecting to " + sock.ConnType + sock.ConnHost + ":" + sock.ConnPort)

	conn, err := net.Dial(sock.ConnType, sock.ConnHost+":"+sock.ConnPort)
	if err != nil {
		log.Debugln("Error Conecting:", err.Error())
		return nil
	}

	log.Debugln("Connected: ", conn.RemoteAddr(), conn.LocalAddr())
	// the socket will not close but will block?
	// if I maintain the links, yeah this could be necessary
	// but if I connect everytime, this is not necessary?
	// I think it is unrelevant ...
	conn.SetDeadline(time.Time{})
	return conn
}

func (raftnet *RaftNet) sendWithNewConnEachTime(toID int, msg string) {
	conn := raftnet.connectTo(toID)

	if conn != nil {
		SendMessageStr(conn, msg)
		// if connecting everytime, lets close the connection after each time
		conn.Close()
	}
}

func (raftnet *RaftNet) sendWithLongConn(toID int, msg string) {
	// with long connection, we cache the conn links
	if raftnet.outlinks[toID] == nil {
		raftnet.outlinks[toID] = raftnet.connectTo(toID)
	}

	if raftnet.outlinks[toID] != nil {
		err := SendMessageStr(raftnet.outlinks[toID], msg)
		retries := 5
		for err != nil && retries > 0 {
			// refresh the connection in case of remote closing by setting it to nil and
			// next time when a message is to be sent, it will refresh
			// no need to worry about the message, it can be lost
			// the application layer will re-transmit
			raftnet.outlinks[toID] = nil
			raftnet.outlinks[toID] = raftnet.connectTo(toID)
			if raftnet.outlinks[toID] != nil {
				err = SendMessageStr(raftnet.outlinks[toID], msg)
			}

			retries--
		}
		if retries == 0 {
			log.Errorf("The link to raftnet%v is dead? retired 5 times already\n", toID)
		}
	}
}

func (raftnet *RaftNet) BeginSending() {
	for i := 0; i < NetWorkSize; i++ {
		raftnet.activated[i] <- true
	}
}

func (raftnet *RaftNet) bgSender(toID int) {
	<-raftnet.activated[toID]

	for {

		for !raftnet.enabled[toID] {
			// wait until the enable flag is set
			// can use a channel for that
			<-raftnet.linkEnabled[toID]
		}

		time.Sleep(5 * time.Millisecond)
		msg, err := raftnet.outboxs[toID].DequeueOrWaitForNextElement()
		if err != nil {
			log.Errorln("Error dequeue outbox", toID, err.Error())
		} else {
			raftnet.sendWithNewConnEachTime(toID, fmt.Sprintf("%v", msg))
		}
	}
}

func CreateARaftNet(id int) *RaftNet {
	var raftnet = RaftNet{}
	raftnet.Id = id

	// create the inbox and outboxes
	raftnet.inbox = goconcurrentqueue.NewFIFO()
	raftnet.outboxs = [NetWorkSize]*goconcurrentqueue.FIFO{
		goconcurrentqueue.NewFIFO(),
		goconcurrentqueue.NewFIFO(),
		goconcurrentqueue.NewFIFO(),
		goconcurrentqueue.NewFIFO(),
		goconcurrentqueue.NewFIFO(),
	}

	raftnet.outlinks = [NetWorkSize]net.Conn{
		nil,
	}

	raftnet.enabled = [NetWorkSize]bool{
		true, true, true, true, true,
	}

	raftnet.linkEnabled = [NetWorkSize]chan bool{
		make(chan bool),
		make(chan bool),
		make(chan bool),
		make(chan bool),
		make(chan bool),
	}
	raftnet.activated = [NetWorkSize]chan bool{
		make(chan bool),
		make(chan bool),
		make(chan bool),
		make(chan bool),
		make(chan bool),
	}

	return &raftnet
}

func (raftnet *RaftNet) Start() {
	log.Infoln("Staring Raft Node:", raftnet.Id)

	time.Sleep(1 * time.Second)

	// start the bg sender
	for id := 0; id < len(raftnet.outboxs); id++ {
		/** allow to self?
		if id == raftnet.Id {
			continue
		}
		**/
		go raftnet.bgSender(id)
	}

	// start listening as the server
	sock := RaftNetConfig[raftnet.Id]
	log.Debugln("Raft starts listening on", sock.ConnHost+":"+sock.ConnPort)
	ln, err := net.Listen(sock.ConnType, sock.ConnHost+":"+sock.ConnPort)
	if err != nil {
		log.Errorf("Raft cannot listen on %+v\n", sock)
		os.Exit(1)
	}

	defer ln.Close()

	for {
		cl, err := ln.Accept()
		if err != nil {
			log.Errorln("Error accepting:", err.Error())
			return
		}
		log.Debugln("Client Connected: ", cl.RemoteAddr().String())

		go raftnet.HandleIncomingConn(cl)
	}

}

func (raftnet *RaftNet) Send(toID int, msg string) {
	/** allow to self?
	if toID == raftnet.Id {
		return
	}
	**/
	raftnet.outboxs[toID].Enqueue(msg)
}

func (raftnet *RaftNet) Receive() (string, error) {
	msg, err := raftnet.inbox.DequeueOrWaitForNextElement()
	if err != nil {
		log.Errorln("Error dequeue inbox", err.Error())
	}

	return fmt.Sprintf("%v", msg), err
}

// knobs to enable/disable the connection
// after each change, signal the bgSender it has changed
// we need four channels
func (raftnet *RaftNet) Enable(toID int) {
	raftnet.enabled[toID] = true
	raftnet.linkEnabled[toID] <- true
}

func (raftnet *RaftNet) Disable(toID int) {
	raftnet.enabled[toID] = false
}
