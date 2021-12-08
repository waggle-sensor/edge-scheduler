package pluginctl

import (
	"fmt"
	"os"

	"github.com/sagecontinuum/ses/pkg/logger"
	"github.com/sagecontinuum/ses/pkg/nodescheduler"
	"github.com/sagecontinuum/ses/pkg/runplugin"
)

type PluginCtl struct {
	ResourceManager *nodescheduler.ResourceManager
}

func NewPluginCtl(kubeconfig string) (*PluginCtl, error) {
	resourceManager, err := nodescheduler.NewK3SResourceManager("", false, kubeconfig, nil, false)
	if err != nil {
		return nil, err
	}
	resourceManager.Namespace = "default"
	return &PluginCtl{ResourceManager: resourceManager}, nil
}

func (p *PluginCtl) Deploy(spec *runplugin.Spec) error {
	sch := &runplugin.Scheduler{
		KubernetesClientset: p.ResourceManager.Clientset,
		RabbitMQClient:      nil,
	}
	return sch.RunPlugin(spec)
}

func (p *PluginCtl) PrintLog(pluginName string, follow bool) (func(), chan os.Signal, error) {
	podLog, err := p.ResourceManager.GetPodLog(pluginName, follow)
	if err != nil {
		return nil, nil, err
	}
	flag := make(chan os.Signal, 1)
	return func() {
		buf := make([]byte, 2000)
		for {
			select {
			case <-flag:
				logger.Debug.Printf("Log handler closed by func")
				podLog.Close()
				return
			default:
				numBytes, err := podLog.Read(buf)
				if numBytes == 0 {
					continue
				}
				if err != nil {
					logger.Debug.Printf("Log handler closed by error: %s", err.Error())
					podLog.Close()
					flag <- nil
					return
				}
				fmt.Println(string(buf[:numBytes]))
			}
		}
	}, flag, nil
}

func (p *PluginCtl) Terminate(pluginName string) error {
	return p.ResourceManager.TerminatePlugin(pluginName)
}
