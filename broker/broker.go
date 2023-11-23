package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

var mutex sync.Mutex
var wg sync.WaitGroup

type Broker struct {
	Clients            []*rpc.Client
	CombinedWorld      [][]byte
	CombinedAliveCells []util.Cell
}

func (b *Broker) GolInitializer(req gol.Request, res *gol.Response) error {
	//reset after each call
	b.CombinedWorld = [][]byte{}
	if req.Parameter.Turns == 0 {
		b.CombinedWorld = copySlice(req.World)
		//b.CombinedAliveCells
	} else {
		//b.CombinedAliveCells = []util.Cell{}
		responses := make([][][]byte, len(b.Clients))
		// Initialize each slice in responses to prevent index out of range error
		for i := range responses {
			responses[i] = make([][]byte, req.Parameter.ImageHeight/req.Parameter.Threads)
		}
		for i, client := range b.Clients {
			// Calculate start and end for this segment
			start := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
			end := (i + 1) * (req.Parameter.ImageHeight / req.Parameter.Threads)
			if i == req.Parameter.Threads-1 {
				end = req.Parameter.ImageHeight
			}
			// Increment the WaitGroup counter
			wg.Add(1)
			// Create a copy of req for each goroutine
			reqCopy := req
			reqCopy.Start = start
			reqCopy.End = end
			// Launch the goroutine with its own copy of req
			go func(client *rpc.Client, reqCopy gol.Request, i int) {
				defer wg.Done()

				localRes := &gol.Response{} // Create a local response object

				fmt.Println("Goroutine for start:", reqCopy.Start, "end:", reqCopy.End)
				client.Call(gol.ProcessGol, reqCopy, localRes)
				fmt.Println("RPC call completed for start:", reqCopy.Start, "end:", reqCopy.End)

				mutex.Lock()
				responses[i] = localRes.Slice
				//b.CombinedAliveCells = append(b.CombinedAliveCells, localRes.AliveCells...)
				mutex.Unlock()
			}(client, reqCopy, i)
		}

		// Wait for all goroutines to complete
		wg.Wait()
		// Now that all goroutines have completed, you can proceed
		// Assemble all the strips together
		b.CombinedWorld = req.World
		for i := 0; i < req.Parameter.Threads; i++ {
			fmt.Println("inside loop")
			strip := responses[i]
			startRow := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
			for r, row := range strip {
				mutex.Lock()
				req.World[startRow+r] = row
				mutex.Unlock()
			}
		}
	}
	res.World = copySlice(b.CombinedWorld)
	res.AliveCells = calculateAliveCells(req.Parameter, b.CombinedWorld)
	//fmt.Println(len(res.World))
	//fmt.Println(len(res.World[0]))
	return nil
}

func copySlice(src [][]byte) [][]byte {
	dst := make([][]byte, len(src))
	for i := range src {
		dst[i] = make([]byte, len(src[i]))
		copy(dst[i], src[i])
	}
	return dst
}
func calculateAliveCells(p gol.Params, world [][]byte) []util.Cell {
	var aliveCell []util.Cell
	for row := 0; row < p.ImageHeight; row++ {
		for col := 0; col < p.ImageWidth; col++ {
			if world[row][col] == 255 {
				aliveCell = append(aliveCell, util.Cell{X: col, Y: row})
			}
		}
	}
	return aliveCell
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
