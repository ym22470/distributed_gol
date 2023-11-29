package main

import (
	"flag"
	"net"
	"net/rpc"
	"os"
	"sync"
	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

var mutex sync.Mutex
var wg sync.WaitGroup

type Broker struct {
	Pause              bool
	Resume             chan bool
	Turn               int
	CellCount          int
	Nodes              int
	Clients            []*rpc.Client
	CombinedWorld      [][]byte
	CombinedAliveCells []util.Cell
}

func (b *Broker) GolInitializer(req gol.Request, res *gol.Response) error {
	//reset global variables in the struct at the beginning of each call
	b.Resume = make(chan bool)
	b.CombinedWorld = [][]byte{}
	turn := 0
	if req.Parameter.Turns == 0 {
		mutex.Lock()
		if b.Pause {
			mutex.Unlock()
			<-b.Resume
		} else {
			mutex.Unlock()
		}
		b.CombinedWorld = copySlice(req.World)
	} else {
		for ; turn < req.Parameter.Turns; turn++ {
			mutex.Lock()
			if b.Pause {
				mutex.Unlock()
				<-b.Resume
			} else {
				mutex.Unlock()
			}
			responses := make([][][]byte, len(b.Clients))
			// Initialize each slice in responses to prevent index out of range error
			for i := range responses {
				responses[i] = make([][]byte, req.Parameter.ImageHeight/b.Nodes)
			}
			for i, client := range b.Clients {
				// Calculate start and end for this segment
				start := i * (req.Parameter.ImageHeight / b.Nodes)
				end := (i + 1) * (req.Parameter.ImageHeight / b.Nodes)
				if i == b.Nodes-1 {
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

					client.Call(gol.ProcessGol, reqCopy, localRes)

					mutex.Lock()
					responses[i] = localRes.Slice
					//b.CombinedAliveCells = append(b.CombinedAliveCells, localRes.AliveCells...)
					mutex.Unlock()
				}(client, reqCopy, i)
			}
			// Wait for all goroutines to complete
			wg.Wait()
			// Now that all goroutines have completed, you can proceed
			// Assemble all the strips together, replace the initial world with the completed world
			b.CombinedWorld = req.World
			for i := 0; i < b.Nodes; i++ {
				strip := responses[i]
				startRow := i * (req.Parameter.ImageHeight / b.Nodes)
				for r, row := range strip {
					mutex.Lock()
					b.CombinedWorld[startRow+r] = row
					mutex.Unlock()
				}
			}
			//update the current state of the turn
			mutex.Lock()
			b.Turn++
			b.CellCount = len(calculateAliveCells(req.Parameter, b.CombinedWorld))
			mutex.Unlock()
		}
	}
	//update the finishing state
	res.World = copySlice(b.CombinedWorld)
	res.AliveCells = calculateAliveCells(req.Parameter, b.CombinedWorld)
	res.End = true
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
	mutex.Lock()
	res.Turns = b.Turn
	res.CellCount = b.CellCount
	mutex.Unlock()
	return nil
}

func (b *Broker) GetLive(req gol.Request, res *gol.Response) error {
	mutex.Lock()
	res.World = b.CombinedWorld
	res.Turns = b.Turn
	mutex.Unlock()
	return nil
}

func (b *Broker) GolKey(req gol.Request, res *gol.Response) error {
	if req.S {
		mutex.Lock()
		res.Turns = b.Turn
		res.World = copySlice(b.CombinedWorld)
		mutex.Unlock()
	} else if req.P {
		mutex.Lock()
		b.Pause = !b.Pause
		mutex.Unlock()
		if !b.Pause {
			b.Resume <- true
		}
	} else if req.K {
		for _, client := range b.Clients {
			wg.Add(1)
			go func(client *rpc.Client) {
				defer wg.Done()
				client.Call(gol.Key, req, res)
			}(client)
		}
		//wait until all the goroutines finish( all the nodes to quit)
		wg.Wait()
		os.Exit(0)
	}
	return nil
}

func main() {
	addresses := []string{
		//"127.0.0.1:8040",
		//"127.0.0.1:8050",
		//"127.0.0.1:8060",
		//"127.0.0.1:8070",
		"54.209.119.86:8030",
		"100.25.188.63:8030",
		"3.86.165.22:8030",
		"3.80.204.84:8030",
	}
	clients := make([]*rpc.Client, 4)
	for n := 0; n < 4; n++ {
		clients[n], _ = rpc.Dial("tcp", addresses[n])
	}
	broker := &Broker{
		Clients:   clients,
		Resume:    make(chan bool),
		Turn:      0,
		CellCount: 0,
		Nodes:     4,
	}
	err := rpc.Register(broker)
	if err != nil {
		return
	}
	pAddr := flag.String("port", "8080", "port to listen on")
	//create a listener to listen to the distributor on the port
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
