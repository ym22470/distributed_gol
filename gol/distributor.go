package gol

import (
	"fmt"
	"net/rpc"
	"os"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	key        <-chan rune
}

func makeCall(client *rpc.Client, world [][]byte, p Params, c distributorChannels) {
	request := Request{World: world, Parameter: p}
	response := new(Response)
	// where is this ProcessGol called? in server?
	client.Call(ProcessGol, request, response)
	//send the content of world and receive on the other side(writePgm) concurrently
	c.ioCommand <- ioOutput
	if p.Turns == 0 {
		c.ioFilename <- fmt.Sprintf("%dx%dx0", p.ImageHeight, p.ImageWidth)
	} else if p.Threads == 1 {
		c.ioFilename <- fmt.Sprintf("%dx%dx%d", p.ImageHeight, p.ImageWidth, p.Turns)
	} else {
		c.ioFilename <- fmt.Sprintf("%dx%dx%d-%d", p.ImageHeight, p.ImageWidth, p.Turns, p.Threads)
	}
	//send the completed world to ioOutput c
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			c.ioOutput <- response.World[i][j]
		}
	}
	//report the final state of the world
	c.events <- FinalTurnComplete{CompletedTurns: response.CompletedTurns, Alive: response.AliveCells}
	// Make sure that the Io has finished any output before exiting.

	c.ioCommand <- ioCheckIdle

	<-c.ioIdle
	c.events <- StateChange{response.CompletedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
func distributor(p Params, c distributorChannels) {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%vx%v", p.ImageHeight, p.ImageWidth)
	server := "127.0.0.1:8030"
	//server := "54.234.177.143:8030"
	//create a client that dials to the tcp port
	client, err := rpc.Dial("tcp", server)
	if err != nil {
		fmt.Println("fialed to dial to server")
		os.Exit(1)

	}

	//close dial when everything is excuted
	defer client.Close()

	//create an empty world slice
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	// Initialize the state
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			world[y][x] = <-c.ioInput
			//if world[y][x] == 255 {
			//	//flip all the alive cell at initial state
			//	//c.events <- CellFlipped{0, util.Cell{X: x, Y: y}}
			//}
		}
	}
	makeCall(client, world, p, c)
}

