package main

import (
	"flag"
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
	"127.0.0.1:8040",
	// "34.201.147.188:8030",
	"127.0.0.1:8051",
	// "54.242.29.84:8030",
	"127.0.0.1:8052",
	"127.0.0.1:8053",
}
var backupServers = []string{
	"127.0.0.1:8054",
	"127.0.0.1:8055",
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
		if numOfServers == 1 {
			s.World = nextState(req.Parameter, s.World, 0, req.Parameter.ImageHeight)
		} else {
			result := make(chan [][]byte)
			//to handle data race condition by passing a copy of world to goroutines
			mutex.Lock()
			worldCopy := copySlice(s.World)
			mutex.Unlock()
			go workers(req.Parameter, worldCopy, result, 0, req.Parameter.ImageHeight)
			temp := <-result
			req.World = copySlice(temp)

		}
		mutex.Lock()
		s.World = copySlice(req.World)
		s.CellCount = len(calculateAliveCells(req.Parameter, s.World))
		s.Turn++
		mutex.Unlock()
	}
	// Send the finished world and AliveCells to respond
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

func (s *Broker) GetLive(req gol.Request, res *gol.Response) error {
	mutex.Lock()
	res.World = copySlice(s.World)
	res.Turns = s.Turn
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
		for i, server := range servers {
			wg.Add(1)
			client, err := rpc.Dial("tcp", server)
			if err != nil {
				log.Fatalf("Failed to connect to server %s: %v", server, err)
				defer client.Close()
			}
			clients[i] = client
			go func() {
				client.Call(gol.Shutdown, new(gol.Request), new(gol.Response))
				defer client.Close()
				wg.Done()
			}()
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
	worldPiece := copySlice(world)
	var wg sync.WaitGroup
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
		req.Parameter = p
		haloSize := p.ImageHeight / len(clients)
		req.Start = i * haloSize
		req.End = (i + 1) * haloSize
		if i == len(clients)-1 {
			req.End = p.ImageHeight
		}

		// Process the world with halo rows
		req.World = make([][]byte, haloSize+2)
		for k := range req.World {
			req.World[k] = make([]byte, req.Parameter.ImageWidth)
		}
		for j := 0; j < len(req.World)-2; j++ {
			copy(req.World[j+1], world[i*haloSize+j])
		}

		// Halo above
		if i == 0 {
			copy(req.World[0], world[req.Parameter.ImageHeight-1])
		} else {
			copy(req.World[0], world[i*haloSize-1])
		}

		// Halo below
		if i == len(clients)-1 {
			copy(req.World[len(req.World)-1], world[0])
		} else {
			copy(req.World[len(req.World)-1], world[(i+1)*haloSize])
		}
		req.Parameter = p

		wg.Add(1)
		res := new(gol.Response)
		go func(req *gol.Request, res *gol.Response) {
			err := client.Call(gol.ProcessGol, req, res)
			defer client.Close()
			if err != nil {
				log.Fatalf("Failed to process world: %v", err)
			}
			wg.Done()
		}(req, res)

		wg.Wait()
		// Get rid of the halo rows
		for j := req.Start; j < req.End; j++ {
			copy(worldPiece[j], res.World[j-req.Start])
		}
	}
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
