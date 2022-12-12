package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kangxie-colorado/golang-primer/messaging/lib"
	lib_test "github.com/kangxie-colorado/golang-primer/messaging/test/lib"
	trafficlight "github.com/kangxie-colorado/golang-primer/messaging/test/traffic_light"
	log "github.com/sirupsen/logrus"
)

func initLog(filename string, logLevel log.Level) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(file)
	log.SetLevel(logLevel)
	//log.SetReportCaller(true)
}

// quick test a simple server , clien->conn
func testSimpleServer() {
	prog := os.Args[1]

	if prog == "server" {
		lib_test.SimpleServer(lib.SocketDescriptor{"tcp", "localhost", "15000"})
	} else if prog == "client" {
		sz, _ := strconv.Atoi(os.Args[2])
		lib_test.SimpleClient(lib.SocketDescriptor{"tcp", "localhost", "15000"}, sz)
	} else {
		fmt.Println("Wrong program type!")
	}
}

// echo server test
func testEchoServer() {
	prog := os.Args[1]

	if prog == "server" {
		initLog("server.log", log.DebugLevel)
		lib_test.EchoServer(lib.SocketDescriptor{"tcp", "localhost", "15000"})
	} else if prog == "client" {
		initLog("client.log", log.InfoLevel)
		lib_test.EchoClient(lib.SocketDescriptor{"tcp", "localhost", "15000"})
	} else {
		fmt.Println("Wrong program type!")
	}
}

// kv server test - KVServer is already hooked up with raft, this won't work anymore
func testKVServer_NotWorking() {
	prog := os.Args[1]

	if prog == "server" {
		initLog("server.log", log.DebugLevel)
		lib_test.KVServer(lib.SocketDescriptor{"tcp", "localhost", "15000"}, 0)
	} else if prog == "client" {
		initLog("client.log", log.InfoLevel)
		kvclient := lib_test.CreateKVClient(lib.SocketDescriptor{"tcp", "localhost", "15000"})
		kvclient.Get("foo")

		kvclient.Set("foo", "bar")
		kvclient.Get("foo")

	} else {
		fmt.Println("Wrong program type!")
	}
}

func testRaftNet() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.DebugLevel)

	var raftnets = make([]lib.RaftNet, 5)

	for i := 0; i < 5; i++ {
		raftnets[i] = *lib.CreateARaftNet(i)
		go raftnets[i].Start()
	}

	// disable 0->2 but and 2->0
	//
	raftnets[0].Disable(2)
	raftnets[2].Disable(0)
	time.Sleep(3 * time.Second)

	for i := 0; i < 5; i++ {
		raftnets[i].BeginSending()
	}

	for i := 0; i < 5; i++ {
		for to := 0; to < 5; to++ {
			raftnets[i].Send(to, fmt.Sprintf("msg %v to %v", i, to))
			time.Sleep(3 * time.Millisecond)
		}
	}

	for i := 0; i < 5; i++ {
		for time := 0; time < 5; time++ {
			fmt.Println(raftnets[i].Receive())
		}
	}
}

// test with disable/enable network links
func testRaftNetWithDisableEnable() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.DebugLevel)

	var raftnets = make([]lib.RaftNet, 5)

	for i := 0; i < 5; i++ {
		raftnets[i] = *lib.CreateARaftNet(i)
		go raftnets[i].Start()
	}

	// disable 0->2 but and 2->0
	//
	raftnets[0].Disable(2)
	raftnets[2].Disable(0)
	time.Sleep(3 * time.Second)

	for i := 0; i < 5; i++ {
		raftnets[i].BeginSending()
	}

	for i := 0; i < 5; i++ {
		for to := 0; to < 5; to++ {
			raftnets[i].Send(to, fmt.Sprintf("msg %v to %v", i, to))
			time.Sleep(3 * time.Millisecond)
		}
	}

	raftnets[0].Enable(2)
	raftnets[2].Enable(0)

	time.Sleep(3 * time.Second)

	for i := 0; i < 5; i++ {
		for time := 0; time < 5; time++ {
			fmt.Println(raftnets[i].Receive())
		}
	}
}