//func workers(p Params, world [][]byte, result chan<- [][]byte, start, end int) {
//	worldPiece := nextState(p, world, start, end)
//	result <- worldPiece
//	close(result)
//}
//
//func copySlice(src [][]byte) [][]byte {
//	dst := make([][]byte, len(src))
//	for i := range src {
//		dst[i] = make([]byte, len(src[i]))
//		copy(dst[i], src[i])
//	}
//	return dst
//}
//
//// distributor divides the work between workers and interacts with other goroutines.
//func distributor(p Params, c distributorChannels) {
//	server := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
//	flag.Parse()
//	client, _ := rpc.Dial("tcp", *server)
//	defer client.Close()
//
//	// The ioInput is just a const for operation
//	// It determines the operation to do
//	c.ioCommand <- ioInput
//	c.ioFilename <- fmt.Sprintf("%vx%v", p.ImageHeight, p.ImageWidth)
//
//	// TODO: Create a 2D slice to store the world.
//	world := make([][]byte, p.ImageHeight)
//	for i := range world {
//		world[i] = make([]byte, p.ImageWidth)
//	}
//
//	// Initialize the state
//	for y := 0; y < p.ImageHeight; y++ {
//		for x := 0; x < p.ImageWidth; x++ {
//			world[y][x] = <-c.ioInput
//			if world[y][x] == 255 {
//				c.events <- CellFlipped{0, util.Cell{X: x, Y: y}}
//			}
//		}
//	}
//
//	//make a call to send the initialized world to the server
//	makeCall(client, world)
//
//	turnCount := 0
//	turn := 0
//	var cellCount int
//	var mutex sync.Mutex
//	ticker := time.NewTicker(2 * time.Second)
//	defer ticker.Stop()
//	go func() {
//		for range ticker.C {
//			mutex.Lock()
//			c.events <- AliveCellsCount{turnCount, cellCount}
//			mutex.Unlock()
//		}
//	}()
//
//	pasued := false
//	resume := make(chan bool)
//	quit := make(chan bool)
//	go func() {
//		for {
//			select {
//			case key := <-c.key:
//				switch key {
//				case 's':
//					c.ioCommand <- ioOutput
//					c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth, turnCount)
//					for y := 0; y < p.ImageHeight; y++ {
//						for x := 0; x < p.ImageWidth; x++ {
//							c.ioOutput <- world[y][x]
//						}
//					}
//					//fmt.Println("here in s")
//				case 'q':
//					c.ioCommand <- ioOutput
//					c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth, turnCount)
//					for y := 0; y < p.ImageHeight; y++ {
//						for x := 0; x < p.ImageWidth; x++ {
//							c.ioOutput <- world[y][x]
//						}
//					}
//					c.events <- FinalTurnComplete{turn, calculateAliveCells(p, world)}
//					c.ioCommand <- ioCheckIdle
//					<-c.ioIdle
//					c.events <- StateChange{turn, Quitting}
//					quit <- true
//				case 'p':
//					pasued = !pasued
//					if pasued {
//						c.events <- StateChange{turn, Paused}
//					} else {
//						fmt.Println("Continuing")
//						c.events <- StateChange{turn, Executing}
//						resume <- true
//					}
//				}
//			case <-quit:
//				return
//			}
//		}
//	}()
//	// TODO: Execute all turns of the Game of Life.
//	turnCount := 0
//	turn := 0
//	var cellCount int
//	var mutex sync.Mutex
//	for ; turn < p.Turns; turn++ {
//		if pasued {
//			<-resume
//		}
//		if p.Threads == 1 {
//			world = nextState(p, world, 0, p.ImageHeight)
//		} else {
//			newSize := p.ImageHeight / p.Threads
//			result := make([]chan [][]byte, p.Threads)
//
//			for i := range result {
//				result[i] = make(chan [][]byte)
//			}
//
//			for i := 0; i < p.Threads; i++ {
//				start := i * newSize
//				end := start + newSize
//				if i == p.Threads-1 {
//					end = p.ImageHeight
//				}
//				worldCopy := copySlice(world)
//				go workers(p, worldCopy, result[i], start, end)
//			}
//
//			for i := 0; i < p.Threads; i++ {
//				result := <-result[i]
//				start := i * newSize
//				// This ensures that we are copying the worker result to the correct place in the world.
//				for j := start; j < start+len(result); j++ {
//					for k := 0; k < p.ImageWidth; k++ {
//						if result[j-start][k] != world[j][k] {
//							c.events <- CellFlipped{turn + 1, util.Cell{X: k, Y: j}}
//						}
//					}
//					copy(world[j], result[j-start])
//				}
//			}
//			mutex.Lock()
//			turnCount++
//			cellCount = len(calculateAliveCells(p, world))
//			mutex.Unlock()
//		}
//		c.events <- TurnComplete{turn}
//	}
//
//	// TODO: Report the final state using FinalTurnCompleteEvent.
//	c.events <- FinalTurnComplete{turn, calculateAliveCells(p, world)}
//
//	// Output
//	c.ioCommand <- ioOutput
//	c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth, p.Turns)
//	for y := 0; y < p.ImageHeight; y++ {
//		for x := 0; x < p.ImageWidth; x++ {
//			c.ioOutput <- world[y][x]
//		}
//	}
//
//	// Make sure that the Io has finished any output before exiting.
//	c.ioCommand <- ioCheckIdle
//	<-c.ioIdle
//
//	c.events <- StateChange{turn, Quitting}
//	quit <- true
//
//	close(quit)
//	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
//	close(c.events)
//}
//
//// Gol next state
//func nextState(p Params, world [][]byte, start, end int) [][]byte {
//	// allocate space
//	nextWorld := make([][]byte, end-start)
//	for i := range nextWorld {
//		nextWorld[i] = make([]byte, p.ImageWidth)
//	}
//
//	directions := [8][2]int{
//		{-1, -1}, {-1, 0}, {-1, 1},
//		{0, -1}, {0, 1},
//		{1, -1}, {1, 0}, {1, 1},
//	}
//
//	for row := start; row < end; row++ {
//		for col := 0; col < p.ImageWidth; col++ {
//			// the alive must be set to 0 everytime when it comes to a different position
//			alive := 0
//			for _, dir := range directions {
//				// + imageHeight make sure the image is connected
//				newRow, newCol := (row+dir[0]+p.ImageHeight)%p.ImageHeight, (col+dir[1]+p.ImageWidth)%p.ImageWidth
//				if world[newRow][newCol] == 255 {
//					alive++
//				}
//			}
//			if world[row][col] == 255 {
//				if alive < 2 || alive > 3 {
//					nextWorld[row-start][col] = 0
//				} else {
//					nextWorld[row-start][col] = 255
//				}
//			} else if world[row][col] == 0 {
//				if alive == 3 {
//					nextWorld[row-start][col] = 255
//				} else {
//					nextWorld[row-start][col] = 0
//				}
//			}
//		}
//	}
//
//	return nextWorld
//}
//
//func calculateAliveCells(p Params, world [][]byte) []util.Cell {
//	var aliveCell []util.Cell
//	for row := 0; row < p.ImageHeight; row++ {
//		for col := 0; col < p.ImageWidth; col++ {
//			if world[row][col] == 255 {
//				aliveCell = append(aliveCell, util.Cell{X: col, Y: row})
//			}
//		}
//	}
//	return aliveCell
//}
