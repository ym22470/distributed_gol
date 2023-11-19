package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var ProcessGol = "Server.ProcessWorld"
var AliveCells = "Server.CountAliveCell"
var Pause = "Server.PauseGol"

type Response struct {
	World          [][]byte
	AliveCells     []util.Cell
	CompletedTurns int
	CellCount      int
	Turns          int
	Pause          bool
	//TestStr        string
}

type Request struct {
	World     [][]byte
	Parameter Params
	Pause     bool
}
