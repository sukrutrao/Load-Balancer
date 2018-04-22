package algorithm

type RoundRobin struct {
	*LoadBalancerBase
	lastAssigned int
}

// TODO - need locks here?
func (r *RoundRobin) assignTask(load int) (*Slave, error) {
	nextIdTry := (lastAssigned + 1) % len(r.Slaves)
	if r.Slaves[nextIdTry].currentLoad+load <= r.Slaves[nextIdTry].maxLoad {
		r.Slaves[nextIdTry].currentLoad += load
		r.lastAssigned = nextIdTry
		return r.Slaves[nextIdTry], nil
	}
	for id := (nextIdTry + 1) % len(r.Slaves); id != nextIdTry; id = (id + 1) % len(r.Slaves) {
		if r.Slaves[id].currentLoad+load <= r.Slaves[id].maxLoad {
			r.Slaves[id].currentLoad += load
			// TODO - not changed lastAssigned here
			return r.Slaves[id], nil
		}
	}
	return nil, "No Slaves available for this load"
}
