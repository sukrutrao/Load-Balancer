package master

import (
	"errors"
)

type LoadBalancerInterface interface {
	assignTask(load uint64) (*Slave, error)
}

type LoadBalancerBase struct {
	slavePool *SlavePool
}

type FirstAvailable struct {
	*LoadBalancerBase
}

func (l *FirstAvailable) assignTask(load uint64) (*Slave, error) {
	l.slavePool.mtx.RLock()
	defer l.slavePool.mtx.RUnlock()
	for i := range l.slavePool.slaves {
		if l.slavePool.slaves[i].currentLoad+load <= l.slavePool.slaves[i].maxLoad {
			return l.slavePool.slaves[i], nil
		}
	}
	return nil, errors.New("No Slave can currently take this load")
}

type RoundRobin struct {
	*LoadBalancerBase
	lastAssigned int
}

func (r *RoundRobin) assignTask(load uint64) (*Slave, error) {
	r.slavePool.mtx.RLock()
	defer r.slavePool.mtx.RUnlock()

	if len(r.slavePool.slaves) == 0 {
		return nil, errors.New("No Slaves available")
	}
	if len(r.slavePool.slaves) == 1 {
		if r.slavePool.slaves[0].currentLoad+load <= r.slavePool.slaves[0].maxLoad {
			r.lastAssigned = 0
			return r.slavePool.slaves[0], nil
		}
	}
	nextIdTry := (r.lastAssigned + 1) % len(r.slavePool.slaves)
	if r.slavePool.slaves[nextIdTry].currentLoad+load <= r.slavePool.slaves[nextIdTry].maxLoad {
		r.lastAssigned = nextIdTry
		return r.slavePool.slaves[nextIdTry], nil
	}
	for id := (nextIdTry + 1) % len(r.slavePool.slaves); id != nextIdTry; id = (id + 1) % len(r.slavePool.slaves) {
		if r.slavePool.slaves[id].currentLoad+load <= r.slavePool.slaves[id].maxLoad {
			return r.slavePool.slaves[id], nil
		}
	}
	return nil, errors.New("No Slaves available for this load")
}

type LeastDifference struct {
	*LoadBalancerBase
}

func (l *LeastDifference) assignTask(load uint64) (*Slave, error) {
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
		return l.slavePool.slaves[minId], nil
	}
	return nil, errors.New("No Slaves available for this load")
}

type LeastLoad struct {
	*LoadBalancerBase
}

func (l *LeastLoad) assignTask(load uint64) (*Slave, error) {
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
		return l.slavePool.slaves[minId], nil
	}
	return nil, errors.New("No Slaves available for this load")
}
