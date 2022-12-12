package lib

import (
	"fmt"
	"net"
	"os"

	"github.com/kangxie-colorado/golang-primer/messaging/lib"
	log "github.com/sirupsen/logrus"
)

func CreateKVClient(sock lib.SocketDescriptor) *KVClient {
	log.Infoln("Staring KV client")

	var conn net.Conn
	if sock.ConnType == "tcp" {
		conn = connectTo(sock.ConnHost + ":" + sock.ConnPort)
	}

	return &KVClient{conn}
}

func connectTo(to string) net.Conn {
	log.Infoln("Connecting to " + to)

	conn, err := net.Dial("tcp", to)
	if err != nil {
		log.Errorln("Error Conecting:", err.Error())
		os.Exit(1)
	}

	log.Infoln("Connected: ", conn.RemoteAddr(), conn.LocalAddr())

	return conn
}

type KVClient struct {
	conn net.Conn
}

func (cl *KVClient) kvClientSendMessage(buf []byte) (string, error) {
	lib.SendMessage(cl.conn, []byte(buf))

	resp, err := lib.RecvMessageStr(cl.conn)
	if len(resp) > 10 && resp[:10] == "REDIRECTED" {
		leaderAddr := resp[10:]

		log.Infoln("Create a new KV client and connect to the leader")
		conn := connectTo(leaderAddr)

		cl.conn = conn
		lib.SendMessage(cl.conn, buf)
		resp, err = lib.RecvMessageStr(cl.conn)
	}

	return resp, err
}

func (cl *KVClient) Set(key, value string) {
	payload := "SET" + encodeKeyValue(key, value)

	resp, _ := cl.kvClientSendMessage([]byte(payload))
	fmt.Println(resp)
}

func (cl *KVClient) Get(key string) {
	payload := "GET" + encodeKey(key)

	resp, _ := cl.kvClientSendMessage([]byte(payload))
	fmt.Println(resp)
}

func (cl *KVClient) Del(key string) {
	payload := "DEL" + encodeKey(key)

	resp, _ := cl.kvClientSendMessage([]byte(payload))
	fmt.Println(resp)
}
