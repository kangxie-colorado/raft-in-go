package lib

import (
	"math"
	"net"
	"os"

	transport "github.com/kangxie-colorado/golang-primer/messaging/lib"
	log "github.com/sirupsen/logrus"
)

func Echo(sock net.Conn) {
	for {
		msg, err := transport.RecvMessage(sock)
		if err != nil || msg == nil {
			return
		}

		transport.SendMessage(sock, msg)
	}

}

func EchoServer(sock transport.SocketDescriptor) {
	ln, err := net.Listen(sock.ConnType, sock.ConnHost+":"+sock.ConnPort)
	if err != nil {
		log.Errorf("Cannot listen on %+v\n", sock)
		os.Exit(1)
	}

	defer ln.Close()

	for {
		cl, err := ln.Accept()
		if err != nil {
			log.Errorln("Error connecting:", err.Error())
			return
		}
		log.Infoln("Client Connected: ", cl.RemoteAddr().String())

		go Echo(cl)
	}

}

func EchoClient(sock transport.SocketDescriptor) {
	log.Infoln("Connecting to " + sock.ConnType + sock.ConnHost + ":" + sock.ConnPort)

	conn, err := net.Dial(sock.ConnType, sock.ConnHost+":"+sock.ConnPort)
	if err != nil {
		log.Errorln("Error Conecting:", err.Error())
		os.Exit(1)
	}

	for n := 0; n < 8; n++ {
		log.Infoln("Testing", math.Pow10(n))
		msgBytes := make([]byte, int(math.Pow10(n)))

		for i := 0; i < len(msgBytes); i++ {
			msgBytes[i] = 'a'
		}

		transport.SendMessage(conn, msgBytes)
		respBytes, _ := transport.RecvMessage(conn)

		if len(msgBytes) != len(respBytes) {
			log.Errorln("Len is not right")
			return
		}

		for i := 0; i < len(respBytes); i++ {
			if respBytes[i] != msgBytes[i] {
				log.Errorln("Different at byte ", i, respBytes[i], "vs", msgBytes[i])
				return
			}
		}
	}

}
