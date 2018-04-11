package slave

import (
	"net"
	"strconv"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
	"github.com/GoodDeeds/load-balancer/common/utility"
)

func (s *Slave) connect() {

	udpAddr, err := net.ResolveUDPAddr("udp4", s.broadcastIP.String()+":"+strconv.Itoa(int(constants.MasterBroadcastPort)))
	utility.CheckFatal(err, s.Logger)
	conn, err := net.DialUDP("udp", nil, udpAddr)
	utility.CheckFatal(err, s.Logger)

	udpAddr, err = net.ResolveUDPAddr("udp4", s.myIP.String()+":"+strconv.Itoa(int(constants.SlaveBroadcastPort)))
	utility.CheckFatal(err, s.Logger)
	conn2, err := net.ListenUDP("udp", udpAddr)
	utility.CheckFatal(err, s.Logger)

	pkt := packets.BroadcastConnectRequest{
		Source: s.myIP,
		Port:   constants.SlaveBroadcastPort,
	}
	encodedBytes, err := packets.EncodePacket(pkt, packets.ConnectionRequest)
	utility.CheckFatal(err, s.Logger)

	// TODO: Retry for some time
	// TODO: Retry with timeout
	_, err = conn.Write(encodedBytes)
	utility.CheckFatal(err, s.Logger)

	var buf [512]byte
	n, _, err := conn2.ReadFromUDP(buf[0:])

	s.Logger.Info(logger.FormatLogMessage("msg", "Connection response", "rsp", string(buf[0:n])))

	// TODO: send an ACK
}
