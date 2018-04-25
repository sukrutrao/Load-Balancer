package main

import (
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/monitoring_src"
)

func main() {
	logger.SetLogLevel(logger.DEBUG)
	m := monitoring.Monitor{
		Logger: logger.NewLogger("monitoring"),
	}
	m.Run()
}
