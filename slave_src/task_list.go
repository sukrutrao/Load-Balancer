package slave

import (
	// "math"
	//	"fmt"

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
	case packets.CountPrimesTaskType:
		CountPrimesTask(t)
	default:
		s.Logger.Error(logger.FormatLogMessage("msg", "Invalid Task Type"))
	}
}

func CountPrimesTask(t *packets.TaskPacket) {
	t.IntResult = countPrimes(t.N)
}

func countPrimes(N int) int {
	if N <= 1 {
		return 0
	}
	count := 0
	for i := 2; i <= N; i++ {
		isPrime := true
		for j := 2; j < i; j++ {
			if i%j == 0 {
				isPrime = false
				break
			}
		}
		if isPrime {
			count++
		}
	}
	//	fmt.Println(count)
	return count
}
