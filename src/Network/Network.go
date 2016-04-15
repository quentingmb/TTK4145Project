package Network

import (
	"Elevator"
	"driver"
	//"encoding/json"
	"extra"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
	//"io/ioutil"
	//"os/exec"
)

var requestlist = make([]Request, 0)
var elevators = make(map[string]bool)
var infolist = make(map[string]Info)
var tcp_connections = make([]*net.TCPConn, 0)

func GetInfoList() map[string]Info {
	return infolist
}

func GetRequestList() []Request {
	return requestlist
}

func InitUpdate(connection *net.TCPConn, localip string, Error chan string) {
	pack := make([]byte, 1024)
	info := infolist[localip]
	pack = PackElevatorMessage(ElevatorMessage{Request: Request{},
		Info: info}, Error)
	time.Sleep(10 * time.Millisecond)
	connection.Write(pack)
	for _, request := range requestlist {
		time.Sleep(10 * time.Millisecond)
		pack = PackElevatorMessage(ElevatorMessage{Request: request,
			Info: Info{}}, Error)
		connection.Write(pack)
	}
}

//Check if there is any new requests, if it is, it passes it to NewRequest
func RequestUpdate(ToSendMsgs chan ElevatorMessage, localip string) {
	var button Elevator.Elev_button
	for {
		for floor, buttons := range Elevator.Button_channel_matrix {
			for butt, channel := range buttons {
				if driver.Read_bit(channel) != 0 {
					if butt == 0 {
						button = Elevator.BUTTON_CALL_UP
					} else if butt == 1 {
						button = Elevator.BUTTON_CALL_DOWN
					} else {
						button = Elevator.BUTTON_COMMAND
					}
					NewRequest(ToSendMsgs, Request{Direction: button, Floor: floor + 1, Type: 1, Ipsource: localip})
					time.Sleep(time.Millisecond)
				}
			}
		}
	}
}

func NetworkManager(conf extra.Config, localip string, ToSendMsgs chan ElevatorMessage) {
	elevlog, err := os.OpenFile("elevator.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening file: " + err.Error())
	}
	defer elevlog.Close()
	log.SetOutput(elevlog)
	listenAdress, _ := net.ResolveTCPAddr("tcp", conf.DefaultListenPort)
	listenConnection, _ := net.ListenTCP("tcp", listenAdress)
	connections := make(chan Connection, 15)
	receivedMsgs := make(chan ElevatorMessage, 15)
	Error := make(chan string, 10)
	go Listener(listenConnection, connections, Error)
	go RequestUpdate(ToSendMsgs, localip)
	go New_Connection(connections, conf.DefaultListenPort, conf.Elevators, Error)
	for {
		select {
		case connection := <-connections: //Managing new/closed connections
			{
				if connection.Connected {
					tcp_connections = append(tcp_connections, connection.Addr)
					go Receiver(connection.Addr, receivedMsgs, connections, Error)
					go AliveNotification(connection.Addr, Error)
					go InitUpdate(connection.Addr, localip, Error)
				} else {
					remoteip := strings.Split(connection.Addr.RemoteAddr().String(), ":")[0]
					errorstate := Info{State: "ERROR", PreviousFloor: 0, Ipsource: remoteip}
					infolist[remoteip] = errorstate
					for i, conn := range tcp_connections {
						if conn == connection.Addr {
							tcp_connections[len(tcp_connections)-1], tcp_connections[i], tcp_connections = nil, tcp_connections[len(tcp_connections)-1], tcp_connections[:len(connections)-1]
						}
					}
					connection.Addr.Close()
				}

			}
		case ReceivedMsg := <-receivedMsgs:
			{
				if ReceivedMsg.Request.Floor > 0 {
					if !((ReceivedMsg.Request.Direction == Elevator.BUTTON_COMMAND) && (ReceivedMsg.Request.Ipsource != localip)) {
						Elevator.SetElevButtonLamp(ReceivedMsg.Request.Direction, ReceivedMsg.Request.Floor, ReceivedMsg.Request.Type)
					}
					if ReceivedMsg.Request.Direction != Elevator.BUTTON_COMMAND {
						ReceivedMsg.Request.Ipsource = ""
					}
					if ReceivedMsg.Request.Type == 0 {
						ReceivedMsg.Request.Type = 1
						for i, b := range requestlist {
							if b == ReceivedMsg.Request {
								requestlist = append(requestlist[:i], requestlist[i+1:]...)
							}
						}
					} else {
						AlreadyExists := false
						for _, b := range requestlist {
							if b == ReceivedMsg.Request {
								AlreadyExists = true
							}
						}
						if !AlreadyExists {
							requestlist = append(requestlist, ReceivedMsg.Request)
						}
					}
				}
				if ReceivedMsg.Info.Ipsource != "" {
					infolist[ReceivedMsg.Info.Ipsource] = ReceivedMsg.Info
				}
			}
		case messageToSend := <-ToSendMsgs:
			{
				pack := make([]byte, 1024)
				pack = PackElevatorMessage(messageToSend, Error)
				for _, connection := range tcp_connections {
					_, err := connection.Write(pack)
					if err != nil {
						Error <- "Problems writing to connection: " + err.Error()
					}
				}
			}
		case err := <-Error:
			{
				log.Println("ERROR: " + err)
			}
		}
	}
}


func NewInfo(info Info, ToSendMsgs chan ElevatorMessage) bool {
	for _, oldinfo := range infolist {
		if oldinfo == info {
			return false
		}
	}
	ToSendMsgs <- ElevatorMessage{Request: Request{}, Info: info}
	return true
}

func NewRequest(ToSendMsgs chan ElevatorMessage, request Request) bool {
	if request.Direction != Elevator.BUTTON_COMMAND {
		request.Ipsource = ""
	}
	for _, r := range requestlist {
		if r == request {
			return false
		}
	}
	ToSendMsgs <- ElevatorMessage{Request: request, Info: Info{}}
	return true
}
