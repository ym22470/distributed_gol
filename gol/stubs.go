package gol

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var BrokerAliveCells = "Broker.GolAliveCells"
var Initializer = "Broker.GolInitializer"
var BrokerKey = "Broker.GolKey"
var Key = "Server.KeyGol"
var ProcessGol = []string{
	"Server.ProcessWorld1",
	"Server.ProcessWorld2",
	"Server.ProcessWorld3",
	"Server.ProcessWorld4",
}
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
	P         bool
	S         bool
	Q         bool
	K         bool
	Resume    bool
}
