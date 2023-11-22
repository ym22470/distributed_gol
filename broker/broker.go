package main

import (
	"flag"
	"net"
	"net/rpc"
	"os"
	"sync"
	"uk.ac.bris.cs/gameoflife/gol"
)

var mutex sync.Mutex

type Broker struct {
	//Servers []string
	Client1       *rpc.Client
	Clients       []*rpc.Client
	CombinedWorld [][]byte
}

func (b *Broker) GolInitializer(req gol.Request, res *gol.Response) error {
	responses := [][][]byte{}
	//counter counts the number of calls finished
	counter := 0
	for i, client := range b.Clients {
		req.Start = i * (req.Parameter.ImageHeight / req.Parameter.Threads)
		req.End = (i + 1) * (req.Parameter.ImageHeight / req.Parameter.Threads)
		//call the AWS nodes concurrently
		go func() {
			client.Call(gol.ProcessGol, req, res)
			responses[i] = res.Slice
			counter++
		}()
	}
	if counter == len(b.Clients) {
		for i := 0; i < req.Parameter.Threads; i++ {
			strip := responses[i]
			startRow := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
			for r, row := range strip {
				mutex.Lock()
				res.World[startRow+r] = row
				mutex.Unlock()
			}
		}
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
		"127.0.0.1:8040",
		"127.0.0.1:8050",
		"127.0.0.1:8060",
		"127.0.0.1:8070",
	}
	clients := make([]*rpc.Client, 4)
	for n := 0; n < 4; n++ {
		clients[n], _ = rpc.Dial("tcp", addresses[n])
	}
	broker := &Broker{
		//Servers: []string{"ProcessNode1", "ProcessNode2", "ProcessNode3", "ProcessNode4"},
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
