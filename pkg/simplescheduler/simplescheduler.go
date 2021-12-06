package simplescheduler

import (
	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
)

type SimpleScheduler struct {
	ResourceManager *nodescheduler.ResourceManager
}

func NewSimpleScheduler(rm *nodescheduler.ResourceManager) *SimpleScheduler {
	return &SimpleScheduler{
		ResourceManager: rm,
	}
}

func (ss *SimpleScheduler) LoadPluginsFromConfigMap() {

}

// Configure sets up the followings in Kubernetes cluster
//
// - "ses" namespace
//
// - "wes-rabbitmq" service at port 5672 in the ses namespace
func (ss *SimpleScheduler) Configure() error {
	err := ss.ResourceManager.CreateNamespace("ses")
	if err != nil {
		return err
	}
	err = ss.ResourceManager.ForwardService("wes-rabbitmq", "default", "ses")
	if err != nil {
		return err
	}
	return nil
}

func (ss *SimpleScheduler) BringUpServices() {
	// need to set up wes services in ses namespace
	// for plugins to use them in the namespace
}

func (ss *SimpleScheduler) Run() {
	logger.Info.Println("Simple scheduler starts")

	logger.Info.Println("Simple scheduler exits")
}
