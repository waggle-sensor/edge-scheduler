package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/zeromq/goczmq"
)

var (
	pathToPythonKB  = "kb.py"
	pathToIPCSocket = "/tmp/kb.sock"
)

type ZMQMessage struct {
	ReturnCode int         `json:"return_code"`
	Command    string      `json:"command"`
	Args       []string    `json:"args"`
	Result     interface{} `json:"result"`
}

func launchKB() {
	args := []string{pathToPythonKB}
	for {
		if _, err := os.Stat(pathToIPCSocket); os.IsExist(err) {
			err = os.Remove(pathToIPCSocket)
			if err != nil {
				fmt.Printf("file failed to remove: %s\n", err.Error())
			} else {
				fmt.Printf("file removed\n")
			}
		}
		fmt.Printf("Launching KB...")
		cmd := exec.Command("python3", args...)
		err := cmd.Run()
		if err != nil {
			fmt.Printf("kb.py failed with %s", err.Error())
		}
		fmt.Printf("Restarting kb.py in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}

func ping() (err error) {
	var socket *goczmq.Sock
	socket, err = goczmq.NewReq("ipc:///tmp/kb.sock")
	defer socket.Destroy()
	var request = ZMQMessage{
		Command: "ping",
	}
	byteJson, _ := json.Marshal(request)
	err = socket.SendFrame(byteJson, goczmq.FlagNone)
	if err != nil {
		return
	}
	reply, _, err := socket.RecvFrame()
	if err != nil {
		return
	}
	var response ZMQMessage
	err = json.Unmarshal(reply, &response)
	if err != nil {
		return
	}
	fmt.Printf("Pong received: %v\n", response)
	if response.Result == "pong" {
		err = nil
	}
	return
}

func main() {
	// go launchKB()
	for {
		if err := ping(); err == nil {
			fmt.Printf("Done\n")
			break
		} else {
			fmt.Printf("%s\n", err.Error())
		}
		time.Sleep(3 * time.Second)
	}

	testKB()

	// socket, err := goczmq.NewReq("ipc:///tmp/kb.sock")
	// defer socket.Destroy()
	// for {
	// 	var command string
	// 	fmt.Scanln(&command)
	//
	// 	var args = []string{"goal1", "hello"}
	//
	// 	var outMessage ZMQMessage
	// 	outMessage.Command = command
	// 	outMessage.Args = append(outMessage.Args, args...)
	// 	byteJson, _ := json.Marshal(outMessage)
	// 	err = socket.SendFrame(byteJson, goczmq.FlagNone)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	//
	// 	reply, _, err := socket.RecvFrame()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	var zmqMessage ZMQMessage
	// 	err = json.Unmarshal(reply, &zmqMessage)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	//
	// 	log.Printf("%s", string(reply))
	// 	log.Printf("%v", zmqMessage)
	// }

}

func send(socket *goczmq.Sock, msg *ZMQMessage) *ZMQMessage {
	byteJson, _ := json.Marshal(msg)
	err := socket.SendFrame(byteJson, goczmq.FlagNone)
	if err != nil {
		log.Fatal(err)
	}

	reply, _, err := socket.RecvFrame()
	if err != nil {
		log.Fatal(err)
	}
	var zmqMessage ZMQMessage
	_ = json.Unmarshal(reply, &zmqMessage)
	return &zmqMessage
}

func testKB() {
	socket, _ := goczmq.NewReq("ipc:///tmp/kb.sock")
	defer socket.Destroy()

	var addRule = ZMQMessage{
		Command: "rule",
		Args:    []string{"goal01", "Daytime(Now) ==> Run(Sampler)"},
	}
	rep := send(socket, &addRule)
	if rep.ReturnCode != 0 {
		log.Fatal(rep)
	}

	var addExpr = ZMQMessage{
		Command: "expr",
		Args:    []string{"goal01", "var > 10 ==> Daytime(Now)"},
	}
	rep = send(socket, &addExpr)
	if rep.ReturnCode != 0 {
		log.Fatal(rep)
	}

	var addMeasure = ZMQMessage{
		Command: "measure",
		Args:    []string{"var", string(time.Now().UnixNano()), "11"},
	}
	rep = send(socket, &addMeasure)
	if rep.ReturnCode != 0 {
		log.Fatal(rep)
	}

	var ask = ZMQMessage{
		Command: "ask",
		Args:    []string{"goal01", "Run(x)"},
	}
	rep = send(socket, &ask)
	if rep.ReturnCode != 0 {
		log.Fatal(rep)
	}
	fmt.Printf("%v", rep.Result)
}
