package master

import (
	"bytes"
	"encoding/gob"
	"net"
	"strconv"
	"time"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/GoodDeeds/load-balancer/common/utility"
)

func (m *Master) connect() {
	service := constants.BroadcastReceiveAddress.String() + ":" + strconv.Itoa(int(constants.MasterBroadcastPort))

	udpAddr, err := net.ResolveUDPAddr("udp4", service)
	utility.CheckFatal(err, m.Logger)

	conn, err := net.ListenUDP("udp", udpAddr)
	utility.CheckFatal(err, m.Logger)

	for {
		// TODO: add timeout and check if master is closed.
		m.handleClient(conn)
	}
}

// TODO: add slave to slavePool.
func (m *Master) handleClient(conn *net.UDPConn) {
	var buf [1024]byte
	n, addr, err := conn.ReadFromUDP(buf[0:])

	packetType := buf[0]

	switch constants.PacketType(packetType) {
	case constants.ConnectionRequest:
		network := bytes.NewBuffer(buf[1:n])
		dec := gob.NewDecoder(network)

		var p packets.BroadcastConnectRequest
		err = dec.Decode(&p)
		if err != nil {
			m.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet", "packet",
				"BroadcastConnectRequest", "err", err.Error()))
		}

		portStr := strconv.Itoa(int(p.Port))
		m.Logger.Info(logger.FormatLogMessage("msg", "Connection request", "ip", p.Source.String(), "port", portStr))

		addr, err = net.ResolveUDPAddr("udp4", p.Source.String()+":"+portStr)

		if err != nil {
			return
		}
		daytime := time.Now().String()
		conn.WriteToUDP([]byte(daytime), addr)
		// TODO: wait for an ACK from slave
	default:
		m.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet for connection"))
	}

}
