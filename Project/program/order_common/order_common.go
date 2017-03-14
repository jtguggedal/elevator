package order_common

import (
	"time"
	"fmt"
	"encoding/json"

	"./../backup"
)

const (
	OrderInternal = 1
	OrderExternal = 2
)

type OrderDirection int
type OrderType int

type Order struct {
	Id         int
	Type       OrderType
	Origin     string
	Floor      int
	Direction  OrderDirection
	AssignedTo string
	Done       bool
}

type OrderList []Order



func AddOrder(orders []Order, newOrder Order) []Order {
	if !OrderExists(orders, newOrder) {
		orders = append(orders, newOrder)
	}
	if newOrder.Type == OrderInternal {
		backup.SaveToFile(orders)
	}
	return orders
}

func RemoveDoneOrders(orders OrderList, doneOrder Order) OrderList {
	for key, order := range orders {
		if OrdersEqual(order, doneOrder) && doneOrder.Done == true {
			orders = append(orders[:key], orders[key+1:]...)
		}
	}
	return orders
}

func GetOrderId() int {
	return int(time.Now().UnixNano()/1e8 - 1488*1e7)
}

func JsonToStruct(input []byte, output interface{}) {
	temp := output
	err := json.Unmarshal(input, temp)
	if err == nil {
		output = temp
	} else {
		fmt.Println("Error decoding JSON:", err)
	}
}

func OrderToJson(id int, orderType OrderType, floor int, direction OrderDirection) []byte {
	ret, _ := json.Marshal(Order{
		Id:        id,
		Type:      orderType,
		Floor:     floor,
		Direction: direction})
	return ret
}

func OrdersEqual(o1, o2 Order) bool {
	if o1.Type == o2.Type &&
		o1.Floor == o2.Floor &&
		o1.Direction == o2.Direction {
		return true
	}
	return false
}

func OrderExists(orders []Order, candidateOrder Order) bool {
	for _, order := range orders {
		if (order.Type == candidateOrder.Type) &&
			(order.Floor == candidateOrder.Floor) &&
			(order.Direction == candidateOrder.Direction) {
			return true
		}
	}
	return false
}
