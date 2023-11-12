package gol

import (
	"sync"
	"uk.ac.bris.cs/gameoflife/gol/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// Create a RPC service that contains variousm
type Server struct{}

func (s *Server) ProcessWorld(req stubs.Request, res *stubs.Response) {
	turnCount := 0
	turn := 0
	//var cellCount int
	var mutex sync.Mutex
	for ; turn < req.Parameter.Turns; turn++ {
		//if pasued {
		//	<-resume
		//		}
		if req.Parameter.Threads == 1 {
			req.Message = nextState(req.Parameter, req.Message, 0, req.Parameter.ImageHeight)
		} else {
			newSize := p.ImageHeight / p.Threads
			result := make([]chan [][]byte, p.Threads)

			for i := range result {
				result[i] = make(chan [][]byte)
			}

			for i := 0; i < p.Threads; i++ {
				start := i * newSize
				end := start + newSize
				if i == p.Threads-1 {
					end = p.ImageHeight
				}
				worldCopy := copySlice(world)
				go workers(p, worldCopy, result[i], start, end)
			}

			for i := 0; i < p.Threads; i++ {
				result := <-result[i]
				start := i * newSize
				// This ensures that we are copying the worker result to the correct place in the world.
				for j := start; j < start+len(result); j++ {
					for k := 0; k < p.ImageWidth; k++ {
						if result[j-start][k] != world[j][k] {
							c.events <- CellFlipped{turn + 1, util.Cell{X: k, Y: j}}
						}
					}
					copy(world[j], result[j-start])
				}
			}
			mutex.Lock()
			turnCount++
			cellCount = len(calculateAliveCells(p, world))
			mutex.Unlock()
		}
		c.events <- TurnComplete{turn}
	}

}
func nextState(p Params, world [][]byte, start, end int) [][]byte {
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

func workers(p Params, world [][]byte, result chan<- [][]byte, start, end int) {
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
