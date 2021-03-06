package ElevatorLogic

import (
	"Elevator"
	"Network"
	"extra"
	"fmt"
)

var internalrequest Network.Request
var externalrequest Network.Request

func InternalOrExternal(request Network.Request) bool {

	if request.Direction == Elevator.BUTTON_COMMAND {
		return true
	} else {
		return false
	}
}

func Externalrequest(localip string, Elevatorlist []extra.Elevator, request Network.Request) {
	var statelist = make(map[string]Network.Info)
	infolist := Network.GetInfoList()
	for host, info := range infolist {
		statelist[host] = info
	}
	Externalloop:
	for _, elevator := range Elevatorlist {
			if info, ok := statelist[elevator.Address]; ok {
				if ((info.State == "UP" || info.State == "IDLE") && info.PreviousFloor <= request.Floor) || ((info.State == "DOWN" || info.State == "IDLE") && info.PreviousFloor >= request.Floor) {
					if info.Ipsource == request.Ipsource {
						if info.Ipsource == localip {
							return request
						} else {
							delete(statelist, elevator.Address)
							continue Externalloop
						}
					}
				}
			}
		}
	
		for _, elevator := range Elevatorlist {
			if info, ok := statelist[elevator.Address]; ok {
				if (info.State == "UP" && info.PreviousFloor >= request.Floor) || (info.State == "DOWN" && info.PreviousFloor <= request.Floor){
					if info.Ipsource == request.Ipsource {
						if info.Ipsource == localip {
							return request
						} else {
							delete(statelist, elevator.Address)
							continue Externalloop
						}
					}
				}
			}
		}
	}

func Internalrequest(localip string, Elevatorlist []extra.Elevator, request Network.Request) {
	var statelist = make(map[string]Network.Info)
	infolist := Network.GetInfoList()
	for host, info := range infolist {
		statelist[host] = info
	}
	Internalloop:
	for i := 0; i < 4; i++ { // N_FLOORS ca marche pas
		for _, elevator := range Elevatorlist {
			if info, ok := statelist[elevator.Address]; ok {
				if i != 0 && (info.State == "UP" && info.PreviousFloor+i == request.Floor) || (info.State == "DOWN" && info.PreviousFloor-i == request.Floor) {
					if statelist[elevator.Address].Ipsource == localip {						
						internalrequest = request
					} else {
						delete(statelist, elevator.Address)
						continue Internalloop					
					}
				}
			}
		}
		for _, elevator := range Elevatorlist {
			if info, ok := statelist[elevator.Address]; ok {
				if info.State == "IDLE" && (info.PreviousFloor == request.Floor+i || info.PreviousFloor == request.Floor-i) {
					if statelist[elevator.Address].Ipsource == localip {
						internalrequest = request						
					} else {
						delete(statelist, elevator.Address)	
						continue Internalloop					
					}
				}
			}
		}
	}

}

// Nextrequest is made to look into all request in order to get the best elevator
func Nextrequest(localip string, Elevatorlist []extra.Elevator) Network.Request {
	var statelist = make(map[string]Network.Info)
	infolist := Network.GetInfoList()
	for host, info := range infolist {
		statelist[host] = info
	}
	requestlist := Network.GetRequestList()

loop:
	for _, request := range requestlist {
		if !InternalOrExternal(request) {
			go Externalrequest(localip, Elevatorlist, request)
			return externalrequest
			continue loop
		} else {
			go Internalrequest(localip, Elevatorlist, request)
			return internalrequest
			continue loop
		}
	}
	return Network.NoRequest[0]
}

//Stop is made to return the request to the elevator, it will stop for these requests
func Stop(localip string, mystate string) []Network.Request {
	var takerequest []Network.Request
	requestlist := Network.GetRequestList()
	for _, request := range requestlist {
		if (request.Direction == Elevator.BUTTON_COMMAND && request.Ipsource == localip) || (request.Direction == Elevator.BUTTON_CALL_UP && mystate == "UP") || (request.Direction == Elevator.BUTTON_CALL_DOWN && mystate == "DOWN") {
			if request.Floor == Elevator.CurrentFloor() && Elevator.ElevAtFloor() {
				takerequest = append(takerequest, request)
			}
		}
	}
	return takerequest
}

//This function sends to an elevator its next state
func Nextstate(localip string, elevators []extra.Elevator, mystate string) (string, []Network.Request) {
	if Elevator.GetElevObstructionSignal() {
		Elevator.SetElevStopLamp(1)
		return "ERROR", nil
	} else if mystate == "ERROR" {
		Elevator.SetElevStopLamp(0)
		return "INIT", nil
	}

	stop := Stop(localip, mystate)
	if len(stop) != 0 {
		return "DOOR_OPEN", stop
	}

	next := Nextrequest(localip, elevators)
	if Elevator.ElevAtFloor() && next.Floor == Elevator.CurrentFloor() {
		return "DOOR_OPEN", append(stop, next)
	}
	if next.Floor > Elevator.CurrentFloor() {
		return "UP", nil
	} else if next.Floor < Elevator.CurrentFloor() && next.Floor != 0 {
		return "DOWN", nil
	} else if Elevator.ElevAtFloor() {
		return "IDLE", nil
	} else {
		return mystate, nil
	}
}
