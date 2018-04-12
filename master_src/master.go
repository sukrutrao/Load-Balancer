package master

import (
	"net"
	"sync"
	"time"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/utility"
	"github.com/op/go-logging"
)

// Master is used to store info of master node which is currently running
type Master struct {
	myIP        net.IP
	broadcastIP net.IP
	slavePool   SlavePool
	Logger      *logging.Logger

	unackedSlaves   map[string]struct{}
	unackedSlaveMtx sync.RWMutex

	close     chan struct{}
	closeWait sync.WaitGroup
}

func (m *Master) initDS() {
	m.close = make(chan struct{})
	m.unackedSlaves = make(map[string]struct{})
}

func (m *Master) Run() {
	m.initDS()
	m.updateAddress()
	go m.connect()
	m.Logger.Info(logger.FormatLogMessage("msg", "Master running"))

	// TODO: this sleep is just for simulation
	// this should be replaced with load balancing.
	time.Sleep(10 * time.Second)
	m.Close()
}

func (m *Master) updateAddress() {
	ipnet, err := utility.GetMyIP()
	if err != nil {
		m.Logger.Fatal(logger.FormatLogMessage("msg", "Failed to get IP", "err", err.Error()))
	}

	m.myIP = ipnet.IP
	for i, b := range ipnet.Mask {
		m.broadcastIP = append(m.broadcastIP, (m.myIP[i] | (^b)))
	}
}

func (m *Master) SlaveIpExists(ip net.IP) bool {
	return m.slavePool.SlaveIpExists(ip)
}

func (m *Master) Close() {
	m.Logger.Info(logger.FormatLogMessage("msg", "Closing Master gracefully..."))
	close(m.close)
	m.closeWait.Wait()
}

// Slave is used to store info of slave node connected to it
type Slave struct {
	ip string

	// Need to acquire Write Lock which modifying any value.
	mtx sync.RWMutex
}

type SlavePool struct {
	mtx    sync.RWMutex
	slaves []*Slave
}

func (sp *SlavePool) AddSlave(slave *Slave) {
	sp.mtx.Lock()
	defer sp.mtx.Unlock()
	sp.slaves = append(sp.slaves, slave)
}

func (sp *SlavePool) RemoveSlave(ip string) bool {
	sp.mtx.Lock()
	defer sp.mtx.Unlock()
	toRemove := -1
	for i, slave := range sp.slaves {
		if slave.ip == ip {
			toRemove = i
			break
		}
	}

	if toRemove < 0 {
		return false
	}

	sp.slaves = append(sp.slaves[:toRemove], sp.slaves[toRemove+1:]...)
	return true
}

func (sp *SlavePool) SlaveIpExists(ip net.IP) bool {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	ipStr := ip.String()
	for _, slave := range sp.slaves {
		if slave.ip == ipStr {
			return true
		}
	}
	return false
}
