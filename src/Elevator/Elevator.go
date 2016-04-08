//A completer 
//ajouter les imports

package Elevator
import (
	"driver"
	"maths"
	"time"
)

const N_FlOORS = 4
const N_BUTTONS = 4

type Elev_button int

const (
		BUTTON_CALL_UP Elev_button = iota
		BUTTON_CALL_DOWN
		BUTTON_COMMAND
) 

var Lamp_channel_matrix = [4][N_BUTTONS]int{
	{driver.LIGHT_UP1, driver.LIGHT_DOWN1, driver.LIGHT_COMMAND1},
	{driver.LIGHT_UP2, driver.LIGHT_DOWN2, driver.LIGHT_COMMAND2},
	{driver.LIGHT_UP3, driver.LIGHT_DOWN3, driver.LIGHT_COMMAND3},
	{driver.LIGHT_UP4, driver.LIGHT_DOWN4, driver.LIGHT_COMMAND4},
}

var Button_channel_matrix = [4][N_BUTTONS]int{
	{driver.BUTTON_UP1, driver.BUTTON_DOWN1, driver.BUTTON_COMMAND1},
	{driver.BUTTON_UP2, driver.BUTTON_DOWN2, driver.BUTTON_COMMAND2},
	{driver.BUTTON_UP3, driver.BUTTON_DOWN3, driver.BUTTON_COMMAND3},
	{driver.BUTTON_UP4, driver.BUTTON_DOWN4, driver.BUTTON_COMMAND4},
}

func SetElevSpeed(speed int) {
	if speed == 0 {
		if driver.Read_bit(driver.MOTORDIR)!=0 {
			driver.Clear_bit(driver.MOTORDIR)
		} else {
			driver.Set_bit(driver.MOTORDIR)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if speed > 0 {
		driver.Clear_bit(driver.MOTORDIR)
	} else {
		driver.Set_bit(driver.MOTORDIR)
	}
	driver.Write_analog(driver.MOTOR, 2048+4*maths.Abs(speed))
}
func CurrentFloor() int {
	floor := 1
	if driver.Read_bit(driver.LIGHT_FLOOR_IND2)!=0 {
		floor = floor + 1
	}
	if driver.Read_bit(driver.LIGHT_FLOOR_IND1)!=0 {
		floor = floor + 2
	}
	return floor
}

func SetElevFloorIndicator(floor int) {
	//one light sould be on
	if floor == 3 || floor == 4 {
		driver.Set_bit(driver.LIGHT_FLOOR_IND1)
	} else {
		driver.Clear_bit(driver.LIGHT_FLOOR_IND1)
	}
	if floor == 2 || floor == 4 {
		driver.Set_bit(driver.LIGHT_FLOOR_IND2)
	} else {
		driver.Clear_bit(driver.LIGHT_FLOOR_IND2)
	}
}
func SetElevButtonLamp(button Elev_button, floor int, value int) {
	if value == 1 {
		driver.Set_bit(Lamp_channel_matrix[floor-1][button])
	} else {
		driver.Clear_bit(Lamp_channel_matrix[floor-1][button])
	}
}
func SetElevDoorOpenLamp(value int) {
	if value == 1 {
		driver.Set_bit(driver.LIGHT_DOOR_OPEN)
	} else {
		driver.Clear_bit(driver.LIGHT_DOOR_OPEN)
	}
}
func SetElevStopLamp(value int) {
	if value == 1 {
		driver.Set_bit(driver.LIGHT_STOP)
	} else {
		driver.Clear_bit(driver.LIGHT_STOP)
	}
}
func GetElevFloorSensorSignal() int {
	if driver.Read_bit(driver.SENSOR_FLOOR1)!=0 {
		return 1
	} else if driver.Read_bit(driver.SENSOR_FLOOR2)!=0 {
		return 2
	} else if driver.Read_bit(driver.SENSOR_FLOOR3)!=0 {
		return 3
	} else if driver.Read_bit(driver.SENSOR_FLOOR4)!=0 {
		return 4
	} else {
		return -1
	}

}

func GetElevButtonSignal(button Elev_button, floor int) int {
	if driver.Read_bit(Button_channel_matrix[floor-1][button])!=0 {
		return 1
	} else {
		return 0
	}
}
func GetElevStopSignal() bool {
	return driver.Read_bit(driver.STOP)!=0
}
func GetElevObstructionSignal() bool {
		return driver.Read_bit(driver.OBSTRUCTION)!=0
}
func ElevAtFloor() bool {
	if GetElevFloorSensorSignal() != -1 {
		return true
	}
	return false
}
func UpdateFloor() {
	floor := GetElevFloorSensorSignal()
	if floor != -1 {
		SetElevFloorIndicator(floor)
	}
}
func ElevInit() {
	// Zero all floor button lamps
	for i := 1; i < 4+1; i++ {    // Faut apres remplacer le 4 par n_floors
		if i != 1 {
			SetElevButtonLamp(BUTTON_CALL_DOWN, i, 0)
		}
		if i != 4 {
			SetElevButtonLamp(BUTTON_CALL_UP, i, 0)
		}
		SetElevButtonLamp(BUTTON_COMMAND, i, 0)
	}
	// Clear stop lamp, door open lamp, and set floor indicator to ground floor.
	SetElevStopLamp(0)
	SetElevDoorOpenLamp(0)
	SetElevFloorIndicator(0)

}
