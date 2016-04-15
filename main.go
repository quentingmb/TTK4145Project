package main

import (
	"driver"
	"Elevator"
	. "fmt"
	"extra"
	"Network"
	"ElevatorLogic"
	"runtime"
	"time"
)

func main() {
	var myinfo Network.Info
	var takerequest []Network.Request

	runtime.GOMAXPROCS(runtime.NumCPU())

	localip := Network.GetLocalIP()
	Println(localip)
	myinfo.Ipsource = localip

	conf := extra.LoadConfig("./config/conf.json")

	generatedmessages_c := make(chan Network.ElevatorMessage, 100)
	go Network.NetworkManager(conf, localip, generatedmessages_c)

	state := "INIT"
	driver.Init(driver.ET_comedi)
	Elevator.ElevInit()

	for {
		time.Sleep(10 * time.Millisecond)
		myinfo.State = state
		Elevator.UpdateFloor()
		myinfo.PreviousFloor = Elevator.CurrentFloor()
		Network.NewInfo(myinfo, generatedmessages_c)
		switch state {
		case "INIT":
			{
				Elevator.SetElevSpeed(-300)
			}
		case "IDLE":
			{
				Elevator.SetElevSpeed(0)
			}
		case "UP":
			{
				Elevator.SetElevSpeed(300)
			}
		case "DOWN":
			{
				Elevator.SetElevSpeed(-300)
			}
		case "DOOR_OPEN":
			{
				Elevator.SetElevDoorOpenLamp(1)
				for _, request := range takerequest {
					request.Type = 0
					Println("Deleting request: ", request)
					time.Sleep(10 * time.Millisecond)
					Network.NewRequest(generatedmessages_c, request)
				}
				Elevator.SetElevSpeed(0)
				time.Sleep(3000 * time.Millisecond)
				Elevator.SetElevDoorOpenLamp(0)
			}
		case "ERROR":
			{
				Elevator.SetElevSpeed(0)
			}
		}
		state, takerequest = ElevatorLogic.Nextstate(localip, conf.Elevators, myinfo.State)
	}
}
