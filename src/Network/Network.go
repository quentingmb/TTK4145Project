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

//Check if there is any new orders, if it is it passes it to Neworder
func RequestChecker(generatedMsgs chan ElevatorMessage, localip string) {
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
					NewRequest(generatedMsgs, Request{Direction: button, Floor: floor + 1, Type: 1, Ipsource: localip})
					time.Sleep(time.Millisecond)
				}
			}
		}
	}
}

func NetworkManager(conf extra.Config, localip string, generatedMsgs chan ElevatorMessage) {
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
	go RequestChecker(generatedMsgs, localip)
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
					errorstate := Info{State: "ERROR", PreviousFloor: 0, Inhouse: false, Ipsource: remoteip}
					infolist[remoteip] = errorstate
					for i, conn := range tcp_connections {
						if conn == connection.Addr {
							tcp_connections[len(tcp_connections)-1], tcp_connections[i], tcp_connections = nil, tcp_connections[len(tcp_connections)-1], tcp_connections[:len(connections)-1]
						}
					}
					connection.Addr.Close()
				}

			}
		case received := <-receivedMsgs:
			{
				if received.Request.Floor > 0 {
					if !((received.Request.Direction == Elevator.BUTTON_COMMAND) && (received.Request.Ipsource != localip)) {
						Elevator.SetElevButtonLamp(received.Request.Direction, received.Request.Floor, received.Request.Type)
					}
					if received.Request.Direction != Elevator.BUTTON_COMMAND {
						received.Request.Ipsource = ""
					}
					if received.Request.Type == 0 {
						received.Request.Type = 1
						for i, b := range requestlist {
							if b == received.Request {
								requestlist = append(requestlist[:i], requestlist[i+1:]...)
							}
						}
					} else {
						AddedBefore := false
						for _, b := range requestlist {
							if b == received.Request {
								AddedBefore = true
							}
						}
						if !AddedBefore {
							requestlist = append(requestlist, received.Request)
						}
					}
				}
				if received.Info.Ipsource != "" {
					infolist[received.Info.Ipsource] = received.Info
				}
			}
		case message := <-generatedMsgs:
			{
				pack := make([]byte, 1024)
				pack = PackElevatorMessage(message, Error)
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

func SendStatuslist(generatedMsgs chan ElevatorMessage) {
	localip := GetLocalIP()
	myinfo := infolist[localip]
	generatedMsgs <- ElevatorMessage{Request: Request{}, Info: myinfo}
}

func NewInfo(info Info, generatedMsgs chan ElevatorMessage) bool {
	for _, oldinfo := range infolist {
		if oldinfo == info {
			return false
		}
	}
	generatedMsgs <- ElevatorMessage{Request: Request{}, Info: info}
	return true
}

func NewRequest(generatedMsgs chan ElevatorMessage, request Request) bool {
	if request.Direction != Elevator.BUTTON_COMMAND {
		request.Ipsource = ""
	}
	for _, r := range requestlist {
		if r == request {
			return false
		}
	}
	generatedMsgs <- ElevatorMessage{Request: request, Info: Info{}}
	return true
}
