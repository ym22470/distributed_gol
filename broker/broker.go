package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

var mutex sync.Mutex
var servers = []string{
	// "127.0.0.1:8040",
	"34.229.134.5:8030",
	// "127.0.0.1:8051",
	"52.91.39.210:8030",
	// "127.0.0.1:8052",
	// "127.0.0.1:8053",
}
var clients = make([]*rpc.Client, len(servers))

// Create a RPC service that contains various
type Broker struct {
	Turn      int
	CellCount int
	Resume    chan bool
	Pause     bool
	World     [][]byte
	Client    []string
}

func (s *Broker) ProcessWorld(req gol.Request, res *gol.Response) error {
	//fmt.Println("Into broker")
	s.Turn = 0
	turn := 0
	s.Resume = make(chan bool)
	mutex.Lock()
	s.World = copySlice(req.World)
	mutex.Unlock()
	// TODO: Execute all turns of the Game of Life.
	for ; turn < req.Parameter.Turns; turn++ {
		mutex.Lock()
		if s.Pause {
			mutex.Unlock()
			<-s.Resume
		} else {
			mutex.Unlock()
		}

		// Number of the servers
		numOfServers := 2
		//req.Parameter.Threads = 1
		if numOfServers == 1 {
			s.World = nextState(req.Parameter, s.World, 0, req.Parameter.ImageHeight)
		} else {
			//req.Parameter.Threads = 1
			result := make(chan [][]byte)
			//for i := 0; i < req.Parameter.Threads; i++ {
			// a := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
			// b := (i + 1) * (req.Parameter.ImageHeight / req.Parameter.Threads)
			// if i == req.Parameter.Threads-1 {
			// 	b = req.Parameter.ImageHeight
			// }
			//to handle data race condition by passing a copy of world to goroutines
			mutex.Lock()
			worldCopy := copySlice(s.World)
			mutex.Unlock()
			go workers(req.Parameter, worldCopy, result, 0, req.Parameter.ImageHeight)
			temp := <-result
			req.World = copySlice(temp)
			//}
		}
		mutex.Lock()
		s.World = copySlice(req.World)
		s.CellCount = len(calculateAliveCells(req.Parameter, s.World))
		s.Turn++
		mutex.Unlock()
	}
	//send the finished world and AliveCells to respond
	mutex.Lock()
	res.World = copySlice(s.World)
	res.AliveCells = calculateAliveCells(req.Parameter, res.World)
	res.CompletedTurns = turn
	mutex.Unlock()
	return nil
}

func (s *Broker) CountAliveCell(req gol.Request, res *gol.Response) error {
	mutex.Lock()
	res.Turns = s.Turn
	res.CellCount = s.CellCount
	mutex.Unlock()
	return nil
}

func (s *Broker) KeyGol(req gol.Request, res *gol.Response) error {
	if req.S {
		mutex.Lock()
		res.Turns = s.Turn
		res.World = s.World
		mutex.Unlock()
	} else if req.P {
		mutex.Lock()
		s.Pause = !s.Pause
		mutex.Unlock()
		if !s.Pause {
			s.Resume <- true
		}
	} else if req.K {
		var wg sync.WaitGroup
		// node quit
		// wg.Add(1)
		for i, server := range servers {
			wg.Add(1)
			client, err := rpc.Dial("tcp", server)
			if err != nil {
				log.Fatalf("Failed to connect to server %s: %v", server, err)
				defer client.Close()
			}
			clients[i] = client
			client.Call(gol.Shutdown, new(gol.Request), new(gol.Response))
			defer client.Close()
			wg.Done()
		}
		wg.Wait()
		os.Exit(0)
	}
	return nil
}

