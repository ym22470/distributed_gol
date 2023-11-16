package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var ProcessGol = "Server.ProcessWorld"
var AliveCells = "Server.CountAliveCell"

type Response struct {
	World          [][]byte
	AliveCells     []util.Cell
	CompletedTurns int
	CellCount      int
	Turns          int
}

type Request struct {
	World     [][]byte
	Parameter Params
}
