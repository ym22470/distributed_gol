package main

import (
	"flag"
	"net"
	"net/rpc"
	"os"
	"uk.ac.bris.cs/gameoflife/gol"
)

type Broker struct {
	Server string
	Client *rpc.Client
}

func (b *Broker) GolInitializer(req gol.Request, res *gol.Response) error {
	err := b.Client.Call(gol.ProcessGol, req, res)
	if err != nil {
		return err
	}
	//for multiple workers call 4 times on 4 AWS nodes and receive the result once it's finished
	return nil
}

func (b *Broker) GolAliveCells(req gol.Request, res *gol.Response) error {
	err := b.Client.Call(gol.AliveCells, req, res)
	if err != nil {
		return err
	}
	return nil
}

func (b *Broker) GolKey(req gol.Request, res *gol.Response) error {
	err := b.Client.Call(gol.Key, req, res)
	if req.K {
		os.Exit(0)
	}
	if err != nil {
		return err
	}
	return nil
}

func main() {
	address := "127.0.0.1:8020"
	client, _ := rpc.Dial("tcp", address)
	broker := &Broker{
		Server: address,
		Client: client,
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
