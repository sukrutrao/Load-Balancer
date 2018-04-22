package algorithm

type LoadBalancerFunctions interface {
	addSlave(slave *slave.Slave) (*Slave, error)
	removeSlave(id int) error
	assignTask(load int) (*Slave, error)
	// may need more functions for more specialized kind of assignments
}

type LoadBalancerBase struct {
	Slaves map[int]struct{} // TODO init this
}

type Slave struct {
	SlaveId     int
	currentLoad int
	maxLoad     int
	SlaveObject *slave.Slave
}

func (l *LoadBalancerBase) addSlave(slave *slave.Slave) (*Slave, error) {
	currentLoad := slave.currentLoad
	maxLoad := slave.maxLoad
	s := Slave{SlaveId: len(l.Slaves), currentLoad: currentLoad, maxLoad: maxLoad, SlaveObject: slave}
	// TODO check for errors
	l.Slaves[s.SlaveId] = s
	return s, nil
}

func (l *LoadBalancerBase) removeSlave(id int) error {
	delete(l.Slaves, id) // TODO - any errors? maybe not
	return nil
}

// TODO - I am not sure if all functions need to be implemented, so a dummy
func (l *LoadBalancerBase) assignTask(load int) (*Slave, error) {
	for key := range l.Slaves {
		if l.Slaves[key].currentLoad+load <= l.Slaves[key].maxLoad {
			l.Slaves[key].currentLoad += load
			return l.Slaves[key], nil
		}
	}
	return nil, "No Slave can currently take this load"
}
