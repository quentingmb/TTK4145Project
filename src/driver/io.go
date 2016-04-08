
package driver  // where "driver" is the folder that contains io.go, io.c, io.h, channels.go, channels.h and driver.go
/*
#cgo CFLAGS: -std=c11
#cgo LDFLAGS: -lcomedi -lm
#include "io.h"
*/
import "C"

func Init(elevatorType int) int {
	return int(C.io_init(C.ElevatorType(elevatorType)))
}

func Set_bit(channel int) {
	C.io_set_bit(C.int(channel))
}

func Clear_bit(channel int) {
	C.io_clear_bit(C.int(channel))
}

func Write_analog(channel int, value int) {
	C.io_write_analog(C.int(channel), C.int(value))
}

func Read_bit(channel int) int {
	return int(C.io_read_bit(C.int(channel)))
}

func Read_analog(channel int) int {
	return int(C.io_read_analog(C.int(channel)))
}