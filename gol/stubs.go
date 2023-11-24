package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var BrokerAliveCells = "Broker.GolAliveCells"
var Initializer = "Broker.GolInitializer"
var BrokerKey = "Broker.GolKey"
var Key = "Server.KeyGol"
var ProcessGol = "Server.ProcessWorld"

type Response struct {
	World      [][]byte
	Slice      [][]byte
	AliveCells []util.Cell
	//CompletedTurns int
	CellCount int
	Turns     int
	End       bool
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
}
