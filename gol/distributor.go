package gol

import (
	"fmt"
	"net/rpc"
	"os"
	"sync"
	"time"
)

var wg sync.WaitGroup

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
	pasued := false
	kill := false
	request := Request{World: world, Parameter: p}
	response := new(Response)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	// responseCell := new(Response)
	go func() {
		for range ticker.C {
			mutex.Lock()
			if !pasued && !kill {
				mutex.Unlock()
				//requestCell := Request{World: world, Parameter: p}
				//responseCell := new(Response)
				mutex.Lock()
				err := client.Call(BrokerAliveCells, request, response)
				if err != nil {
					return
				}
				c.events <- AliveCellsCount{CompletedTurns: response.Turns, CellsCount: response.CellCount}
				mutex.Unlock()
			} else {
				mutex.Unlock()
			}
		}
	}()

	quit := make(chan bool)
	go func() {
		for {
			select {
			case key := <-c.key:
				switch key {
				case 's':
					requestkey := Request{S: true}
					err := client.Call(BrokerKey, requestkey, response)
					if err != nil {
						return
					}
					c.ioCommand <- ioOutput
					c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth, response.Turns)
					for y := 0; y < p.ImageHeight; y++ {
						for x := 0; x < p.ImageWidth; x++ {
							c.ioOutput <- response.World[y][x]
						}
					}
				case 'q':
					// requestkey := Request{Q: true}
					// TODO: q function
					// client.Call(Key, requestkey, response)
					c.events <- FinalTurnComplete{CompletedTurns: response.CompletedTurns, Alive: response.AliveCells}
					c.ioCommand <- ioCheckIdle
					<-c.ioIdle
					c.events <- StateChange{response.Turns, Quitting}
					quit <- true
				case 'p':
					mutex.Lock()
					pasued = !pasued
					requestkey := Request{P: true, Resume: pasued}
					mutex.Unlock()
					err := client.Call(BrokerKey, requestkey, response)
					if err != nil {
						return
					}
					if pasued {
						c.events <- StateChange{response.Turns, Paused}
					} else {
						fmt.Println("Continuing")
						c.events <- StateChange{response.Turns, Executing}
					}
				case 'k':
					wg.Add(1)
					requestkey := Request{S: true}
					err := client.Call(BrokerKey, requestkey, response)
					if err != nil {
						return
					}
					c.ioCommand <- ioOutput
					c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageHeight, p.ImageWidth, response.Turns)
					for y := 0; y < p.ImageHeight; y++ {
						for x := 0; x < p.ImageWidth; x++ {
							c.ioOutput <- response.World[y][x]
						}
					}
					requestkey = Request{K: true}
					mutex.Lock()
					kill = true
					mutex.Unlock()
					client.Call(BrokerKey, requestkey, response)
					c.ioCommand <- ioCheckIdle
					<-c.ioIdle
					c.events <- StateChange{response.Turns, Quitting}
					fmt.Println("reached end")
					wg.Done()
					quit <- true
					os.Exit(0)
				}
			case <-quit:
				close(quit)
				return
			}
		}
	}()
	client.Call(Initializer, request, response)
	//wait until the keypress goroutine finishes
	wg.Wait()
	//if the specified last tern is finished, quit with output pgm
	if response.End {
		//fmt.Println(len(response.World))
		//send the content of world and receive on the other side(writePgm) concurrently
		c.ioCommand <- ioOutput
		// Since the output here is only required when the turns are ran out, so doesn't need the ticker anymore
		mutex.Lock()
		pasued = true
		mutex.Unlock()
		if p.Turns == 0 {
			c.ioFilename <- fmt.Sprintf("%dx%dx0", p.ImageHeight, p.ImageWidth)
		} else if p.Threads == 1 {
			c.ioFilename <- fmt.Sprintf("%dx%dx%d", p.ImageHeight, p.ImageWidth, p.Turns)
		} else {
			c.ioFilename <- fmt.Sprintf("%dx%dx%d-%d", p.ImageHeight, p.ImageWidth, p.Turns, p.Threads)
		}
		//send the completed world to ioOutput c
		mutex.Lock()
		//fmt.Println(len(response.World[0]))
		for i := 0; i < p.ImageHeight; i++ {
			for j := 0; j < p.ImageWidth; j++ {
				c.ioOutput <- response.World[i][j]
			}
		}
		mutex.Unlock()

		//report the final state of the world
		mutex.Lock()
		c.events <- FinalTurnComplete{CompletedTurns: response.CompletedTurns, Alive: response.AliveCells}
		mutex.Unlock()
		// Make sure that the Io has finished any output before exiting.
		fmt.Println("send complete")
		c.ioCommand <- ioCheckIdle
		fmt.Println("send complete")
		fmt.Println(len(response.AliveCells))

		<-c.ioIdle
		c.events <- StateChange{response.CompletedTurns, Quitting}

		// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
		close(c.events)
	} else {
		mutex.Lock()
		c.events <- FinalTurnComplete{CompletedTurns: response.CompletedTurns, Alive: response.AliveCells}
		mutex.Unlock()
		// Make sure that the Io has finished any output before exiting.
		fmt.Println("send complete")
		c.ioCommand <- ioCheckIdle
		fmt.Println("send complete")
		fmt.Println(len(response.AliveCells))

		<-c.ioIdle
		c.events <- StateChange{response.CompletedTurns, Quitting}

		// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
		close(c.events)
	}
}

func distributor(p Params, c distributorChannels) {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%vx%v", p.ImageHeight, p.ImageWidth)

	// Do remember to modify this ip address
	broker := "127.0.0.1:8030"
	//server := "54.224.85.190:8030"

	//create a client that dials to the tcp port
	client, _ := rpc.Dial("tcp", broker)
	//close dial when everything is excuted
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {

		}
	}(client)

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
