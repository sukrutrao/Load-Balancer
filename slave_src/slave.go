package slave

import (
	"net"
	"sync"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"github.com/op/go-logging"
)

// Master is used to store info of master node
type Master struct {
	ip string
}

// Slave is used to store info of slave node which is currently running
type Slave struct {
	myIP        net.IP
	broadcastIP net.IP
	loadReqPort uint16
	reqSendPort uint16
	master      Master

	Logger *logging.Logger

	close     chan struct{}
	closeWait sync.WaitGroup
}

func (s *Slave) initDS() {
	s.close = make(chan struct{})
}

// Run starts the slave
func (s *Slave) Run() {
	s.initDS()
	s.Logger.Info(logger.FormatLogMessage("msg", "Slave running"))
	s.updateAddress()
	s.connect()
	s.closeWait.Wait()
}

func (s *Slave) updateAddress() {
	ipnet, err := utility.GetMyIP()
	if err != nil {
		s.Logger.Fatal(logger.FormatLogMessage("msg", "Failed to get IP", "err", err.Error()))
	}

	s.myIP = ipnet.IP
	for i, b := range ipnet.Mask {
		s.broadcastIP = append(s.broadcastIP, (s.myIP[i] | (^b)))
	}
}

func (s *Slave) Close() {
	s.Logger.Info(logger.FormatLogMessage("msg", "Closing Slave gracefully..."))
	close(s.close)
	s.closeWait.Wait()
}
