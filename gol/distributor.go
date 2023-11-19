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
	//resume := make(chan bool)
	var mutex sync.Mutex
	request := Request{World: world, Parameter: p, Pause: false}
	response := new(Response)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	// responseCell := new(Response)
	go func() {
		for range ticker.C {
			if !response.Pause {
				//requestCell := Request{World: world, Parameter: p}
				//responseCell := new(Response)
				mutex.Lock()
				client.Call(AliveCells, request, response)
				// fmt.Println("The turn is now: ", responseCell.Turns)
				c.events <- AliveCellsCount{CompletedTurns: response.Turns, CellsCount: response.CellCount}
				mutex.Unlock()
				//<-resume
			}

		}
	}()

	//Keypress function
	//pasued := false
	//resume := make(chan bool)
	quit := make(chan bool)
	go func() {
		//fmt.Println("here in s")
		for {
			select {
			case key := <-c.key:
				switch key {
				case 's':
					//fmt.Println("here in s")
					c.ioCommand <- ioOutput
					c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth, response.Turns)
					for y := 0; y < p.ImageHeight; y++ {
						for x := 0; x < p.ImageWidth; x++ {
							c.ioOutput <- response.World[y][x]
						}
					}
				case 'q':
					//fmt.Println("here in q")
					c.ioCommand <- ioOutput
					c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth, response.Turns)
					for y := 0; y < p.ImageHeight; y++ {
						for x := 0; x < p.ImageWidth; x++ {
							c.ioOutput <- response.World[y][x]
						}
					}
					c.events <- FinalTurnComplete{CompletedTurns: response.CompletedTurns, Alive: response.AliveCells}
					c.ioCommand <- ioCheckIdle
					<-c.ioIdle
					c.events <- StateChange{response.Turns, Quitting}
					quit <- true
				case 'p':
					fmt.Println("haha")
					mutex.Lock()
					client.Call(Pause, request, response)
					fmt.Println("haha")
					mutex.Unlock()
					//pasued = !pasued
					if response.Pause {
						fmt.Println("Paused,press p to resume")
						c.events <- StateChange{response.Turns, Paused}
					} else if !response.Pause {
						fmt.Println("Continuing")
						c.events <- StateChange{response.Turns, Executing}
						//resume <- true
					}
				case 'k':

				}
			case <-quit:
				//fmt.Println("here in s")
				return
			}
		}
	}()
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
	server := "127.0.0.1:8030"
	//server := "3.80.132.4:8030"

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
