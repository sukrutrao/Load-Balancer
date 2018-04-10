package main

import (
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/slave_src"
)

func main() {
	logger.SetLogLevel(logger.DEBUG)
	s := slave.Slave{
		Logger: logger.NewLogger("master"),
	}
	s.Run()
}
