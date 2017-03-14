package order_cost

import (
    "math"

    "./../fsm"
    . "./../order_common"
)

const (
	Idle             = 2
	MovingSameDir    = 1
	MovingOpositeDir = 10
	PerFloor         = 3
	PerOrderDepth    = 10
	Stuck			 = 10000
)


func OrderCost(orders OrderList, order Order, elevatorData fsm.ElevatorData_t) int {
	var orderCost int
	distance := order.Floor - elevatorData.Floor

	idle := elevatorData.State == fsm.Idle
	movingSameDir := (int(order.Direction) == int(elevatorData.State)) && !idle
	movingOpositeDir := !movingSameDir && !idle
	stuck := elevatorData.State == fsm.Stuck

	targetAbove := distance > 0
	targetBelow := distance < 0
	distance = int(math.Abs(float64(distance)))

	if targetAbove || targetBelow {
		orderCost += distance * PerFloor
	}
	if stuck {
		orderCost += Stuck
	}
	if movingSameDir {
		orderCost += MovingSameDir
	} else if movingOpositeDir {
		orderCost += MovingOpositeDir
	} else if idle {
		orderCost += Idle
	}

	if order.Type == OrderExternal {
		orderDepth := 0
		for _, ord := range orders {
			if ord.AssignedTo == elevatorData.Id {
				orderDepth += 1
			}
		}
		orderCost += orderDepth * PerOrderDepth
	}
	//fmt.Println("Order cost:", orderCost, o)
	return orderCost
}

// Function that, based on cost function, returns the next order to take fram an order list
func GetNextOrder(orders OrderList, localId string) (Order, bool) {
	if len(orders) == 0 {
		return Order{Floor: -1}, false
	}
	oldestEntryTime := int(10e15)
	var nextOrder Order
	nextOrder.Floor = -1
	lowestCost := 1000000
	elevatorData := fsm.GetElevatorData()

	// Serving internal orders first
	if orders[0].Type == OrderInternal {
		for _, order := range orders {
			cost := OrderCost(orders, order, elevatorData)
			if cost < lowestCost {
				lowestCost = cost
				nextOrder = order
			}
		}
		return nextOrder, nextOrder.Floor != -1
	}
	for _, order := range orders {
		if order.Id < oldestEntryTime && string(order.AssignedTo) == localId {
			oldestEntryTime = order.Id
			nextOrder = order
		}
	}
	for _, order := range orders {
		if TakeOrderOnTheWay(order, nextOrder) && string(order.AssignedTo) == localId {
			nextOrder = order
		}
	}
	return nextOrder, nextOrder.Floor != -1
}

// Function that returns true if a given order is on the way from current elevator location to target floor
func TakeOrderOnTheWay(order, activeOrder Order) bool {
	elevatorState := fsm.GetElevatorData()
	if activeOrder.Direction == order.Direction {
		if (elevatorState.Floor > order.Floor) && (order.Floor > activeOrder.Floor) {
			return true
		} else if elevatorState.Floor < order.Floor && order.Floor < activeOrder.Floor {
			return true
		}
	}
	return false
}
