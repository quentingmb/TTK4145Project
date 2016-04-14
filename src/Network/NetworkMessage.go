package Network

import (
	"Elevator"
	
)

type Request struct {
 	Direction Elevator.Elev_button
 	Floor int
 	Type int //internal or external
 	Ipsource string 
 }
 type Info struct {
 	PreviousFloor int
 	Inhouse bool
 	State string 
 	Ipsource string
 }
 type ElevatorMessage struct {
 	Request Request
 	Info   Info
 }

 var NoRequest=[]Request{Request{}}
