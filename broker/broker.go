package main

import (
	"flag"
	"net"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/gol"
)

type Broker struct {
	//servers []string
}

func (b *Broker) GolInitializer(req gol.Request, res *gol.Response) error {
	server := "127.0.0.1:8020"
	client, _ := rpc.Dial("tcp", server)
	client.Call(gol.ProcessGol, req, res)
	return nil
}

func main() {
	broker := &Broker{
		//servers: []string{},
	}
	err := rpc.Register(broker)
	if err != nil {
		return
	}
	pAddr := flag.String("port", "8030", "port to listen on")
	//create a listener to listen to the distributor on the port
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
