package Network

import (
	"driver"
	"Elevator"
	"encoding/json"
	"fmt"
	"log"
	"extra"
	"net"
	"os"
	"strings"
	"time"
	//"io/ioutil"
	"os/exec"
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

 type Con struct {
 	Address *net.TCPConn
	Connected bool
 }
var NoRequest=[]Request{Request{}}
var requestlist=make([]Request,0)
var elevators = make(map[string]bool)
var infolist = make(map[string]Info)
var connections = make([]*net.TCPConn, 0)

func GetLocalIP() string {
	oneliner := "ifconfig | grep 129.241.187 | cut -d':' -f2 | cut -d' ' -f1" //Favourite Oneliner
	cmd := exec.Command("bash", "-c", oneliner)
	out, err := cmd.Output()
	if err != nil {
		fmt.Println(err)
	}
	ip := strings.TrimSpace(string(out))
	return ip
}

func GetInfoList() map[string]Info {
	return infolist
}

func GetRequestList() []Request {
	return requestlist
}

func PackElevatorMessage(message ElevatorMessage, error_c chan string) []byte {
	send, err := json.Marshal(message)
	if err != nil {
		error_c <- "Could not pack message: " + err.Error()
	}
	return send
}

func UnpackElevatorMessage(packed []byte, error_c chan string) ElevatorMessage {
	var message ElevatorMessage
	err := json.Unmarshal(packed, &message)
	if err != nil {
		error_c <- "Error in unpacking the message: " + err.Error()
	}
	return message
}

func InitUpdate(connection *net.TCPConn, myip string, error_c chan string) {
	pack := make([]byte, 1024)
	info := infolist[myip]
	pack = PackElevatorMessage(ElevatorMessage{Request: Request{}, 
	Info: info}, error_c)
	time.Sleep(10 * time.Millisecond)
	connection.Write(pack)
	for _, request := range requestlist {
		time.Sleep(10 * time.Millisecond)
		pack = PackElevatorMessage(ElevatorMessage{Request: request, 
		Info: Info{}}, error_c)
		connection.Write(pack)
	}
}

//Check if there is any new orders, if it is it passes it to Neworder
func Requestdistr(generatedMsgs_c chan ElevatorMessage, myip string) {
	var button Elevator.Elev_button
	for {
		for floor, buttons := range Elevator.Button_channel_matrix {
			for butt, channel := range buttons {
				if driver.Read_bit(channel)!=0 {
					if butt == 0 {
						button = Elevator.BUTTON_CALL_UP
					} else if butt == 1 {
						button = Elevator.BUTTON_CALL_DOWN
					} else {
						button = Elevator.BUTTON_COMMAND
					}
					NewRequest(generatedMsgs_c, Request{Direction: button, Floor: floor + 1, Type: 1, Ipsource: myip})
					time.Sleep(time.Millisecond)
				}
			}
		}
	}
}

func Dialer(connect_c chan Con, port string, elevators []extra.Elevator, error_c chan string) {
	local, _ := net.ResolveTCPAddr("tcp", "localhost"+port)
	localconn, _ := net.DialTCP("tcp", nil, local)
	connect_c <- Con{Address: localconn, Connected: true}
	for {
	elevatorloop:
		for _, elevator := range elevators {
			cons := connections
			for _, connection := range cons {
				if strings.Split(connection.RemoteAddr().String(), ":")[0] == elevator.Address {
					continue elevatorloop
				}
			}
			raddr, err := net.ResolveTCPAddr("tcp", elevator.Address+port)
			dialConn, err := net.DialTCP("tcp", nil, raddr)
			if err != nil {
				error_c <- "Dial trouble: " + err.Error()
			} else {
				connect_c <- Con{Address: dialConn, Connected: true}
			}
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func Listener(conn *net.TCPListener, connect_c chan Con, error_c chan string) {
	for {
		newConn, err := conn.AcceptTCP()
		if err != nil {
			error_c <- "Accept trouble: " + err.Error()
		}
		connect_c <- Con{Address: newConn, Connected: true}
	}
}

func Receiver(conn *net.TCPConn, receivedMsgs_c chan ElevatorMessage, connections_c chan Con, error_c chan string) {
	buf := make([]byte, 1024)
	keepalivebyte := []byte("KEEPALIVE")
receiverloop:
	for {
		err := conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
			error_c <- "Trouble setting read deadline: " + err.Error()
			connections_c <- Con{Address: conn, Connected: false}
			return
		}
		bit, err := conn.Read(buf[0:])
		if err != nil {
			error_c <- "Trouble receiving: " + err.Error()
			connections_c <- Con{Address: conn, Connected: false}
			return
		}
		if string(buf[:bit]) == string(keepalivebyte) {
			continue receiverloop
		}
		unpacked := UnpackElevatorMessage(buf[:bit], error_c)
		receivedMsgs_c <- unpacked
	}
}

func SendAliveMessages(connection *net.TCPConn, error_c chan string) {
	for {
		_, err := connection.Write([]byte("KEEPALIVE"))
		if err != nil {
			error_c <- "error in sending keepalive message: " + err.Error()
			return
		}
		time.Sleep(time.Second)
	}
}

func TCPPeerToPeer(conf extra.Config, myip string, generatedmessages_c chan ElevatorMessage) {
	elevlog, err := os.OpenFile("elevator.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening file: " + err.Error())
	}
	defer elevlog.Close()
	log.SetOutput(elevlog)
	listenaddr, _ := net.ResolveTCPAddr("tcp", conf.DefaultListenPort)
	listenconn, _ := net.ListenTCP("tcp", listenaddr)
	connections_c := make(chan Con, 15)
	receivedmessages_c := make(chan ElevatorMessage, 15)
	error_c := make(chan string, 10)
	go Listener(listenconn, connections_c, error_c)
	go Requestdistr(generatedmessages_c, myip)
	go Dialer(connections_c, conf.DefaultListenPort, conf.Elevators, error_c)
	for {
		select {
		case connection := <-connections_c: //Managing new/closed connections
			{
				if connection.Connected {
					connections = append(connections, connection.Address)
					go Receiver(connection.Address, receivedmessages_c, connections_c, error_c)
					go SendAliveMessages(connection.Address, error_c)
					go InitUpdate(connection.Address, myip, error_c)
				} else {
					remoteip := strings.Split(connection.Address.RemoteAddr().String(), ":")[0]
					errorstate := Info{State: "ERROR", PreviousFloor: 0, Inhouse: false, Ipsource: remoteip}
					infolist[remoteip] = errorstate
					for i, con := range connections {
						if con == connection.Address {
							connections[len(connections)-1], connections[i], connections = nil, connections[len(connections)-1], connections[:len(connections)-1]
						}
					}
					connection.Address.Close()
				}

			}
		case received := <-receivedmessages_c:
			{
				if received.Request.Floor > 0 {
					if !((received.Request.Direction == Elevator.BUTTON_COMMAND) && (received.Request.Ipsource != myip)) {
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
		case message := <-generatedmessages_c:
			{
				pack := make([]byte, 1024)
				pack = PackElevatorMessage(message, error_c)
				for _, connection := range connections {
					_, err := connection.Write(pack)
					if err != nil {
						error_c <- "Problems writing to connection: " + err.Error()
					}
				}
			}
		case err := <-error_c:
			{
				log.Println("ERROR: " + err)
			}
		}
	}
}

func SendStatuslist(generatedMsgs_c chan ElevatorMessage) {
	myip := GetLocalIP()
	myinfo := infolist[myip]
	generatedMsgs_c <- ElevatorMessage{Request: Request{}, Info: myinfo}
}

func NewInfo(info Info, generatedMsgs_c chan ElevatorMessage) bool {
	for _, oldinfo := range infolist {
		if oldinfo == info {
			return false
		}
	}
	generatedMsgs_c <- ElevatorMessage{Request: Request{}, Info: info}
	return true
}

func NewRequest(generatedMsgs_c chan ElevatorMessage, request Request) bool {
	if request.Direction != Elevator.BUTTON_COMMAND {
		request.Ipsource = ""
	}
	for _, r := range requestlist {
		if r == request {
			return false
		}
	}
	generatedMsgs_c <- ElevatorMessage{Request: request, Info: Info{}}
	return true
}

