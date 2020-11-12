package knowledgebase

import (
	"os/exec"
	"strings"
	"time"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/zeromq/goczmq"
)

var (
	pathToPythonKB = "kb.py"
	chanToKB       = make(chan Event)
)

// InitializeKB launches the KB engine and keeps it alive.
// It also receives events from the engine
func InitializeKB() {
	go launchKB()
	go runIPCToKB()
	go runEventReceiver()
}

// RegisterRules registers rules of a goal to the KB engine
func RegisterRules(scienceGoal *datatype.ScienceGoal, nodeName string) {
	mySubGoal := scienceGoal.GetMySubGoal(nodeName)

	logger.Info.Printf("Loading rules to KB...")
	rules := []string{scienceGoal.ID}
	rules = append(rules, mySubGoal.Rules...)
	chanToKB <- datatype.Message {
		datatype.RequestToKB {
			Command: "rule",
			Args: rules,
		}
	}

	logger.Info.Printf("Loading statements to KB...")
	statements := []string{scienceGoal.ID}
	statements = append(statements, mySubGoal.Statements...)
	chanToKB <- datatype.Message {
		datatype.RequestToKB {
			Command: "state",
			Args: statements,
		}
	}
}

func runIPCToKB() {
	for {
		chanExit := make(chan error)
		go func() {
			socket, err := goczmq.NewReq("ipc:///tmp/kb.sock")
			if err != nil {
				chanExit <- err
				return
			}
			defer socket.Destroy()
			for {
				request := <-chanToKB
				byteJson, _ := json.Marshal(request)
				err = socket.SendFrame(byteJson, goczmq.FlagNone)
				_, _, err = socket.RecvFrame()
				if err != nil {
					chanExit <- err
					return
				}
			}
		}()
		err := <-chanExit
		logger.Error.Printf("IPC to KB failed: %s", err)
		time.Sleep(3 * time.Second)
	}
}

// runEvenReceiver receives Run, Stop events of plugins
// from knowledgebase
func runEventReceiver() {
	for {
		chanExit := make(chan error)
		go func() {
			logger.Info.Printf("(re)Starting evnet listener...")
			socket, err := goczmq.NewPair("ipc:///tmp/event.sock")
			if err != nil {
				chanExit <- err
				return
			}
			defer socket.Destroy()
			for {
				message, _, err := socket.RecvFrame()
				if err != nil {
					chanExit <- err
					return
				}
				// message format must be "Runnable|Stoppable Plugin"
				sp := strings.Split(message, " ")

				chanToGoalManager <- datatype.Message{
					datatype.EventPluginContext{
						Name:   sp[1],
						Status: sp[0],
					},
				}
				logger.Info.Printf("Event received: %s", message)
			}
		}()
		err := <-chanExit
		logger.Error.Printf("Event receiver failed: %s", err)
		time.Sleep(3 * time.Second)
	}
	// socket, err := goczmq.NewPair("ipc:///tmp/kb.sock")

	// chanEventToManager <-
}

func launchKB() {
	args := []string{pathToPythonKB}
	for {
		// if _, err := os.Stat(pathToIPCSocket); os.IsExist(err) {
		// 	err = os.Remove(pathToIPCSocket)
		// 	if err != nil {
		// 		fmt.Printf("file failed to remove: %s\n", err.Error())
		// 	} else {
		// 		fmt.Printf("file removed\n")
		// 	}
		// }
		logger.Info.Printf("Launching KB...")
		cmd := exec.Command("python3", args...)
		err := cmd.Run()
		if err != nil {
			logger.Info.Printf("kb.py failed with %s", err.Error())
		}
		logger.Info.Printf("Restarting kb.py in 5 seconds...")
		time.Sleep(5 * time.Second)
	}
}
