package slave

import (
	"errors"
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
		n, _, err := connRecv.ReadFromUDP(buf[:])
		if err != nil {
			s.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			continue
		}

		err = packets.DecodePacket(buf[:n], &p)
		if err != nil {
			s.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			p.Ack = false
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

	ack := packets.BroadcastConnectResponse{
		Ack:  true,
		IP:   s.myIP,
		Port: myPort,
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
	return nil
}

func (s *Slave) initListeners() error {
	err := s.initInfoListener()
	if err != nil {
		return err
	}
	return nil
}

type tcpData struct {
	n   int
	buf [2048]byte
}

func (s *Slave) initInfoListener() error {
	ln, err := net.Listen("tcp", s.myIP.String()+":0")
	if err != nil {
		return err
	}

	go s.infoListenManager(ln)

	port := ln.Addr().(*net.TCPAddr).Port
	s.infoReqPort = uint16(port)
	return nil
}

func (s *Slave) infoListenManager(ln net.Listener) {
	s.closeWait.Add(1)

	// TODO: handle error in accept
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
			s.infoListener(conn, packetChan)
		}
	}

	s.closeWait.Done()
}

func (s *Slave) collectIncomingRequests(conn net.Conn, packetChan chan<- tcpData) {
	s.closeWait.Add(1)

	end := false
	for !end {
		select {
		case <-s.close:
			end = true
		default:
			var buf [2048]byte
			// TODO: add timeout
			n, err := conn.Read(buf[0:])
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Error in reading from TCP"))
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

	s.closeWait.Done()
}

func (s *Slave) infoListener(conn net.Conn, packetChan <-chan tcpData) {

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
		case packets.InfoRequest:
			var p packets.InfoRequestPacket
			err := packets.DecodePacket(packet.buf[:packet.n], &p)
			if err != nil {
				s.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
					"packet", packetType.String(), "err", err.Error()))
				return
			}

			// TODO: respond with load

		default:
			s.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet for connection"))
		}

	// Timeout
	case <-time.After(constants.WaitForInfoReqTimeout):

	}

}

// TODO: do like req listener. Check with Sukrut.
func (s *Slave) reqListener() {
	s.closeWait.Add(1)
	s.closeWait.Done()
}
