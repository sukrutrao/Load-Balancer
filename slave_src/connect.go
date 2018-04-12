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

	udpAddr, err := net.ResolveUDPAddr("udp4", s.broadcastIP.String()+":"+strconv.Itoa(int(constants.MasterBroadcastPort)))
	utility.CheckFatal(err, s.Logger)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	utility.CheckFatal(err, s.Logger)

	udpAddr, err = net.ResolveUDPAddr("udp4", s.myIP.String()+":"+strconv.Itoa(int(constants.SlaveBroadcastPort)))
	utility.CheckFatal(err, s.Logger)
	connRecv, err := net.ListenUDP("udp", udpAddr)
	utility.CheckFatal(err, s.Logger)

	tries := 0
	var p packets.BroadcastConnectResponse
	backoff := constants.ConnectRetryBackoffBaseTime
	for !p.Ack && tries < constants.MaxConnectRetry {

		pkt := packets.BroadcastConnectRequest{
			Source: s.myIP,
			Port:   constants.SlaveBroadcastPort,
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
		Port: constants.SlaveBroadcastPort,
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
