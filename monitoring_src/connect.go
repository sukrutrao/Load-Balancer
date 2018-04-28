package monitoring

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

func (mo *Monitor) connect() error {

	// TODO: store the ip of the master

	// TODO: create a TCP connection for info and requests.
	err := mo.initListeners()
	utility.CheckFatal(err, mo.Logger)

	udpAddr := &net.UDPAddr{
		IP:   mo.broadcastIP,
		Port: int(constants.MasterBroadcastPort),
	}
	conn, err := net.DialUDP("udp", nil, udpAddr)
	utility.CheckFatal(err, mo.Logger)

	udpAddr = &net.UDPAddr{
		IP:   mo.myIP,
		Port: 0,
	}
	connRecv, err := net.ListenUDP("udp", udpAddr)
	utility.CheckFatal(err, mo.Logger)
	myPort := utility.PortFromUDPConn(connRecv)

	tries := 0
	var p packets.BroadcastConnectResponse
	backoff := constants.ConnectRetryBackoffBaseTime
	for !p.Ack && tries < constants.MaxConnectRetry {

		pkt := packets.BroadcastConnectRequest{
			Source: mo.myIP,
			Port:   myPort,
		}
		encodedBytes, err := packets.EncodePacket(pkt, packets.MonitorConnectionRequest)
		utility.CheckFatal(err, mo.Logger)

		_, err = conn.Write(encodedBytes)
		utility.CheckFatal(err, mo.Logger)

		var buf [2048]byte
		// TODO: add timeout
		n, _, err := connRecv.ReadFromUDP(buf[:])
		if err != nil {
			mo.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			continue
		}

		err = packets.DecodePacket(buf[:n], &p)
		if err != nil {
			mo.Logger.Error(logger.FormatLogMessage("err", err.Error()))
			p.Ack = false
			continue
		}

		tries++

		if !p.Ack {
			mo.Logger.Warning(logger.FormatLogMessage("msg", "Got a NAC for connection request.", "try", strconv.Itoa(tries)))
			if tries < constants.MaxConnectRetry {
				time.Sleep(backoff)
				backoff = backoff * 2
			}
		}

	}

	if !p.Ack {
		return errors.New("Failed to connect to Master")
	}

	mo.master.ip = p.IP

	ack := packets.BroadcastConnectResponse{
		Ack:         true,
		IP:          mo.myIP,
		Port:        myPort,
		ReqSendPort: mo.reqSendPort,
	}
	ackBytes, err := packets.EncodePacket(ack, packets.MonitorConnectionAck)
	utility.CheckFatal(err, mo.Logger)
	for i := 0; i < constants.NumBurstAcks; i++ {
		_, err = conn.Write(ackBytes)
		if err != nil {
			if i == 0 {
				mo.Logger.Critical(logger.FormatLogMessage("msg", "Failed to send Ack", "err", err.Error()))
				os.Exit(1)
			} else {
				mo.Logger.Warning(logger.FormatLogMessage("msg", "Failed to send some Acks", "err", err.Error()))
			}
		}
	}

	mo.Logger.Info(logger.FormatLogMessage("msg", "Connection response", "ack", strconv.FormatBool(p.Ack), "server_ip", p.IP.String()))
	return nil
}

// Listeners.

func (mo *Monitor) initListeners() error {
	err := mo.initReqListener()
	if err != nil {
		return err
	}
	return nil
}

type tcpData struct {
	n   int
	buf [2048]byte
}

/// Request listener

func (mo *Monitor) initReqListener() error {
	ln, err := net.Listen("tcp", mo.myIP.String()+":0")
	if err != nil {
		return err
	}

	mo.closeWait.Add(1)
	go mo.reqListenManager(ln)

	port := ln.Addr().(*net.TCPAddr).Port
	mo.reqSendPort = uint16(port)
	return nil
}

func (mo *Monitor) reqListenManager(ln net.Listener) {
	defer mo.closeWait.Done()

	// TODO: handle error in accept
	ln.(*net.TCPListener).SetDeadline(time.Now().Add(constants.MonitorConnectionAcceptTimeout))
	conn, err := ln.Accept()
	if err != nil {
		return
	}

	mo.closeWait.Add(1)
	go mo.reqRecvAndUpdater(conn)

	// TODO: send requests from time to time
	end := false
	for !end {
		select {
		case <-mo.close:
			end = true
		default:
			packet := packets.MonitorRequestPacket{}
			bytes, err := packets.EncodePacket(packet, packets.MonitorRequest)
			if err != nil {
				mo.Logger.Warning(logger.FormatLogMessage("msg", "Failed to encode packet"))
			}

			_, err = conn.Write(bytes)
			if err != nil {
				mo.Logger.Warning(logger.FormatLogMessage("msg", "Failed to send Req packet"))
			}

			<-time.After(constants.MonitorRequestInterval)
		}
	}

}

func (mo *Monitor) reqRecvAndUpdater(conn net.Conn) {

	end := false
	for !end {
		select {
		case <-mo.close:
			end = true
		default:
			// NOTE: make sure this size can fit the max slaves.
			var buf [2048]byte
			// TODO: add timeout
			conn.SetReadDeadline(time.Now().Add(constants.MonitorReceiveTimeout))
			n, err := conn.Read(buf[0:])
			if err != nil {
				mo.Logger.Error(logger.FormatLogMessage("msg", "Error in reading from TCP", "err", err.Error()))
				if err == io.EOF {
					close(mo.close)
					end = true
				}
				continue
			}

			packetType, err := packets.GetPacketType(buf[:n])
			if err != nil {
				mo.Logger.Error(logger.FormatLogMessage("err", err.Error()))
				return
			}

			switch packetType {
			case packets.MonitorResponse:
				var p packets.MonitorResponsePacket
				err := packets.DecodePacket(buf[:n], &p)
				if err != nil {
					mo.Logger.Error(logger.FormatLogMessage("msg", "Failed to decode packet",
						"packet", packetType.String(), "err", err.Error()))
					return
				}

				if updated, added, deleted := mo.UpdateSlaveIPs(p.SlaveIPs); updated {
					go mo.UpdateGrafana(added, deleted)
				}

			default:
				mo.Logger.Warning(logger.FormatLogMessage("msg", "Received invalid packet"))
			}
		}
	}

	mo.closeWait.Done()
}
