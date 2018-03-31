package utility

import (
	"errors"
	"net"
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
