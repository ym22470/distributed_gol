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

// var wg sync.WaitGroup

type Broker struct {
	Clients    []*rpc.Client
	World      [][]byte
	AliveCells []util.Cell
	Turn       int
	Resume     chan bool
	Pause      bool
	CellCount  int
}

func (b *Broker) ProcessWorld(req gol.Request, res *gol.Response) error {
	// responses := make([][][]byte, len(b.Clients))
	// for i := 0; i < len(b.Clients); i++ {
	// 	responses[i] = make([][]byte, req.Parameter.ImageHeight)
	// 	for j := range responses[i] {
	// 		responses[i][j] = make([]byte, req.Parameter.ImageWidth)
	// 	}
	// }
	// res.World = make([][]byte, req.Parameter.ImageHeight)
	// for i := range res.World {
	// 	res.World[i] = make([]byte, req.Parameter.ImageWidth)
	// }
	res.World = make([][]byte, req.Parameter.ImageHeight)
	for i := range res.World {
		res.World[i] = make([]byte, req.Parameter.ImageWidth)
	}
	req.Parameter.Threads = len(b.Clients)
	b.Turn = 0
	//turn := 0
	b.Resume = make(chan bool)
	mutex.Lock()
	b.World = make([][]byte, req.Parameter.ImageHeight)
	for i := range b.World {
		b.World[i] = make([]byte, req.Parameter.ImageWidth)
	}
	//b.World = copySlice(req.World)
	mutex.Unlock()
	// TODO: Execute all turns of the Game of Life.
	//fmt.Println("the req turns is", req.Parameter.Turns)
	finish := make(chan bool, 1)
	if req.Parameter.Turns == 0 {
		//res.World = req.World
		b.World = copySlice(req.World)
	} else {
		for turn := 0; turn < req.Parameter.Turns; turn++ {
			// var wg sync.WaitGroup
			mutex.Lock()
			if b.Pause {
				mutex.Unlock()
				<-b.Resume
			} else {
				mutex.Unlock()
			}
			for i, client := range b.Clients {
				req.Start = i * (req.Parameter.ImageHeight / req.Parameter.Threads)
				req.End = (i + 1) * (req.Parameter.ImageHeight / req.Parameter.Threads)
				if i == req.Parameter.Threads-1 {
					req.End = req.Parameter.ImageHeight
				}
				client := client
				i := i
				worldCopy := copySlice(b.World)
				go func() {
					reqSlice := gol.Request{World: worldCopy, Start: req.Start, End: req.End}
					client.Call(gol.ProcessGol, reqSlice, res)
					//responses[i] = res.World
					start := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
					end := (i + 1) * (req.Parameter.ImageHeight / req.Parameter.Threads)
					if i == req.Parameter.Threads-1 {
						end = req.Parameter.ImageHeight
					}
					mutex.Lock()
					for i := start; i < end; i++ {
						copy(req.World[i], res.World[i-start])
					}
					mutex.Unlock()
					finish <- true
				}()
				mutex.Lock()
				b.World = copySlice(req.World)
				mutex.Unlock()
			}
			for i := 0; i < req.Parameter.Threads; i++ {
				<-finish
			}
			mutex.Lock()
			//b.World = copySlice(req.World)
			//req.World = copySlice(b.World)
			//fmt.Println(len(b.World))
			b.CellCount = len(calculateAliveCells(req.Parameter, b.World))
			b.Turn++
			mutex.Unlock()
		}
	}
	//send the finished world and AliveCells to respond
	mutex.Lock()
	res.World = copySlice(b.World)
	res.AliveCells = calculateAliveCells(req.Parameter, b.World)
	res.CompletedTurns = b.Turn
	mutex.Unlock()
	return nil
}

func (b *Broker) CountAliveCell(req gol.Request, res *gol.Response) error {
	mutex.Lock()
	res.Turns = b.Turn
	res.CellCount = b.CellCount
	mutex.Unlock()
	return nil
}

func (b *Broker) KeyGol(req gol.Request, res *gol.Response) error {
	if req.S {
		mutex.Lock()
		res.Turns = b.Turn
		res.World = copySlice(b.World)
		mutex.Unlock()
	} else if req.P {
		mutex.Lock()
		b.Pause = !b.Pause
		mutex.Unlock()
		if !b.Pause {
			b.Resume <- true
		}
	} else if req.K {
		os.Exit(0)
	}
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

func main() {
	addresses := []string{
		"127.0.0.1:8040",
		"127.0.0.1:8050",
		//"127.0.0.1:8060",
		//"127.0.0.1:8070",
	}
	clients := make([]*rpc.Client, 2)
	for n := 0; n < 2; n++ {
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
