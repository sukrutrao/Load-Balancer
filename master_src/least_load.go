package master

import (
	"errors"
	//	"fmt"
)

type LeastLoad struct {
	*LoadBalancerBase
}

// TODO - need locks here?
func (l *LeastLoad) assignTask(load uint64) (*Slave, error) {
	// TODO - need to lock Slavepool!!!
	l.slavePool.mtx.RLock()
	defer l.slavePool.mtx.RUnlock()

	if len(l.slavePool.slaves) == 0 {
		return nil, errors.New("No Slaves available")
	}
	minLoad := l.slavePool.slaves[0].currentLoad
	minId := -1
	for id := 0; id < len(l.slavePool.slaves); id++ {
		if l.slavePool.slaves[id].currentLoad <= minLoad && l.slavePool.slaves[id].maxLoad >= load {
			minLoad = l.slavePool.slaves[id].currentLoad
			minId = id
		}
	}
	if minId >= 0 {
		l.slavePool.slaves[minId].currentLoad += load // TODO do we need this?
		return l.slavePool.slaves[minId], nil
	}
	return nil, errors.New("No Slaves available for this load")
}
