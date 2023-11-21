package main

import (
	"flag"
	"net"
	"net/rpc"
	"os"
	"uk.ac.bris.cs/gameoflife/gol"
)

type Broker struct {
	Servers []string
	Client1 *rpc.Client
	Clients []*rpc.Client
}

func (b *Broker) GolInitializer(req gol.Request, res *gol.Response) error {
	for i, client := range b.Clients {
		funcName := gol.ProcessGol[i]
		client.Call(funcName, req, res)
	}
	//for multiple workers call 4 times on 4 AWS nodes and receive the result once it's finished
	return nil
}

func (b *Broker) GolAliveCells(req gol.Request, res *gol.Response) error {
	err := b.Client1.Call(gol.AliveCells, req, res)
	if err != nil {
		return err
	}
	return nil
}

func (b *Broker) GolKey(req gol.Request, res *gol.Response) error {
	err := b.Client1.Call(gol.Key, req, res)
	if req.K {
		os.Exit(0)
	}
	if err != nil {
		return err
	}
	return nil
}

func main() {
	addresses := []string{
		"127.0.0.1:8020",
		"127.0.0.1:8020",
		"127.0.0.1:8020",
		"127.0.0.1:8020",
	}
	clients := make([]*rpc.Client, 4)
	for n := 0; n < 4; n++ {
		clients[n], _ = rpc.Dial("tcp", addresses[n])
	}
	broker := &Broker{
		Servers: []string{"ProcessNode1", "ProcessNode2", "ProcessNode3", "ProcessNode4"},
		Clients: clients,
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
