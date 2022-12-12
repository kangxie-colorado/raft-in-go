package lib

import (
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// scheme for msg: SET|GET|DEL + LEN-of-KEY(5 bytes) + Key + [LEN-of-VAL(5 bytes) + Val]

func parseKeyValue(msg string) (string, string) {
	keyLenStr := msg[3:8]
	keyLen, _ := strconv.Atoi(keyLenStr)
	key := msg[8 : 8+keyLen]

	value := msg[8+keyLen:]

	log.Infoln("Key:", key, "Value:", value)

	return key, value
}

func parseKey(msg string) string {
	keyLenStr := msg[3:8]
	keyLen, _ := strconv.Atoi(keyLenStr)
	key := msg[8 : 8+keyLen]
	log.Infoln("Key:", key)

	return key
}
func encodeKeyValue(key, value string) string {
	return fmt.Sprintf("%05d", len(key)) + key + value
}

func encodeKey(key string) string {
	return fmt.Sprintf("%05d", len(key)) + key
}
