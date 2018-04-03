package slave

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"net"
	"os"
	"strconv"
)

func (s *Slave) connect() {

	udpAddr, err := net.ResolveUDPAddr("udp4", s.broadcastIP.String()+":"+strconv.Itoa(int(constants.MasterBroadcastPort)))
	checkError(err)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	checkError(err)

	udpAddr, err = net.ResolveUDPAddr("udp4", s.myIP.String()+":"+strconv.Itoa(int(constants.SlaveBroadcastPort)))
	checkError(err)
	conn2, err := net.ListenUDP("udp", udpAddr)
	checkError(err)

	var network bytes.Buffer
	network.WriteByte(byte(constants.ConnectionRequest))
	enc := gob.NewEncoder(&network)
	err = enc.Encode(packets.BroadcastConnectRequest{s.myIP, constants.SlaveBroadcastPort})
	checkError(err)
	_, err = conn.Write(network.Bytes())

	var buf [512]byte
	n, _, err := conn2.ReadFromUDP(buf[0:])

	fmt.Println(string(buf[0:n]))

}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}
}
