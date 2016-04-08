
package ElevatorLogic

import (
	"Elevator"
	"extra"
	"Network"
)
//This function does a BFS-search through all orders to find the most effective solution
func Nextrequest(myip string, Elevatorlist []extra.Elevator) Network.Request {
	var statelist = make(map[string]Network.Info)
	infolist := Network.GetInfoList()
	for host, info := range infolist {
		statelist[host] = info
	}
	requestlist := Network.GetRequestList()
insideloop:
	for _, request := range requestlist {
		if request.Direction != Elevator.BUTTON_COMMAND {
			continue insideloop
		}
		for _, elevator := range Elevatorlist {
			if info, ok := statelist[elevator.Address]; ok {
				if ((info.State == "UP" || info.State == "IDLE") && info.PreviousFloor <= request.Floor) || ((info.State == "DOWN" || info.State == "IDLE") && info.PreviousFloor >= request.Floor) {
					if info.Ipsource == request.Ipsource {
						if info.Ipsource == myip {
							return request
						} else {
							delete(statelist, elevator.Address)
							continue insideloop
						}
					}
				}
			}
		}
		for _, elevator := range Elevatorlist {
			if info, ok := statelist[elevator.Address]; ok {
				if (info.State == "UP" && info.PreviousFloor >= request.Floor) || (info.State == "DOWN" && info.PreviousFloor <= request.Floor){
					if info.Ipsource == request.Ipsource {
						if info.Ipsource == myip {
							return request
						} else {
							delete(statelist, elevator.Address)
							continue insideloop
						}
					}
				}
			}
		}
	}
requestloop:
	for _, request := range requestlist {
		if request.Direction == Elevator.BUTTON_COMMAND {
			continue requestloop
		}
		for i := 0; i < 4; i++ {    // N_FLOORS ca marche pas
			for _, elevator := range Elevatorlist {
				if info, ok := statelist[elevator.Address]; ok {
					if i != 0 && (info.State == "UP" && info.PreviousFloor+i == request.Floor) || (info.State == "DOWN" && info.PreviousFloor-i == request.Floor) {
						if statelist[elevator.Address].Ipsource == myip {
							return request
						} else {
							delete(statelist, elevator.Address)
							continue requestloop
						}
					}
				}
			}
			for _, elevator := range Elevatorlist {
				if info, ok := statelist[elevator.Address]; ok {
					if info.State == "IDLE" && (info.PreviousFloor == request.Floor+i || info.PreviousFloor == request.Floor-i) {
						if statelist[elevator.Address].Ipsource == myip {
							return request
						} else {
							delete(statelist, elevator.Address)
							continue requestloop
						}
					}
				}
			}
		}
	}
	return Network.NoRequest[0]
}

//This function return orders the elevator should stop for
func Stop(myip string, mystate string) []Network.Request {
	var takerequest []Network.Request
	requestlist := Network.GetRequestList()
	for _, request := range requestlist {
		if (request.Direction == Elevator.BUTTON_COMMAND && request.Ipsource == myip) || (request.Direction == Elevator.BUTTON_CALL_UP && mystate == "UP") || (request.Direction == Elevator.BUTTON_CALL_DOWN && mystate == "DOWN") {
			if request.Floor == Elevator.CurrentFloor() && Elevator.ElevAtFloor() {
				takerequest = append(takerequest, request)
			}
		}
	}
	return takerequest
}
//This function returns the next state for the elevator
func Nextstate(myip string, elevators []extra.Elevator, mystate string) (string, []Network.Request) {
	if Elevator.GetElevObstructionSignal() {
		Elevator.SetElevStopLamp(1)
		return "ERROR", nil
	} else if mystate == "ERROR" {
		Elevator.SetElevStopLamp(0)
		return "INIT", nil
	}

	stop := Stop(myip, mystate)
	if len(stop) != 0 {
		return "DOOR_OPEN", stop
	}

	next := Nextrequest(myip, elevators)
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
