package lib

import (
	"fmt"
	"io"
	"net"
	"os"

	transport "github.com/kangxie-colorado/golang-primer/messaging/lib"
)

func Handle(conn net.Conn) {

	// read in segment
	// make a temporary bytes var to read from the connection
	tmp := make([]byte, 1024)
	// make 0 length data bytes (since we'll be appending)
	data := make([]byte, 0)
	// keep track of full length read
	length := 0

	// loop through the connection stream, appending tmp to data
	for {
		// read to the tmp var
		n, err := conn.Read(tmp)
		if err != nil {
			// log if not normal error
			if err != io.EOF {
				fmt.Printf("Read error - %s\n", err)
			}
			break
		}

		// append read data to full data
		data = append(data, tmp[:n]...)

		// update total read var
		length += n
	}

	// log bytes read
	fmt.Printf("READ  %d bytes\n", length)

	fmt.Println("len is", len(data))
	for i := 0; i < len(data); i++ {
		if data[i] != 'a' {
			fmt.Println("bytes not mathcing at", i)
			return
		}
	}

}

func SimpleServer(sock transport.SocketDescriptor) {
	ln, err := net.Listen(sock.ConnType, sock.ConnHost+":"+sock.ConnPort)
	if err != nil {
		fmt.Printf("Cannot listen on %+v\n", sock)
		os.Exit(1)
	}

	defer ln.Close()

	for {
		cl, err := ln.Accept()
		if err != nil {
			fmt.Println("Error connecting:", err.Error())
			return
		}
		fmt.Println("Client Connected: ", cl.RemoteAddr().String())

		go Handle(cl)
	}

}

func SimpleClient(sock transport.SocketDescriptor, n int) {
	fmt.Println("Connecting to " + sock.ConnType + sock.ConnHost + ":" + sock.ConnPort)

	conn, err := net.Dial(sock.ConnType, sock.ConnHost+":"+sock.ConnPort)
	if err != nil {
		fmt.Println("Error Conecting:", err.Error())
		os.Exit(1)
	}

	fmt.Println("Testing", n)
	msgBytes := make([]byte, n)

	for i := 0; i < len(msgBytes); i++ {
		msgBytes[i] = 'a'
	}

	conn.Write(msgBytes)

}
