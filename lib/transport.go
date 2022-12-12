package lib

import (
	"fmt"
	"net"
	"strconv"

	log "github.com/sirupsen/logrus"
)

func Min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func Max(a, b int) int {
	if a < b {
		return b
	}

	return a
}

func SizeTo12Bytes(sz int) []byte {
	strSz := fmt.Sprintf("%012d", sz)
	byteSz := []byte(strSz)

	return byteSz
}

func GetSizeFrom12Bytes(sz []byte) int {
	strSz := string(sz)
	intSz, _ := strconv.Atoi(strSz)

	return intSz
}

func SendMessageStr(conn net.Conn, msg string) error {
	return SendMessage(conn, []byte(msg))
}

func SendMessage(conn net.Conn, msg []byte) error {
	szBytes := SizeTo12Bytes(len(msg))
	msg = append(szBytes, msg...)

	writeBytes, err := conn.Write(msg)
	if err != nil {
		log.Debugln("Error when writing", err.Error())
	}

	log.Debugln("Written bytes:", writeBytes)

	return err
}

func RecvMessageStr(conn net.Conn) (string, error) {
	msgBytes, err := RecvMessage(conn)
	return string(msgBytes), err
}

func RecvMessage(conn net.Conn) ([]byte, error) {
	// conn.read can read either upto EOF or when buffer is filled
	szBuf := make([]byte, 12)
	_, err := conn.Read(szBuf)
	if err != nil {
		log.Debugln("Error when reading:", err.Error())
		return []byte{}, err
	}

	sz := GetSizeFrom12Bytes(szBuf)
	if sz == 0 {
		return nil, nil
	}

	tmp := make([]byte, 1024)
	msgBuf := make([]byte, 0)
	readLength := 0

	for {
		if sz-readLength < 1024 {
			// read last batch, re-make tmp
			tmp = make([]byte, sz-readLength)
		}

		readBytes, err := conn.Read(tmp)
		if err != nil || readBytes == 0 {
			log.Debugln("Error when reading", err.Error())
			break
		}

		msgBuf = append(msgBuf, tmp[:readBytes]...)
		readLength += readBytes
		if readLength == sz {
			break
		}

	}

	return msgBuf, err
}
