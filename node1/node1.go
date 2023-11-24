package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"sync"

	"uk.ac.bris.cs/gameoflife/gol"
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
	fmt.Println("Into PW in node 3")
	worldPiece := nextState(req.Parameter, req.World, req.Start, req.End)
	res.World = make([][]byte, req.End-req.Start)
	for i := range res.World {
		res.World[i] = make([]byte, req.Parameter.ImageWidth)
	}
	//res.World = copySlice(worldPiece)
	for i := req.Start; i < req.End; i++ {
		copy(res.World[i-req.Start], worldPiece[i-req.Start])
	}
	return nil
}

func (s *Server) CountAliveCell(req gol.Request, res *gol.Response) error {
	mutex.Lock()
	res.Turns = s.Turn
	res.CellCount = s.CellCount
	mutex.Unlock()
	return nil
}

func (s *Server) Shutdown(req gol.Request, res *gol.Response) error {
	os.Exit(0)
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

func copySlice(src [][]byte) [][]byte {
	dst := make([][]byte, len(src))
	for i := range src {
		dst[i] = make([]byte, len(src[i]))
		copy(dst[i], src[i])
	}
	return dst
}

func main() {
	fmt.Println("Node3")
	pAddr := flag.String("port", "8040", "port to listen on")
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
			fmt.Println(err)
		}
	}(listener)
	rpc.Accept(listener)
}
