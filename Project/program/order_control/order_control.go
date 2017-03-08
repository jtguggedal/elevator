package order_control

import (
	"./../driver"
	"./../network/peers"
	"./../network"
	"./../fsm"
	"./../order_cost"
	"time"
	"fmt"
	"encoding/json"
	"math"
)

const (
	directionUp		= driver.ButtonExternalUp
	directionDown	= driver.ButtonExternalDown
)

const (
	orderInternal = 1
	orderExternal = 2
)

const (
	costIdle				= 2
	costMovingSameDir		= 1
	costMovingOpositeDir	= 6
	costPerFloor			= 3
)


type orderDirection int
type orderType int

type Order struct {
	Id			int
	Type		orderType
	Origin		network.Ip
	Floor		int
	Direction	orderDirection
	AssignedTo	network.Ip
	Done 		bool
}

type orderList []Order

var localId network.Ip