/***
// gobtest - library code deleted - just keep the tests here

****************************************************************************************
// [tw-mbp-kxie test (master)]$ go run *go server
// Qv+BAwEBEEFwcGVuZEVudHJpZXNNc2cB/4IAAQMBBUluZGV4AQQAAQhQcmV2VGVybQEEAAEHRW50cmllcwH/hgAAACH/hQIBARJbXWxpYi5SYWZ0TG9nRW50cnkB/4YAAf+EAAAs/4MDAQEMUmFmdExvZ0VudHJ5Af+EAAECAQRUZXJtAQQAAQRJdGVtARAAAAA4/4IDAgIGc3RyaW5nDA0AC1NFVCBGT08gQkFSAAECAQZzdHJpbmcMDgAMU0VUIEZPTyBCQVIyAAA=
// {0 0 [{0 SET FOO BAR}]}
// {0 0 [{0 SET FOO BAR} {1 SET FOO BAR2}]}
// Qv+BAwEBEEFwcGVuZEVudHJpZXNNc2cB/4IAAQMBBUluZGV4AQQAAQhQcmV2VGVybQEEAAEHRW50cmllcwH/hgAAACH/hQIBARJbXWxpYi5SYWZ0TG9nRW50cnkB/4YAAf+EAAAs/4MDAQEMUmFmdExvZ0VudHJ5Af+EAAECAQRUZXJtAQQAAQRJdGVtARAAAABV/4IBAgIDAQQBBnN0cmluZwwOAAxTRVQgRk9PIEJBUjMAAQYBBnN0cmluZwwQAA5TRVQgRk9PMiBCQVIyMwABCAEGc3RyaW5nDAkAB0RFTCBGT08AAA==
// {1 0 [{2 SET FOO BAR3} {3 SET FOO2 BAR23} {4 DEL FOO}]}
****************************************************************************************

func testGOb() {
	gob.Register(lib.AppendEntriesMsg{})

	m := lib.CreateAppendEntriesMsg(0, 0, []lib.RaftLogEntry{lib.CreateRaftLogEntry(0, "SET FOO BAR"), lib.CreateRaftLogEntry(1, "SET FOO BAR2")})
	fmt.Println(lib.ToGOB64(&m))

	// upto SET FOO BAR
	m1 := lib.FromGOB64("Qv+BAwEBEEFwcGVuZEVudHJpZXNNc2cB/4IAAQMBBUluZGV4AQQAAQhQcmV2VGVybQEEAAEHRW50cmllcwH/hgAAACH/hQIBARJbXWxpYi5SYWZ0TG9nRW50cnkB/4YAAf+EAAAs/4MDAQEMUmFmdExvZ0VudHJ5Af+EAAECAQRUZXJtAQQAAQRJdGVtARAAAAAd/4IDAQIGc3RyaW5nDA0AC1NFVCBGT08gQkFSAAA=")
	fmt.Printf("%v\n", m1)

	// upto SET FOO BAR2
	m2 := lib.FromGOB64("Qv+BAwEBEEFwcGVuZEVudHJpZXNNc2cB/4IAAQMBBUluZGV4AQQAAQhQcmV2VGVybQEEAAEHRW50cmllcwH/hgAAACH/hQIBARJbXWxpYi5SYWZ0TG9nRW50cnkB/4YAAf+EAAAs/4MDAQEMUmFmdExvZ0VudHJ5Af+EAAECAQRUZXJtAQQAAQRJdGVtARAAAAA4/4IDAgIGc3RyaW5nDA0AC1NFVCBGT08gQkFSAAECAQZzdHJpbmcMDgAMU0VUIEZPTyBCQVIyAAA=")
	fmt.Printf("%v\n", m2)

	m3 := lib.CreateAppendEntriesMsg(1, 0, []lib.RaftLogEntry{lib.CreateRaftLogEntry(2, "SET FOO BAR3"), lib.CreateRaftLogEntry(3, "SET FOO2 BAR23"), lib.CreateRaftLogEntry(4, "DEL FOO")})
	fmt.Println(lib.ToGOB64(&m3))

	m4 := lib.FromGOB64("Qv+BAwEBEEFwcGVuZEVudHJpZXNNc2cB/4IAAQMBBUluZGV4AQQAAQhQcmV2VGVybQEEAAEHRW50cmllcwH/hgAAACH/hQIBARJbXWxpYi5SYWZ0TG9nRW50cnkB/4YAAf+EAAAs/4MDAQEMUmFmdExvZ0VudHJ5Af+EAAECAQRUZXJtAQQAAQRJdGVtARAAAABV/4IBAgIDAQQBBnN0cmluZwwOAAxTRVQgRk9PIEJBUjMAAQYBBnN0cmluZwwQAA5TRVQgRk9PMiBCQVIyMwABCAEGc3RyaW5nDAkAB0RFTCBGT08AAA==")
	fmt.Printf("%v\n", m4)

}
***/

