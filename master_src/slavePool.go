package master

import (
	"net"
	"strconv"
	"sync"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/op/go-logging"
)

type SlavePool struct {
	mtx    sync.RWMutex
	slaves []*Slave
	Logger *logging.Logger
}

func (sp *SlavePool) NumSlaves() int {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	return len(sp.slaves)
}

func (sp *SlavePool) AddSlave(slave *Slave) {
	slave.Logger = sp.Logger
	slave.InitConnections()
	slave.InitDS()
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

func (sp *SlavePool) SlaveExists(ip net.IP, id uint16) bool {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	ipStr := ip.String()
	for _, slave := range sp.slaves {
		if slave.ip == ipStr && slave.id == id {
			return true
		}
	}
	return false
}

func (sp *SlavePool) GetAllSlaveIPs() []string {
	sp.mtx.RLock()
	defer sp.mtx.RUnlock()
	var ips []string

	for _, s := range sp.slaves {
		ips = append(ips, s.ip)
	}

	return ips
}
func (sp *SlavePool) Close(log *logging.Logger) {
	// close all go routines/listeners
	log.Info(logger.FormatLogMessage("msg", "Closing Slave Pool"))
	for _, s := range sp.slaves {
		s.Close()
	}
}

func (sp *SlavePool) gc(log *logging.Logger) []*Slave {
	toRemove := []int{}
	for i, slave := range sp.slaves {
		select {
		case <-slave.close:
			slave.Close()
			toRemove = append(toRemove, i)
		default:
		}
	}

	var slaves []*Slave
	if len(toRemove) > 0 {
		sp.mtx.Lock()
		defer sp.mtx.Unlock()

		for i, idx := range toRemove {
			log.Info(logger.FormatLogMessage("msg", "Slave removed in gc",
				"slave_ip", sp.slaves[idx-i].ip, "slave_id", strconv.Itoa(int(sp.slaves[idx-i].id))))

			slaves = append(slaves, sp.slaves[idx-i])
			sp.slaves = append(sp.slaves[:idx-i], sp.slaves[idx+1-i:]...)

		}

	}

	return slaves
}
