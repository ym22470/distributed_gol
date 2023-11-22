package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

var mutex sync.Mutex

// Create a RPC service that contains various
type Server struct {
	Turn      int
	CellCount int
	Resume    chan bool
	Pause     bool
	World     [][]byte
	Slice     [][]byte
}

func (s *Server) ProcessWorld(req gol.Request, res *gol.Response) error {
	//fmt.Println(req.Parameter.Turns)
	s.Turn = 0
	turn := 0
	s.Resume = make(chan bool)
	mutex.Lock()
	s.World = copySlice(req.World)
	mutex.Unlock()
	//fmt.Println("turn completed")
	//fmt.Println(req.Parameter.Turns)
	// TODO: Execute all turns of the Game of Life.
	fmt.Println(len(req.World[0]) + 3)
	if req.Parameter.Turns > 0 {
		for ; turn < req.Parameter.Turns; turn++ {
			mutex.Lock()
			if s.Pause {
				mutex.Unlock()
				<-s.Resume
			} else {
				mutex.Unlock()
			}
			chans := make(chan [][]byte)
			mutex.Lock()
			//fmt.Println(len(req.World[0]) + 2)
			worldCopy := copySlice(s.World)
			//fmt.Println(len(worldCopy[0]) + 1)
			mutex.Unlock()
			go workers(req.Parameter, worldCopy, chans, req.Start, req.End)
			//fmt.Println("turn completed")
			strip := <-chans
			//fmt.Println("turn completed")
			mutex.Lock()
			//fmt.Println(len(strip[0]))
			s.Slice = copySlice(strip)
			mutex.Unlock()
			//fmt.Println("turn completed")
			//count the number of cells and turns
			mutex.Lock()
			s.CellCount = len(calculateAliveCells(req.Parameter, s.Slice))
			s.Turn++
			mutex.Unlock()
			//fmt.Println("turn completed")
		}
	} else {
		s.Slice = req.World[req.Start:req.End]
		mutex.Lock()
		s.CellCount = len(calculateAliveCells(req.Parameter, s.Slice))
		mutex.Unlock()
	}
	//case when the resquest turn is 0

	fmt.Println(len(req.World))
	fmt.Println(len(req.World[0]) + 4)
	//send the finished world and AliveCells to respond
	mutex.Lock()
	fmt.Println(len(s.Slice))
	res.Slice = s.Slice
	//datarace here, need mutex lock
	res.AliveCells = calculateAliveCells(req.Parameter, s.Slice)
	fmt.Println(len(res.AliveCells))
	res.CompletedTurns = turn
	mutex.Unlock()
	return nil
}

func (s *Server) CountAliveCell(req gol.Request, res *gol.Response) error {
	mutex.Lock()
	res.Turns = s.Turn
	res.CellCount = s.CellCount
	mutex.Unlock()
	return nil
}

func (s *Server) KeyGol(req gol.Request, res *gol.Response) error {
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
	worldPiece := nextState(p, world, start, end)
	result <- worldPiece
	fmt.Println("state updated")
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
	for row := 0; row < len(world); row++ {
		for col := 0; col < len(world[row]); col++ {
			//mutex.Lock()
			if world[row][col] == 255 {
				//mutex.Unlock()
				aliveCell = append(aliveCell, util.Cell{X: col, Y: row})
			} else {
				//mutex.Unlock()
			}
		}
	}
	return aliveCell
}

func main() {
	pAddr := flag.String("port", "8050", "port to listen on")
	flag.Parse()
	//initialise server
	server := &Server{
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
