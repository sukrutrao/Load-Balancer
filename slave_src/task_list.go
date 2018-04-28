package slave

import (
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

func RunFibTask(t *packets.TaskPacket) {
	t.Result = findFibonacci(t.N)
}

func findFibonacci(N int) uint64 {
	var a0 uint64
	var a1 uint64
	a0 = 0
	a1 = 1
	for i := 0; i < N/2; i++ {
		a0 = a0 + a1
		a1 = a0 + a1
	}
	if N%2 == 0 {
		return a0
	} else {
		return a1
	}
}

func (s *Slave) runTask(t *packets.TaskPacket) {
	switch t.TaskTypeID {
	case packets.FibonacciTaskType:
		RunFibTask(t)
	default:
		s.Logger.Error(logger.FormatLogMessage("msg", "Invalid Task Type"))
	}
}
