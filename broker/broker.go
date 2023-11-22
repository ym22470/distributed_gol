package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/gol"
)

var mutex sync.Mutex
var wg sync.WaitGroup

type Broker struct {
	//Servers []string
	//Client1       *rpc.Client
	Clients       []*rpc.Client
	CombinedWorld [][]byte
}

func (b *Broker) GolInitializer(req gol.Request, res *gol.Response) error {
	responses := make([][][]byte, len(b.Clients))
	// Initialize each slice in responses to prevent index out of range error
	for i := range responses {
		responses[i] = make([][]byte, req.Parameter.ImageHeight/req.Parameter.Threads)
	}
	for i, client := range b.Clients {
		req.Start = i * (req.Parameter.ImageHeight / req.Parameter.Threads)
		req.End = (i + 1) * (req.Parameter.ImageHeight / req.Parameter.Threads)
		// Increment the WaitGroup counter
		wg.Add(1)
		// Launch a goroutine to make calls to the servers
		go func() {
			defer wg.Done() // Decrement the counter when the goroutine completes
			client.Call(gol.ProcessGol, req, res)
			responses[i] = res.Slice
		}()
	}
	// Wait for all goroutines to complete
	wg.Wait()
	// Now that all goroutines have completed, you can proceed
	// Assemble all the strips together
	for i := 0; i < req.Parameter.Threads; i++ {
		strip := responses[i]
		startRow := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
		for r, row := range strip {
			mutex.Lock()
			res.World[startRow+r] = row
			mutex.Unlock()
		}
	}
	fmt.Println(len(res.World))
	return nil
}

func (b *Broker) GolAliveCells(req gol.Request, res *gol.Response) error {
	//err := b.Client1.Call(gol.AliveCells, req, res)
	//if err != nil {
	//	return err
	//}
	return nil
}

func (b *Broker) GolKey(req gol.Request, res *gol.Response) error {
	//err := b.Client1.Call(gol.Key, req, res)
	//if req.K {
	//	os.Exit(0)
	//}
	//if err != nil {
	//	return err
	//}
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