package order_cost

import (
	. "./../order_handler"
	"./../fsm"
	"math"
)

const (
	costIdle				= 2
	costMovingSameDir		= 1
	costMovingOpositeDir	= 5
	costPerFloor			= 2
)


func sortOrders(	peerUpdateChannel <-chan network.PeerStatus,
					newOrderChannel <-chan,
					doneOrderChannel <-chan) (Order) {

	var nodes []network.PeerStatus
	

	for {
		select {
			case nodes = <- peerUpdateChannel:
			case order := <- newOrderChannel:
			case order := <- doneOrderChannel:

		}
	}
}


func orderCost(o Order, e fsm.ElevatorData_t) {
	var cost int
	distance := o.Floor - e.Floor

	idle := e.State != fsm.Idle
	movingSameDir := (o.Direction == e.State) && !idle
	movingOpositeDir := !movingSameDir && !idle

	targetSameFloor := distance == 0
	targetAbove := distance > 0
	targetBelow := distance < 0
	distance = int(math.Abs(distance))

	cost = targetSameFLoor ? cost : movingSameDir
	if targetAbove  || targetBelow {
		cost += distance * costPerFloor
	}
	if movingSameDir {
		cost += costMovingSameDir
	} else if movingSameDir {
		cost += costMovingOpositeDir
	} else if idle {
		cost += costIdle
	}
}
