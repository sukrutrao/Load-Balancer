package master

import (
	"errors"
)

// import (
// 	"github.com/GoodDeeds/load-balancer/master_src"
// )

type LoadBalancerInterface interface {
	// addSlave(slave *slave.Slave) (*Slave, error)
	// removeSlave(id int) error
	assignTask(load int) (*Slave, error)
	// may need more functions for more specialized kind of assignments
}

type LoadBalancerBase struct {
	slavePool *SlavePool
}

// func (l *LoadBalancerBase) addSlave(slave *Slave) (*Slave, error) {
// 	currentLoad := slave.currentLoad
// 	maxLoad := slave.maxLoad
// 	s := Slave{SlaveId: len(l.Slaves), currentLoad: currentLoad, maxLoad: maxLoad, SlaveObject: slave}
// 	// TODO check for errors
// 	l.Slaves[s.SlaveId] = s
// 	return s, nil
// }

// func (l *LoadBalancerBase) removeSlave(id int) error {
// 	delete(l.Slaves, id) // TODO - any errors? maybe not
// 	return nil
// }

// TODO - I am not sure if all functions need to be implemented, so a dummy
func (l *LoadBalancerBase) assignTask(load int) (*Slave, error) {
	for i := range l.slavePool.slaves {
		if l.slavePool.slaves[i].currentLoad+load <= l.slavePool.slaves[i].maxLoad {
			l.slavePool.slaves[i].currentLoad += load
			return l.slavePool.slaves[i], nil
		}
	}
	return nil, errors.New("No Slave can currently take this load")
}
