package main

import (
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/natsbroker"
	"github.com/ubiqueworks/go-clean-architecture/service/consumer/handler"
)

const Name = "consumer"

var (
	GitCommit string
	Version   string
)

func main() {
	service, err := framework.Create(Name, Version, GitCommit, handler.ServiceHandler())
	if err != nil {
		panic(err)
	}

	service.AddComponent(natsbroker.Create())
	service.Bootstrap()
}
