package slave

import (
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/GoodDeeds/load-balancer/common/utility"
)

func (s *Slave) connect() error {

	// TODO: store the ip of the master

	// TODO: create a TCP connection for info and requests.
	err := s.initListeners()
	utility.CheckFatal(err, s.Logger)

	udpAddr := &net.UDPAddr{
		IP:   s.broadcastIP,
		Port: int(constants.MasterBroadcastPort),
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	utility.CheckFatal(err, s.Logger)

	udpAddr = &net.UDPAddr{
		IP:   s.myIP,
		Port: 0,
	}
	connRecv, err := net.ListenUDP("udp", udpAddr)
	utility.CheckFatal(err, s.Logger)
	myPort := utility.PortFromUDPConn(connRecv)

	tries := 0
	var p packets.BroadcastConnectResponse
	backoff := constants.ConnectRetryBackoffBaseTime
	for !p.Ack && tries < constants.MaxConnectRetry {

		pkt := packets.BroadcastConnectRequest{
			Source: s.myIP,
			Port:   myPort,
		}
		encodedBytes, err := packets.EncodePacket(pkt, packets.ConnectionRequest)
		utility.CheckFatal(err, s.Logger)

		_, err = conn.Write(encodedBytes)
		utility.CheckFatal(err, s.Logger)

		var buf [2048]byte
		// TODO: add timeout
		connRecv.SetReadDeadline(time.Now().Add(constants.ReceiveTimeout))
		n, _, err := connRecv.ReadFromUDP(buf[:])
		if err != nil {
			s.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			tries++
			continue
		}

		err = packets.DecodePacket(buf[:n], &p)
		if err != nil {
			s.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			p.Ack = false
			tries++
			continue
		}

		tries++

		if !p.Ack {
			s.Logger.Warning(logger.FormatLogMessage("msg", "Got a NAC for connection request.", "try", strconv.Itoa(tries)))
			if tries < constants.MaxConnectRetry {
				time.Sleep(backoff)
				backoff = backoff * 2
			}
		}

	}

	if !p.Ack {
		return errors.New("Failed to connect to Master")
	}

	s.master.ip = p.IP

	ack := packets.BroadcastConnectResponse{
		Ack:         true,
		IP:          s.myIP,
		Port:        myPort,
		LoadReqPort: s.loadReqPort,
		ReqSendPort: s.reqSendPort,
	}
	ackBytes, err := packets.EncodePacket(ack, packets.ConnectionAck)
	utility.CheckFatal(err, s.Logger)
	for i := 0; i < constants.NumBurstAcks; i++ {
		_, err = conn.Write(ackBytes)
		if err != nil {
			if i == 0 {
				s.Logger.Critical(logger.FormatLogMessage("msg", "Failed to send Ack", "err", err.Error()))
				os.Exit(1)
			} else {
				s.Logger.Warning(logger.FormatLogMessage("msg", "Failed to send some Acks", "err", err.Error()))
			}
		}
	}

	s.Logger.Info(logger.FormatLogMessage("msg", "Connection response", "ack", strconv.FormatBool(p.Ack), "server_ip", p.IP.String()))
	s.closeWait.Done()
	return nil
}

// Listeners.

func (s *Slave) initListeners() error {
	err := s.initLoadListener()
	if err != nil {
		return err
	}
	err = s.initReqListener()
	if err != nil {
		return err
	}
	s.Logger.Info(logger.FormatLogMessage("loadReqPort", strconv.Itoa(int(s.loadReqPort)), "reqSendPort", strconv.Itoa(int(s.reqSendPort))))
	return nil
}

type tcpData struct {
	n   int
	buf [2048]byte
}

func (s *Slave) collectIncomingRequests(conn net.Conn, packetChan chan<- tcpData) {
	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			var buf [2048]byte
			// TODO: add timeout
			conn.SetReadDeadline(time.Now().Add(constants.SlaveReceiveTimeout))
			n, err := conn.Read(buf[0:])
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Error in reading from TCP", "err", err.Error()))
				if err == io.EOF {
					select {
					case <-s.close:
					default:
						close(s.close)
					}
					end = true
				}
				continue
			}

			var bufCopy [2048]byte
			copy(bufCopy[:], buf[:])
			packetChan <- tcpData{
				n:   n,
				buf: bufCopy,
			}
		}
	}

}

/// Load listener

func (s *Slave) initLoadListener() error {
	ln, err := net.Listen("tcp", s.myIP.String()+":0")
	if err != nil {
		return err
	}

	s.closeWait.Add(1)
	go s.loadListenManager(ln)

	port := ln.Addr().(*net.TCPAddr).Port
	s.loadReqPort = uint16(port)
	return nil
}

