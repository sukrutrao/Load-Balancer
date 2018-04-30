package utility

import (
	"errors"
	"net"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/op/go-logging"
)

var ipNotFoundError error = errors.New("IP not found")

func GetMyIP() (*net.IPNet, error) {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return &net.IPNet{}, err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			ipnet.IP = ipnet.IP.To4()
			if ipnet.IP != nil {
				return ipnet, nil
			}
		}
	}

	return &net.IPNet{}, ipNotFoundError

}

func CheckFatal(err error, log *logging.Logger) {
	if err != nil {
		log.Fatal(logger.FormatLogMessage("err", err.Error()))
	}
}

func PortFromUDPConn(udpConn *net.UDPConn) uint16 {
	return uint16(udpConn.LocalAddr().(*net.UDPAddr).Port)
}
