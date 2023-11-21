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

// Create a RPC service that contains various
type Server struct {
	Turn      int
	CellCount int
	Resume    chan bool
	Pause     bool
	World     [][]byte
}

func (s *Server) ProcessWorld(req gol.Request, res *gol.Response) error {
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
		if req.Parameter.Threads == 1 {
			s.World = nextState(req.Parameter, s.World, 0, req.Parameter.ImageHeight)
		} else {
			chans := make([]chan [][]byte, req.Parameter.Threads)
			for i := 0; i < req.Parameter.Threads; i++ {
				chans[i] = make(chan [][]byte)
				a := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
				b := (i + 1) * (req.Parameter.ImageHeight / req.Parameter.Threads)
				if i == req.Parameter.Threads-1 {
					b = req.Parameter.ImageHeight
				}
				//to handle data race condition by passing a copy of world to goroutines
				mutex.Lock()
				worldCopy := copySlice(s.World)
				mutex.Unlock()
				go workers(req.Parameter, worldCopy, chans[i], a, b)

			}
			//combine all the strips produced by workers
			for i := 0; i < req.Parameter.Threads; i++ {
				strip := <-chans[i]
				startRow := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
				for r, row := range strip {
					mutex.Lock()
					req.World[startRow+r] = row
					mutex.Unlock()
				}
			}
			mutex.Lock()
			s.World = copySlice(req.World)
			mutex.Unlock()
		}
		//count the number of cells and turns
		mutex.Lock()
		s.CellCount = len(calculateAliveCells(req.Parameter, s.World))
		s.Turn++
		mutex.Unlock()
	}
	//send the finished world and AliveCells to respond
	mutex.Lock()
	res.World = s.World
	res.AliveCells = calculateAliveCells(req.Parameter, s.World)
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
	pAddr := flag.String("port", "8020", "port to listen on")
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
	defer listener.Close()
	rpc.Accept(listener)
}
