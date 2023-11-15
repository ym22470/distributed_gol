package main

import (
	"flag"
	"net"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

// Create a RPC service that contains various
type Server struct{}

func (s *Server) ProcessWorld(req gol.Request, res *gol.Response) error {

	// var mutex sync.Mutex

	turn := 0
	// TODO: Execute all turns of the Game of Life.
	for ; turn < req.Parameter.Turns; turn++ {
		if req.Parameter.Threads == 1 {
			req.World = nextState(req.Parameter, req.World, 0, req.Parameter.ImageHeight)
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
				worldCopy := copySlice(req.World)
				go workers(req.Parameter, worldCopy, chans[i], a, b)

			}
			//combine all the strips produced by workers
			for i := 0; i < req.Parameter.Threads; i++ {
				strip := <-chans[i]
				startRow := i * (req.Parameter.ImageHeight / req.Parameter.Threads)
				for r, row := range strip {
					req.World[startRow+r] = row
				}
			}
		}
		res.CellCount = len(calculateAliveCells(req.Parameter, req.World))
		res.TurnCount = turn
	}
	//send the finished world and AliveCells to respond
	res.World = req.World
	res.AliveCells = calculateAliveCells(req.Parameter, res.World)
	res.CompletedTurns = turn
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
	pAddr := flag.String("port", "8030", "port to listen on")
	flag.Parse()
	err := rpc.Register(&Server{})
	if err != nil {
		return
	}
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
