package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var ProcessGol = "Server.ProcessWorld"

type Response struct {
	World          [][]byte
	AliveCells     []util.Cell
	CompletedTurns int
}

type Request struct {
	World     [][]byte
	Parameter Params
}
