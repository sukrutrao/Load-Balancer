package master

import (
	"errors"
	//	"fmt"
)

type LeastDifference struct {
	*LoadBalancerBase
}

// TODO - need locks here?
func (l *LeastDifference) assignTask(load int) (*Slave, error) {
	// TODO - need to lock Slavepool!!!
	l.slavePool.mtx.RLock()
	defer l.slavePool.mtx.RUnlock()

	if len(l.slavePool.slaves) == 0 {
		return nil, errors.New("No Slaves available")
	}
	minDifference := l.slavePool.slaves[0].maxLoad - l.slavePool.slaves[0].currentLoad
	minId := -1
	for id := 0; id < len(l.slavePool.slaves); id++ {
		if l.slavePool.slaves[id].maxLoad-l.slavePool.slaves[id].currentLoad <= minDifference && l.slavePool.slaves[id].maxLoad-l.slavePool.slaves[id].currentLoad >= load {
			minDifference = l.slavePool.slaves[id].currentLoad
			minId = id
		}
	}
	if minId >= 0 {
		l.slavePool.slaves[minId].currentLoad += load // TODO do we need this?
		return l.slavePool.slaves[minId], nil
	}
	return nil, errors.New("No Slaves available for this load")
}
