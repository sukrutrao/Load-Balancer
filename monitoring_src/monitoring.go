package monitoring

import (
	"net"
	"sync"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"github.com/op/go-logging"
)

// Master is used to store info of master node
type Master struct {
	ip net.IP
}

type Monitor struct {
	myIP        net.IP
	broadcastIP net.IP
	reqSendPort uint16
	master      Master

	Logger *logging.Logger

	slaveIPs []string
	mtx      sync.RWMutex

	close     chan struct{}
	closeWait sync.WaitGroup
}

func (mo *Monitor) initDS() {
	mo.close = make(chan struct{})
}

func (mo *Monitor) Run() {
	mo.initDS()
	mo.updateAddress()
	mo.Logger.Info(logger.FormatLogMessage("msg", "Monitor running"))
	mo.connect()
	mo.closeWait.Wait()
}

func (mo *Monitor) updateAddress() {
	ipnet, err := utility.GetMyIP()
	if err != nil {
		mo.Logger.Fatal(logger.FormatLogMessage("msg", "Failed to get IP", "err", err.Error()))
	}

	mo.myIP = ipnet.IP
	for i, b := range ipnet.Mask {
		mo.broadcastIP = append(mo.broadcastIP, (mo.myIP[i] | (^b)))
	}
}

func (mo *Monitor) UpdateSlaveIPs(slaveIPs []string) {
	mo.mtx.Lock()
	defer mo.mtx.Unlock()
	mo.slaveIPs = slaveIPs
}

func (mo *Monitor) UpdateGrafana() {
	mo.mtx.RLock()
	defer mo.mtx.RUnlock()
}
