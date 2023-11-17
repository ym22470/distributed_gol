package gol

import (
	"fmt"
	"net/rpc"
	"sync"
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
	var mutex sync.Mutex
	request := Request{World: world, Parameter: p}
	response := new(Response)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	// responseCell := new(Response)
	go func() {
		for range ticker.C {
			//requestCell := Request{World: world, Parameter: p}
			//responseCell := new(Response)
			mutex.Lock()
			client.Call(AliveCells, request, response)
			// fmt.Println("The turn is now: ", responseCell.Turns)
			c.events <- AliveCellsCount{CompletedTurns: response.Turns, CellsCount: response.CellCount}
			mutex.Unlock()
		}
	}()
	client.Call(ProcessGol, request, response)

	pasued := false
	resume := make(chan bool)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case key := <-c.key:
				switch key {
				case 's':
					c.ioCommand <- ioOutput
					c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth)
					for y := 0; y < p.ImageHeight; y++ {
						for x := 0; x < p.ImageWidth; x++ {
							c.ioOutput <- world[y][x]
						}
					}
					//fmt.Println("here in s")
				case 'q':
					c.ioCommand <- ioOutput
					c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth)
					for y := 0; y < p.ImageHeight; y++ {
						for x := 0; x < p.ImageWidth; x++ {
							c.ioOutput <- world[y][x]
						}
					}
					c.events <- FinalTurnComplete{CompletedTurns: response.CompletedTurns, Alive: response.AliveCells}
					c.ioCommand <- ioCheckIdle
					<-c.ioIdle
					c.events <- StateChange{response.CompletedTurns, Quitting}
					quit <- true
				case 'p':
					pasued = !pasued
					if pasued {
						c.events <- StateChange{response.CompletedTurns, Paused}
					} else {
						fmt.Println("Continuing")
						c.events <- StateChange{response.CompletedTurns, Executing}
						resume <- true
					}
				}
			case <-quit:
				return
			}

		}
	}()
	//report the final state of the world
	mutex.Lock()
	c.events <- FinalTurnComplete{CompletedTurns: response.CompletedTurns, Alive: response.AliveCells}
	mutex.Unlock()
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

	// Do remember to modify this ip address
	//	server := "127.0.0.1:8030"
	server := "3.80.132.4:8030"

	//create a client that dials to the tcp port
	client, _ := rpc.Dial("tcp", server)
	//close dial when everything is excuted
	defer client.Close()

	//fmt.Println("create a new world here")
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
	makeCall(client, world, p, c)
}
