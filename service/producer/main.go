package main

import (
	"github.com/ubiqueworks/go-clean-architecture/framework"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/cloudstore"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/natsbroker"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/transport/http"
	"github.com/ubiqueworks/go-clean-architecture/framework/component/transport/rpc"
	"github.com/ubiqueworks/go-clean-architecture/service/producer/handler"
)

const Name = "producer"

var (
	GitCommit string
	Version   string
)

func main() {
	service, err := framework.Create(Name, Version, GitCommit, handler.ServiceHandler())
	if err != nil {
		panic(err)
	}

	service.AddComponent(cloudstore.Create())
	service.AddComponent(natsbroker.Create())
	service.AddComponent(microhttp.Create(handler.InitHttpFunc), framework.HandlerComponent)
	service.AddComponent(microrpc.Create(handler.InitRpcFunc), framework.HandlerComponent)
	service.Bootstrap()
}
