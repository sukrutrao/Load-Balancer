package slave

import (
	"fmt"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"net"
	"os"
)

type Master struct {
	ip string
}

type Slave struct {
	myIP        net.IP
	broadcastIP net.IP
	close       chan struct{}
}

func (s *Slave) Run() {
	fmt.Println("Slave running")
	s.updateAddress()
	s.connect()
}

func (s *Slave) updateAddress() {

	ipnet, err := utility.GetMyIP()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
		os.Exit(1)
	}

	s.myIP = ipnet.IP

	for i, b := range ipnet.Mask {
		s.broadcastIP = append(s.broadcastIP, (s.myIP[i] | (^b)))
	}

}
