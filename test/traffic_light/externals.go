package trafficlight

import (
	"fmt"
	"net"
	"os"

	"github.com/kangxie-colorado/golang-primer/messaging/lib"
	log "github.com/sirupsen/logrus"
)

type LightControl struct {
	conns [2]*net.UDPConn
}

func CreateLightControl(sock [2]lib.SocketDescriptor) LightControl {
	conn1 := ConnectToLight(sock[0])
	conn2 := ConnectToLight(sock[1])

	return LightControl{[2]*net.UDPConn{conn1, conn2}}
}

func (lightcontrol *LightControl) changeColor(lightNo int, color TrafficLightColor, countDown string) {

	msg := ""
	switch color {
	case Green:
		msg = "G" + countDown
	case Yellow:
		msg = "Y" + countDown

	case Red:
		msg = "R" + countDown

	default:
		msg = "R"

	}

	lightcontrol.conns[lightNo].Write([]byte(msg))

}

func ConnectToLight(sock lib.SocketDescriptor) *net.UDPConn {
	udpAddr, err := net.ResolveUDPAddr(sock.ConnType, sock.ConnHost+":"+sock.ConnPort)
	if err != nil {
		log.Errorln("Error Resolving:", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialUDP(sock.ConnType, nil, udpAddr)
	if err != nil {
		log.Errorln("Error Conecting:", err.Error())
		os.Exit(1)
	}

	log.Infoln("Connected: ", conn.RemoteAddr(), conn.LocalAddr())
	return conn
}

func StartButtonListener_old(tflctl *TrafficLight) {
	udpAddr, err := net.ResolveUDPAddr(tflctl.MySock.ConnType, tflctl.MySock.ConnHost+":"+tflctl.MySock.ConnPort)
	if err != nil {
		log.Errorln("Error Resolving:", err.Error())
		os.Exit(1)
	}

	udpConn, err := net.ListenUDP(tflctl.MySock.ConnType, udpAddr)
	if err != nil {
		log.Errorln("Error Listenting:", err.Error())
		os.Exit(1)
	}

	for {
		buf := make([]byte, 1024)
		udpConn.Read(buf)

		fmt.Println("Button Pressed")
		tflctl.InputQueue.Enqueue("Button Pressed")

	}
}

func StartButtonListener(tflctl *TFLControl) {
	udpAddr, err := net.ResolveUDPAddr(tflctl.mySock.ConnType, tflctl.mySock.ConnHost+":"+tflctl.mySock.ConnPort)
	if err != nil {
		log.Errorln("Error Resolving:", err.Error())
		os.Exit(1)
	}

	udpConn, err := net.ListenUDP(tflctl.mySock.ConnType, udpAddr)
	if err != nil {
		log.Errorln("Error Listenting:", err.Error())
		os.Exit(1)
	}

	for {
		buf := make([]byte, 1024)
		udpConn.Read(buf)

		fmt.Println("Button Pressed")
		tflctl.inputQueue.Enqueue("Button Pressed")

	}

}
