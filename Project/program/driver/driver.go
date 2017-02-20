package driver

/*
#cgo CFLAGS: -std=c11
#cgo LDFLAGS: -lcomedi -lm
#include "elev.h"
*/
import "C"

//import "fmt"
import "time"

type MotorDirection int
type ButtonType int

const (
	MOTOR_DIRECTION_STOP = 0
	MOTOR_DIRECTION_UP   = 1
	MOTOR_DIRECTION_DOWN = -1
)

const (
	BUTTON_CALL_UP   = 0
	BUTTON_CALL_DOWN = 1
	BUTTON_COMMAND   = 2
)

const (
	NUMBER_OF_FLOORS  = int(C.N_FLOORS)
	NUMBER_OF_BUTTONS = int(C.N_BUTTONS)
)

type ButtonEvent struct {
	Button int
	Floor  int
	Status int
}

func ElevatorDriverInit() {
	C.elev_init()

	SetStopLamp(1)
	SetDoorOpenLamp(1)
	SetMotorDirection(MOTOR_DIRECTION_DOWN)

	for GetFloorSensorSignal() == -1 {
		// Wait for elevator to get down to closest floor and thereby known state
	}
	SetMotorDirection(MOTOR_DIRECTION_STOP)
}

// Function for polling buttons. Returns struct with button type, floor and button state when button is pressed
func EventListener(buttonEventChannel chan ButtonEvent, floorEventChannel chan int) {
	var prevFloor int
	var currentFloorSignal int
	for {
		currentFloorSignal = GetFloorSensorSignal()
		if currentFloorSignal != prevFloor && currentFloorSignal >= 0 {
			//driver.SetFloorIndicator(driver.GetFloorSensorSignal())
			floorEventChannel <- currentFloorSignal
			prevFloor = currentFloorSignal
		}
		if currentFloorSignal == -1 {
			// Elevator between two floors
		}
		for floor := 0; floor < NUMBER_OF_FLOORS; floor++ {
			for button := 0; button < NUMBER_OF_BUTTONS; button++ {
				if GetButtonSignal(button, floor) != 0 {
					buttonEventChannel <- ButtonEvent{
						Floor:  floor,
						Button: button,
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
