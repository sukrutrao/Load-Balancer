package main

import (
	"os"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/master_src"
)

func main() {
	algo := "round_robin"
	if len(os.Args) >= 2 {
		algo = os.Args[1]
	}
	logger.SetLogLevel(logger.DEBUG)
	m := master.Master{
		Logger: logger.NewLogger("master"),
	}
	m.Run(algo)
}