func testMultilServerMultiClient() {

	prog := "client"

	// default values to open up a routine to debug
	// I don't know how to play with the configuration json yet
	// so leave the no argument version for debugging
	// which means go run *go server 0
	if len(os.Args) > 1 {
		prog = os.Args[1]
	}

	if prog == "server" {
		raftID := 1
		if len(os.Args) > 2 {
			raftID, _ = strconv.Atoi(os.Args[2])
		}
		initLog("server"+strconv.Itoa(raftID)+".log", log.InfoLevel)
		log.Infoln("********************************************************************************************")

		port := 25000 + raftID
		if raftID > 2 {
			// slower follower... to simulate the appendentris failure
			time.Sleep(3 * time.Second)
		}
		lib_test.KVServer(lib.SocketDescriptor{"tcp", "localhost", strconv.Itoa(port)}, raftID)

	} else if prog == "client" {
		clientID := 2
		if len(os.Args) > 2 {
			clientID, _ = strconv.Atoi(os.Args[2])
		}

		initLog("client"+strconv.Itoa(clientID)+".log", log.DebugLevel)
		log.Infoln("********************************************************************************************")

		if clientID == 1 {
			kvclient := lib_test.CreateKVClient(lib.SocketDescriptor{"tcp", "localhost", "25000"})

			kvclient.Get("foo")
			kvclient.Set("foo", "bar-client1")
			time.Sleep(3 * time.Second)
			kvclient.Get("foo")
		} else if clientID == 2 {
			kvclient := lib_test.CreateKVClient(lib.SocketDescriptor{"tcp", "localhost", "25000"})

			kvclient.Get("foo")
			kvclient.Set("foo", "bar-client2")
			kvclient.Get("foo2")
			kvclient.Set("foo2", "bar-client2")
			kvclient.Get("foo")
			kvclient.Get("foo2")

			time.Sleep(3 * time.Second)

			kvclient.Del("foo")

		} else {
			// a client just to read off a follower
			// go run *go client 25002 (the port 25002 is connecting to server 2)
			kvclient := lib_test.CreateKVClient(lib.SocketDescriptor{"tcp", "localhost", os.Args[2]})
			kvclient.Get("foo")
			kvclient.Get("foo2")

			kvclient.Get("Kang")

			kvclient.Set("Kang", "Great!")
			kvclient.Get("Kang")
		}

	} else {
		fmt.Println("Wrong program type!")
	}
}

func testGoTalkToPythonSocket() {
	lightConn := trafficlight.ConnectToLight(lib.SocketDescriptor{"udp", "localhost", "10000"})

	lightConn.Write([]byte{'G'})
	time.Sleep(500 * time.Millisecond)

	lightConn.Write([]byte{'R'})
	time.Sleep(500 * time.Millisecond)

	lightConn.Write([]byte{'Y'})
	time.Sleep(500 * time.Millisecond)

	lightConn.Write([]byte{'G'})
	time.Sleep(500 * time.Millisecond)

}

/**
// code refactored away - just to keep a footprint of this test here
func testPythonTalkToGOSocket() {
	trafficlight.StartButtonListener(lib.SocketDescriptor{"udp", "localhost", "20000"})
}
**/

func testTrafficLight() {
	trafficlight.TestTFLStateMachine()

	tfl := trafficlight.CreateTFLControl([2]int{30, 60},
		[2]lib.SocketDescriptor{lib.SocketDescriptor{"udp", "localhost", "10000"}, lib.SocketDescriptor{"udp", "localhost", "10001"}},
		lib.SocketDescriptor{"udp", "localhost", "20000"})

	tfl.Start()
}

func main() {
	testMultilServerMultiClient()
}
