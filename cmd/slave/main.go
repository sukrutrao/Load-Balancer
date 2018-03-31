package main

import (
	"github.com/GoodDeeds/load-balancer/slave_src"
)

func main() {
	s := slave.Slave{}
	s.Run()
}