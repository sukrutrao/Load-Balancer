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
			m.Logger.Info(logger.FormatLogMessage("msg", "Stopping to accept new slave connections"))
			end = true
			break
		default:
			m.handleClient(conn, packetChan)
		}
	}
	m.closeWait.Done()
}

type connectionReqData struct {
	n    int
	addr *net.UDPAddr
	buf  [2048]byte
}

func (m *Master) collectIncomingRequests(conn *net.UDPConn, packetChan chan<- connectionReqData) {
	for {
		var buf [2048]byte
		n, addr, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			m.Logger.Error(logger.FormatLogMessage("msg", "Error in reading from UDP"))
			continue
		}

		var bufCopy [2048]byte
		copy(bufCopy[:], buf[:])
		packetChan <- connectionReqData{
			n:    n,
			addr: addr,
			buf:  bufCopy,
		}
	}
}

func (m *Master) handleClient(conn *net.UDPConn, packetChan <-chan connectionReqData) {

	select {
	case packet, ok := <-packetChan:
		if !ok {
			break
		}

		packetType, err := packets.GetPacketType(packet.buf[:])
		if err != nil {
			m.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			return
		}

		switch packetType {
		case packets.ConnectionRequest:
			// Processing request.
			var p packets.BroadcastConnectRequest
			err := packets.DecodePacket(packet.buf[:packet.n], &p)
			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet", "packet",
					"BroadcastConnectRequest", "err", err.Error()))
				return
			}

			portStr := strconv.Itoa(int(p.Port))
			m.Logger.Info(logger.FormatLogMessage("msg", "Connection request", "ip", p.Source.String(), "port", portStr))

			isAck := false

			if !m.SlaveIpExists(p.Source) {
				isAck = true
				m.unackedSlaveMtx.Lock()
				if _, ok := m.unackedSlaves[p.Source.String()+":"+portStr]; !ok {
					m.unackedSlaves[p.Source.String()+":"+portStr] = struct{}{}
				}
				m.unackedSlaveMtx.Unlock()
			}

			addr, err := net.ResolveUDPAddr("udp4", p.Source.String()+":"+portStr)
			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to resolve slave address", "err", err.Error()))
				return
			}

			// Sending ACK.
			ack := packets.BroadcastConnectResponse{
				Ack: isAck,
				IP:  m.myIP,
			}
			ackBytes, err := packets.EncodePacket(ack, packets.ConnectionResponse)
			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to send Ack for connection", "err", err.Error()))
				return
			}
			conn.WriteToUDP(ackBytes, addr)

		case packets.ConnectionAck:

			var p packets.BroadcastConnectResponse
			err := packets.DecodePacket(packet.buf[:packet.n], &p)
			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet", "packet",
					"BroadcastConnectResponse", "err", err.Error()))
				return
			}

			portStr := strconv.Itoa(int(p.Port))

			m.unackedSlaveMtx.Lock()
			if _, ok := m.unackedSlaves[p.IP.String()+":"+portStr]; ok {
				delete(m.unackedSlaves, p.IP.String()+":"+portStr)
				m.unackedSlaveMtx.Unlock()
				m.slavePool.AddSlave(&Slave{
					ip: p.IP.String(),
				})
				m.Logger.Info(logger.FormatLogMessage("msg", "Connection request granted", "ip", p.IP.String(), "port", portStr))
			} else {
				m.unackedSlaveMtx.Unlock()
			}

		default:
			m.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet for connection"))
		}

	// Timeout
	case <-time.After(constants.WaitForSlaveTimeout):

	}

}
