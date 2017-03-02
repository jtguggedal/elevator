package order_cost

import (
	"./../fsm"
	"./../network"
	"math"
)

type orderType int

type orderDirection int

type Order struct {
	Id			int
	Type		orderType
	Origin		network.Ip
	Floor		int
	Direction	orderDirection
	AssignedTo	int
	Done 		bool
}

const (
	costIdle				= 2
	costMovingSameDir		= 1
	costMovingOpositeDir	= 5
	costPerFloor			= 2
)


func sortOrders(	peerUpdateChannel <-chan network.PeerStatus,
					newOrderChannel <-chan Order,
					doneOrderChannel <-chan Order)  {

	//var liveNodes []network.PeerStatus


	for {
		select {
			//case nodes := <- peerUpdateChannel:
			//case order := <- newOrderChannel:
			//case order := <- doneOrderChannel:

		}
	}
}


func OrderCost(o Order, e fsm.ElevatorData_t) {
	var cost int
	distance := o.Floor - e.Floor

	idle := e.State != fsm.Idle
	movingSameDir := (int(o.Direction) == int(e.State)) && !idle
	movingOpositeDir := !movingSameDir && !idle

	//targetSameFloor := distance == 0
	targetAbove := distance > 0
	targetBelow := distance < 0
	distance = int(math.Abs(float64(distance)))

	if targetAbove  || targetBelow {
		cost += distance * costPerFloor
	}
	if movingSameDir {
		cost += costMovingSameDir
	} else if movingOpositeDir {
		cost += costMovingOpositeDir
	} else if idle {
		cost += costIdle
	}
}