func (s *Slave) loadListenManager(ln net.Listener) {

	// TODO: handle error in accept
	ln.(*net.TCPListener).SetDeadline(time.Now().Add(constants.SlaveConnectionAcceptTimeout))
	conn, _ := ln.Accept()

	packetChan := make(chan tcpData)
	go s.collectIncomingRequests(conn, packetChan)

	end := false
	for !end {
		select {
		case <-s.close:
			s.Logger.Info(logger.FormatLogMessage("msg", "Stopping Info Listener"))
			end = true
			break
		default:
			s.loadListener(conn, packetChan)
		}
	}

	s.closeWait.Done()
}

func (s *Slave) loadListener(conn net.Conn, packetChan <-chan tcpData) {

	select {
	case packet, ok := <-packetChan:
		if !ok {
			break
		}

		packetType, err := packets.GetPacketType(packet.buf[:packet.n])
		if err != nil {
			s.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			return
		}
		switch packetType {
		case packets.LoadRequest:
			var p packets.LoadRequestPacket
			err := packets.DecodePacket(packet.buf[:packet.n], &p)
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
					"packet", packetType.String(), "err", err.Error()))
				return
			}

			// TODO: respond with proper load
			res := packets.LoadResponsePacket{
				Timestamp: time.Now(),
				Load:      3.1415,
			}

			bytes, err := packets.EncodePacket(res, packets.LoadResponse)
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Failed to encode packet",
					"packet", packets.LoadResponse.String(), "err", err.Error()))
				return
			}

			_, err = conn.Write(bytes)
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Failed to send packet",
					"packet", packets.LoadResponse.String(), "err", err.Error()))
				return
			}

		default:
			s.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet"))
		}

	// Timeout
	case <-time.After(constants.WaitForReqTimeout):

	}

}

/// Request listener

func (s *Slave) initReqListener() error {
	ln, err := net.Listen("tcp", s.myIP.String()+":0")
	if err != nil {
		return err
	}

	s.closeWait.Add(1)
	go s.reqListenManager(ln)

	port := ln.Addr().(*net.TCPAddr).Port
	s.reqSendPort = uint16(port)
	return nil
}

func (s *Slave) reqListenManager(ln net.Listener) {

	// TODO: handle error in accept
	ln.(*net.TCPListener).SetDeadline(time.Now().Add(constants.SlaveConnectionAcceptTimeout))
	conn, _ := ln.Accept()

	packetChan := make(chan tcpData)
	go s.collectIncomingRequests(conn, packetChan)
	go s.sendChannelHandler(conn)

	end := false
	for !end {
		select {
		case <-s.close:
			s.Logger.Info(logger.FormatLogMessage("msg", "Stopping Request Listener"))
			end = true
			break
		default:
			s.reqListener(conn, packetChan)
		}
	}

	s.closeWait.Done()
}

func (s *Slave) reqListener(conn net.Conn, packetChan <-chan tcpData) {

	select {
	case packet, ok := <-packetChan:
		if !ok {
			break
		}

		packetType, err := packets.GetPacketType(packet.buf[:])
		if err != nil {
			s.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			return
		}

		switch packetType {
		// TODO: call functions from Sukrut
		// Structure
		// case packets.ThePacketType:
		// Get packet from bytes and call appropriate function.
		case packets.TaskRequest:
			var p packets.TaskRequestPacket
			err := packets.DecodePacket(packet.buf[:packet.n], &p)
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
					"packet", packetType.String(), "err", err.Error()))
				return
			}
			go s.getTask(p)
		case packets.TaskStatusRequest:
			var p packets.TaskStatusRequestPacket
			err := packets.DecodePacket(packet.buf[:packet.n], &p)
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
					"packet", packetType.String(), "err", err.Error()))
				return
			}
			go s.respondTaskStatusPacket(p)
		default:
			s.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet"))
		}

	// Timeout
	case <-time.After(constants.WaitForReqTimeout):

	}

}

func (s *Slave) sendChannelHandler(conn net.Conn) {
	s.closeWait.Add(1)
	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			pt := <-s.sendChan
			bytes, err := packets.EncodePacket(pt.Packet, pt.PacketType)
			if err != nil {
				if err == io.EOF {
					// TODO: remove myself from slavepool
					s.Logger.Warning(logger.FormatLogMessage("msg", "Closing slave"))
					close(s.close)
					end = true
				}
				continue
			}
			_, err = conn.Write(bytes)
			if err != nil {
				s.Logger.Warning(logger.FormatLogMessage("msg", "Failed to send packet", "err", err.Error()))
			}

			<-time.After(constants.TaskInterval)

		}
	}
	s.closeWait.Done()
}
