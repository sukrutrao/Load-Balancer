package master

import (
	"net"
	"strconv"
	"time"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/GoodDeeds/load-balancer/common/utility"
)

func (m *Master) connect() {
	m.closeWait.Add(1)
	service := constants.BroadcastReceiveAddress.String() + ":" + strconv.Itoa(int(constants.MasterBroadcastPort))

	udpAddr, err := net.ResolveUDPAddr("udp4", service)
	utility.CheckFatal(err, m.Logger)

	conn, err := net.ListenUDP("udp", udpAddr)
	utility.CheckFatal(err, m.Logger)

	packetChan := make(chan connectionReqData)

	go m.collectIncomingRequests(conn, packetChan)

	end := false
	for !end {
		select {
		case <-m.close:
			// Master is closed.
			// TODO: end connection with slaves.
			m.Logger.Info(logger.FormatLogMessage("msg", "Stopping to accept slave connection request"))
			end = true
			break
		default:
			// TODO: add timeout and check if master is closed.
			m.handleClient(conn, packetChan)
		}
	}
	m.closeWait.Done()
}

type connectionReqData struct {
	n    int
	addr *net.UDPAddr
	buf  [1024]byte
}

func (m *Master) collectIncomingRequests(conn *net.UDPConn, packetChan chan<- connectionReqData) {
	for {
		var buf [1024]byte
		n, addr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			m.Logger.Error(logger.FormatLogMessage("msg", "Error in reading from UDP"))
			continue
		}

		var bufCopy [1024]byte
		copy(bufCopy[:], buf[:])
		packetChan <- connectionReqData{
			n:    n,
			addr: addr,
			buf:  bufCopy,
		}
	}
}

// TODO: add slave to slavePool.
func (m *Master) handleClient(conn *net.UDPConn, packetChan <-chan connectionReqData) {

	select {
	case packet, ok := <-packetChan:
		if !ok {
			break
		}

		packetType := packets.PacketType(packet.buf[0])

		switch packetType {
		case packets.ConnectionRequest:
			var p packets.BroadcastConnectRequest
			err := packets.DecodePacket(packet.buf[1:packet.n], &p)
			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet", "packet",
					"BroadcastConnectRequest", "err", err.Error()))
			}

			portStr := strconv.Itoa(int(p.Port))
			m.Logger.Info(logger.FormatLogMessage("msg", "Connection request", "ip", p.Source.String(), "port", portStr))

			addr, err := net.ResolveUDPAddr("udp4", p.Source.String()+":"+portStr)

			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to resolve slave address", "err", err.Error()))
				return
			}

			// TODO: send ACK/NAC
			daytime := time.Now().String()
			conn.WriteToUDP([]byte(daytime), addr)
			// TODO: wait for an ACK from slave

			m.slavePool.AddSlave(&Slave{
				ip: p.Source.String(),
			})
		default:
			m.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet for connection"))
		}
	case <-time.After(constants.WaitForSlaveTimeout):

	}

}
