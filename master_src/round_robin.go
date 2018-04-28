package master

import (
	"errors"
	"fmt"
)

type RoundRobin struct {
	*LoadBalancerBase
	lastAssigned int
}

// TODO - need locks here?
func (r *RoundRobin) assignTask(load int) (*Slave, error) {
	// TODO - need to lock Slavepool!!!
	if len(r.slavePool.slaves) == 0 {
		return nil, errors.New("No Slaves available")
	}
	if len(r.slavePool.slaves) == 1 {
		fmt.Println("Number of slaves is 1")
		if r.slavePool.slaves[0].currentLoad+load <= r.slavePool.slaves[0].maxLoad {
			r.slavePool.slaves[0].currentLoad += load
			r.lastAssigned = 0
			return r.slavePool.slaves[0], nil
		}
	}
	nextIdTry := (r.lastAssigned + 1) % len(r.slavePool.slaves)
	if r.slavePool.slaves[nextIdTry].currentLoad+load <= r.slavePool.slaves[nextIdTry].maxLoad {
		r.slavePool.slaves[nextIdTry].currentLoad += load
		r.lastAssigned = nextIdTry
		return r.slavePool.slaves[nextIdTry], nil
	}
	for id := (nextIdTry + 1) % len(r.slavePool.slaves); id != nextIdTry; id = (id + 1) % len(r.slavePool.slaves) {
		if r.slavePool.slaves[id].currentLoad+load <= r.slavePool.slaves[id].maxLoad {
			r.slavePool.slaves[id].currentLoad += load
			// TODO - not changed lastAssigned here
			return r.slavePool.slaves[id], nil
		}
	}
	return nil, errors.New("No Slaves available for this load")
}
