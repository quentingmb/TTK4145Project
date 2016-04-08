package extra

import (
"encoding/json"
	//"fmt"
	"io/ioutil"
	"log"
	//"os/exec"
	//"strings"
	)

type Elevator struct {
	Address string
}

type Config struct {
	Elevators         []Elevator
	DefaultListenPort string
	Timeout           int
	NumFloors         int
	StopReverseTime   int
}

var config Config

func LoadConfig(filename string) Config {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println(err)
	}
	err = json.Unmarshal(file, &config)
	if err != nil {
		log.Println(err)
	}
	return config
}
