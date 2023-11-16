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