func nextState(p gol.Params, world [][]byte, start, end int) [][]byte {
	// allocate space
	nextWorld := make([][]byte, end-start)
	for i := range nextWorld {
		nextWorld[i] = make([]byte, p.ImageWidth)
	}

	directions := [8][2]int{
		{-1, -1}, {-1, 0}, {-1, 1},
		{0, -1}, {0, 1},
		{1, -1}, {1, 0}, {1, 1},
	}

	for row := start; row < end; row++ {
		for col := 0; col < p.ImageWidth; col++ {
			// the alive must be set to 0 everytime when it comes to a different position
			alive := 0
			for _, dir := range directions {
				// + imageHeight make sure the image is connected
				newRow, newCol := (row+dir[0]+p.ImageHeight)%p.ImageHeight, (col+dir[1]+p.ImageWidth)%p.ImageWidth
				if world[newRow][newCol] == 255 {
					alive++
				}
			}
			if world[row][col] == 255 {
				if alive < 2 || alive > 3 {
					nextWorld[row-start][col] = 0
				} else {
					nextWorld[row-start][col] = 255
				}
			} else if world[row][col] == 0 {
				if alive == 3 {
					nextWorld[row-start][col] = 255
				} else {
					nextWorld[row-start][col] = 0
				}
			}
		}
	}
	return nextWorld
}

func workers(p gol.Params, world [][]byte, result chan<- [][]byte, start, end int) {
	fmt.Println("go worker")
	worldPiece := copySlice(world)
	// server1 := "127.0.0.1:8040"
	// client1, _ := rpc.Dial("tcp", server1)
	// server2 := "127.0.0.1:8051"
	// client2, _ := rpc.Dial("tcp", server2)
	// defer client1.Close()
	// defer client2.Close()
	// // req := new(gol.Request)
	// // req.World = copySlice(world)
	// // req.Start = 0
	// // req.End = p.ImageHeight
	// // req.Parameter = p
	// // res := new(gol.Response)
	// // client1.Call(gol.ProcessGol, req, res)
	// // worldPiece = copySlice(res.World)
	// req := new(gol.Request)
	// req.World = copySlice(world)
	// req.Start = 0 * (p.ImageHeight / 2)
	// req.End = 1 * (p.ImageHeight / 2)
	// req.Parameter = p
	// res := new(gol.Response)
	// client1.Call(gol.ProcessGol, req, res)
	// for i := 0; i < req.End; i++ {
	// 	copy(worldPiece[i], res.World[i])
	// }
	// req2 := new(gol.Request)
	// req2.World = copySlice(world)
	// req2.Start = 1 * (p.ImageHeight / 2)
	// req2.End = 2 * (p.ImageHeight / 2)
	// req2.Parameter = p
	// res2 := new(gol.Response)
	// client2.Call(gol.ProcessGol, req2, res2)
	// for i := req2.Start; i < req2.End; i++ {
	// 	copy(worldPiece[i], res2.World[i-req2.Start])
	// }
	// servers := []string{
	// 	"127.0.0.1:8040",
	// 	"127.0.0.1:8051",
	// 	// "127.0.0.1:8052",
	// 	// "127.0.0.1:8053",
	// }
	//clients := make([]*rpc.Client, len(servers))
	for i, server := range servers {
		client, err := rpc.Dial("tcp", server)
		if err != nil {
			log.Fatalf("Failed to connect to server %s: %v", server, err)
		}
		defer client.Close()
		clients[i] = client
	}
	// Process the world in pieces
	for i, client := range clients {
		req := new(gol.Request)
		req.World = copySlice(world)
		req.Start = i * (p.ImageHeight / len(clients))
		req.End = (i + 1) * (p.ImageHeight / len(clients))
		if i == len(clients)-1 {
			req.End = p.ImageHeight
		}
		req.Parameter = p

		res := new(gol.Response)
		err := client.Call(gol.ProcessGol, req, res)
		defer client.Close()
		if err != nil {
			log.Fatalf("Failed to process world: %v", err)
		}

		for j := req.Start; j < req.End; j++ {
			copy(worldPiece[j], res.World[j-req.Start])
		}
	}
	//worldPiece := nextState(p, world, start, end)
	result <- worldPiece
	close(result)
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
	pAddr := flag.String("port", "8030", "port to listen on")
	flag.Parse()
	//initialise server
	server := &Broker{
		Resume:    make(chan bool),
		Turn:      0,
		CellCount: 0,
	}
	err := rpc.Register(server)
	if err != nil {
		return
	}
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {

		}
	}(listener)
	rpc.Accept(listener)
}
