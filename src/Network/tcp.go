package Network

import (
	//"driver"
	//"Elevator"
	"encoding/json"
	"fmt"
	//"log"
	"extra"
	"net"
	//"os"
	"strings"
	"time"
	//"io/ioutil"
	"os/exec"
)

type Connection struct {
	Addr      *net.TCPConn
	Connected bool
}

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

func PackElevatorMessage(message ElevatorMessage, Error chan string) []byte {
	pack, err := json.Marshal(message)
	if err != nil {
		Error <- "Error in packing the message: " + err.Error()
	}
	return pack
}

func UnpackElevatorMessage(packedMessage []byte, Error chan string) ElevatorMessage {
	var msg ElevatorMessage
	err := json.Unmarshal(packedMessage, &msg)
	if err != nil {
		Error <- "Error in unpacking the message: " + err.Error()
	}
	return msg
}

func New_Connection(Connect chan Connection, port string, elevators []extra.Elevator, Error chan string) {
	localAdress, _ := net.ResolveTCPAddr("tcp", "localhost"+port)
	localconn, _ := net.DialTCP("tcp", nil, localAdress)
	Connect <- Connection{Addr: localconn, Connected: true}
	for {
	elevloop:
		for _, elev := range elevators {
			cons := tcp_connections
			for _, connection := range cons {
				if strings.Split(connection.RemoteAddr().String(), ":")[0] == elev.Address {
					continue elevloop
				}
			}
			r_addr, err := net.ResolveTCPAddr("tcp", elev.Address+port)
			dialConn, err := net.DialTCP("tcp", nil, r_addr)
			if err != nil {
				Error <- "Could not dial: " + err.Error()
			} else {
				Connect <- Connection{Addr: dialConn, Connected: true}
			}
		}
		time.Sleep(1000 * time.Millisecond)
	}
}

func Listener(connection *net.TCPListener, Connect chan Connection, Error chan string) {
	for {
		newTcpConn, err := connection.AcceptTCP()
		if err != nil {
			Error <- "Could not accept: " + err.Error()
		}
		Connect <- Connection{Addr: newTcpConn, Connected: true}
	}
}

func Receiver(connection *net.TCPConn, receivedMsgs chan ElevatorMessage, connections chan Connection, Error chan string) {
	buffer := make([]byte, 1024)
	keepalivebyte := []byte("KEEPALIVE")
receiverloop:
	for {
		err := connection.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err != nil {
			Error <- "Could not set read deadline: " + err.Error()
			connections <- Connection{Addr: connection, Connected: false}
			return
		}
		bit, err := connection.Read(buffer[0:])
		if err != nil {
			Error <- " receiving Problem: " + err.Error()
			connections <- Connection{Addr: connection, Connected: false}
			return
		}
		if string(buffer[:bit]) == string(keepalivebyte) {
			continue receiverloop
		}
		unpackedMsg := UnpackElevatorMessage(buffer[:bit], Error)
		receivedMsgs <- unpackedMsg
	}
}

func AliveNotification(connection *net.TCPConn, Error chan string) {
	for {
		_, err := connection.Write([]byte("KEEPALIVE"))
		if err != nil {
			Error <- "Could not send keepalive message: " + err.Error()
			return
		}
		time.Sleep(time.Second)
	}
}
