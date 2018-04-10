package main

import (
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/master_src"
)

func main() {
	logger.SetLogLevel(logger.DEBUG)
	m := master.Master{
		Logger: logger.NewLogger("master"),
	}
	m.Run()
}
