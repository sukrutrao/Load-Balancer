package master

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"net"
	"os"
	"strconv"
	"time"
)

func (m *Master) connect() {
	service := constants.BroadcastReceiveAddress.String() + ":" + strconv.Itoa(int(constants.MasterBroadcastPort))
	udpAddr, err := net.ResolveUDPAddr("udp4", service)
	checkError(err)
	conn, err := net.ListenUDP("udp", udpAddr)
	checkError(err)
	for {
		handleClient(conn)
	}
}

func handleClient(conn *net.UDPConn) {
	var buf [1024]byte
	n, addr, err := conn.ReadFromUDP(buf[0:])

	packetType := buf[0]

	switch int8(packetType) {
	case constants.ConnectionRequest:
		fmt.Println("ConnectionRequest")
		network := bytes.NewBuffer(buf[1:n])
		dec := gob.NewDecoder(network)

		var p packets.BroadcastConnectRequest
		err = dec.Decode(&p)
		checkError(err)

		fmt.Println("IP", p.Source.String(), "PORT", p.Port)

		addr, err = net.ResolveUDPAddr("udp4", p.Source.String()+":"+strconv.Itoa(int(p.Port)))

		if err != nil {
			return
		}
		daytime := time.Now().String()
		conn.WriteToUDP([]byte(daytime), addr)
	default:
		fmt.Println("Invalid Packet")
	}

}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}
}
