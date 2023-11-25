package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var BrokerAliveCells = "Broker.CountAliveCell"
var BrokerGol = "Broker.ProcessWorld"
var BrokerKey = "Broker.KeyGol"
var Key = "Server.KeyGol"
var ProcessGol = "Server.ProcessWorld"
var AliveCells = "Server.CountAliveCell"
var Shutdown = "Server.Shutdown"
var Live = "Broker.GetLive"

type Response struct {
	World          [][]byte
	AliveCells     []util.Cell
	CompletedTurns int
	CellCount      int
	Turns          int
	Cells          util.Cell
}

type Request struct {
	World     [][]byte
	Parameter Params
	P         bool
	S         bool
	Q         bool
	K         bool
	Resume    bool
	Start     int
	End       int
	Turn      int
	OldWorld  [][]byte
	NewWorld  [][]byte
	Cell      util.Cell
}
