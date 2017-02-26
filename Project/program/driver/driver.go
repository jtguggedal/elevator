package driver

/*
#cgo CFLAGS: -std=gnu11
#cgo LDFLAGS: -lcomedi -lm
#include "elev.h"
#include "channels.h"
#include "io.h"
*/
import "C"

//import "fmt"
import "time"

type MotorDirection int
type ButtonType int

const (
	DirectionDown MotorDirection = -1
	DirectionStop MotorDirection = 0
	DirectionUp   MotorDirection= 1
)

const (
	ButtonExternalUp   		= 0
	ButtonExternalDown 		= 1
	ButtonInternalOrder   	= 2
)

const (
	NUMBER_OF_FLOORS  = int(C.N_FLOORS)
	NUMBER_OF_BUTTONS = int(C.N_BUTTONS)
)

type ButtonEvent struct {
	Type 	ButtonType
	Floor  	int
	Status 	int
}

func ElevatorDriverInit(simulator bool) {
	if simulator {
		C.elev_init(C.ET_Simulation)
	} else {
		C.elev_init(C.ET_Comedi)
	}


	SetStopLamp(1)
	SetDoorOpenLamp(1)
	SetMotorDirection(DirectionDown)

	for GetFloorSensorSignal() == -1 {
		// Wait for elevator to get down to closest floor and thereby known state
	}
	SetMotorDirection(DirectionStop)
}

// Function for enabling event listener for elevator and button events
// TODO: call this from init function?
func EventListener(
		buttonEventChannel chan<- ButtonEvent,
		floorEventChannel chan<- int) {

	var prevFloorSignal int
	var currentFloorSignal int
	for {

		// Passing on which floor the elevator is at if it has changed
		currentFloorSignal = GetFloorSensorSignal()
		if currentFloorSignal != prevFloorSignal {
			floorEventChannel <- currentFloorSignal
			prevFloorSignal = currentFloorSignal
		}
		// Polling all buttons
		for floor := 0; floor < NUMBER_OF_FLOORS; floor++ {
			for button := 0; button < NUMBER_OF_BUTTONS; button++ {
				if GetButtonSignal(button, floor) != 0 {
					buttonEventChannel <- ButtonEvent{
						Floor:  floor,
						Type: ButtonType(button),
						Status: GetButtonSignal(button, floor)}
					time.Sleep(500 * time.Millisecond)
				}

			}
		}
	}
}

func SetMotorDirection(direction MotorDirection) {
	C.elev_set_motor_direction(C.elev_motor_direction_t(direction))
}

func SetButtonLamp(button ButtonType, floor, value int) {
	C.elev_set_button_lamp(C.elev_button_type_t(button), C.int(floor), C.int(value))
}

func SetFloorIndicator(floor int) {
	C.elev_set_floor_indicator(C.int(floor))
}

func SetDoorOpenLamp(value int) {
	C.elev_set_door_open_lamp(C.int(value))
}

func SetStopLamp(value int) {
	C.elev_set_stop_lamp(C.int(value))
}

func GetButtonSignal(button int, floor int) int {
	return int(C.elev_get_button_signal(C.elev_button_type_t(button), C.int(floor)))
}

func GetFloorSensorSignal() int {
	return int(C.elev_get_floor_sensor_signal())
}

func GetStopSignal() int {
	return int(C.elev_get_stop_signal())
}

func GetObstructionSignal() int {
	return int(C.elev_get_obstruction_signal())
}
