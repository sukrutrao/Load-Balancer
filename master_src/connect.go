package master

import (
	"net"
	"reflect"
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
	end := false
	for !end {
		select {
		case <-m.close:
			end = true
		default:
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
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
					"packet", packetType.String(), "err", err.Error()))
				return
			}

			portStr := strconv.Itoa(int(p.Port))
			m.Logger.Info(logger.FormatLogMessage("msg", "Connection request", "ip", p.Source.String(), "port", portStr))

			isAck := false

			if m.slavePool.NumSlaves() < constants.MaxSlaves {
				// TODO: instead of ip, check ip and port combination.
				if !m.SlaveExists(p.Source, p.Port) {
					isAck = true
					m.unackedSlaveMtx.Lock()
					if _, ok := m.unackedSlaves[p.Source.String()+":"+portStr]; !ok {
						m.unackedSlaves[p.Source.String()+":"+portStr] = struct{}{}
					}
					m.unackedSlaveMtx.Unlock()
				} else {
					m.Logger.Warning(logger.FormatLogMessage("msg", "Multiple request for connection", "ip", p.Source.String()))
				}
			} else {
				m.Logger.Warning(logger.FormatLogMessage("msg", "Connection request after max slave limit"))
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
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
					"packet", packetType.String(), "err", err.Error()))
				return
			}

			portStr := strconv.Itoa(int(p.Port))

			m.unackedSlaveMtx.Lock()
			if _, ok := m.unackedSlaves[p.IP.String()+":"+portStr]; ok {
				delete(m.unackedSlaves, p.IP.String()+":"+portStr)
				m.unackedSlaveMtx.Unlock()
				m.slavePool.AddSlave(&Slave{
					ip:          p.IP.String(),
					id:          p.Port,
					loadReqPort: p.LoadReqPort,
					reqSendPort: p.ReqSendPort,
				})
				m.Logger.Info(logger.FormatLogMessage("msg", "Connection request granted", "ip", p.IP.String(), "port", portStr))
			} else {
				m.unackedSlaveMtx.Unlock()
			}

		case packets.MonitorConnectionRequest:
			// Processing request.
			var p packets.BroadcastConnectRequest
			err := packets.DecodePacket(packet.buf[:packet.n], &p)
			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
					"packet", packetType.String(), "err", err.Error()))
				return
			}

			portStr := strconv.Itoa(int(p.Port))
			m.Logger.Info(logger.FormatLogMessage("msg", "Monitor connection request", "ip", p.Source.String(), "port", portStr))

			isAck := false

			if !m.monitor.acked && (!reflect.DeepEqual(m.monitor.ip, p.Source) || m.monitor.id != p.Port) {
				isAck = true
				m.monitor.id = p.Port
				m.monitor.ip = p.Source
				m.monitor.acked = false
				m.monitor.close = make(chan struct{})
			} else {
				m.Logger.Warning(logger.FormatLogMessage("msg", "Multiple request for monitor connection", "ip", p.Source.String()))
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
			ackBytes, err := packets.EncodePacket(ack, packets.MonitorConnectionResponse)
			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to send Ack for connection", "err", err.Error()))
				return
			}
			conn.WriteToUDP(ackBytes, addr)

		case packets.MonitorConnectionAck:

			var p packets.BroadcastConnectResponse
			err := packets.DecodePacket(packet.buf[:packet.n], &p)
			if err != nil {
				m.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
					"packet", packetType.String(), "err", err.Error()))
				return
			}

			portStr := strconv.Itoa(int(p.Port))

			if !m.monitor.acked && reflect.DeepEqual(m.monitor.ip, p.IP) && m.monitor.id == p.Port {
				m.monitor.acked = true
				m.monitor.reqSendPort = p.ReqSendPort
				if err := m.StartMonitor(); err != nil {
					m.monitor.acked = false
					m.Logger.Error(logger.FormatLogMessage("msg", "Failed to start the monitor", "err", err.Error()))
				} else {
					m.Logger.Info(logger.FormatLogMessage("msg", "Monitor connection request granted", "ip", p.IP.String(), "port", portStr))
				}
			} else if !reflect.DeepEqual(m.monitor.ip, p.IP) || m.monitor.id != p.Port {
				m.Logger.Warning(logger.FormatLogMessage("msg", "Another Monitor request", "ip", p.IP.String(), "port", portStr))
			}

		default:
			m.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet for connection"))
		}

	// Timeout
	case <-time.After(constants.WaitForSlaveTimeout):

	}

}
