package gol

import (
	"fmt"
	"net/rpc"
	"time"
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
	client.Call(ProcessGol, request, response)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	go func() {
		request1 := Request{World: world, Parameter: p}
		response1 := new(Response)
		for range ticker.C {
			fmt.Println("here")
			client.Call(ProcessGol, request1, response1)
			c.events <- AliveCellsCount{CompletedTurns: response1.TurnCount, CellsCount: response1.CellCount}
		}
	}()

	//report the final state of the world
	c.events <- FinalTurnComplete{CompletedTurns: response.CompletedTurns, Alive: response.AliveCells}
	// Make sure that the Io has finished any output before exiting.

	c.ioCommand <- ioCheckIdle

	<-c.ioIdle
	c.events <- StateChange{response.CompletedTurns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

// func cellCount(client *rpc.Client, world [][]byte, p Params, c distributorChannels) {
// 	request := Request{World: world, Parameter: p}
// 	response := new(Response)
// 	client.Call(ProcessGol, request, response)
// 	c.events <- AliveCellsCount{CompletedTurns: response.CompletedTurns, CellsCount: response.CellCount}
// 	close(c.events)
// }

func distributor(p Params, c distributorChannels) {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%vx%v", p.ImageHeight, p.ImageWidth)

	// Do remember to modify this ip address
	server := "127.0.0.1:8030"
	//create a client that dials to the tcp port
	client, _ := rpc.Dial("tcp", server)
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
		}
	}

	// ticker := time.NewTicker(2 * time.Second)
	// defer ticker.Stop()
	// go func() {
	// 	for range ticker.C {
	// 		cellCount(client, world, p, c)
	// 	}
	// }()

	makeCall(client, world, p, c)
}
