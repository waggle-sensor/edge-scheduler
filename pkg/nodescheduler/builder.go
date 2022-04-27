package nodescheduler

import (
	"net/url"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/interfacing"
	"github.com/sagecontinuum/ses/pkg/nodescheduler/policy"
)

// NOTE: I do not know why we need this if we can't vary arguments in the functions
//       for multiple builders
// type NodeSchedulerBuilder interface {
// 	AddGoalManager() NodeSchedulerBuilder
// 	AddResourceManager() NodeSchedulerBuilder
// 	AddKnowledgebase() NodeSchedulerBuilder
// 	AddAPIServer() NodeSchedulerBuilder
// 	Build() *NodeScheduler
// }

type RealNodeScheduler struct {
	nodeScheduler *NodeScheduler
}

func NewRealNodeSchedulerBuilder(nodeID string, version string) *RealNodeScheduler {
	schedulingPolicy := policy.NewSimpleSchedulingPolicy()
	return &RealNodeScheduler{
		nodeScheduler: &NodeScheduler{
			Version:                     version,
			NodeID:                      nodeID,
			Simulate:                    false,
			SchedulingPolicy:            schedulingPolicy,
			chanContextEventToScheduler: make(chan datatype.EventPluginContext, maxChannelBuffer),
			chanFromGoalManager:         make(chan datatype.Event, maxChannelBuffer),
			chanFromResourceManager:     make(chan datatype.Event, maxChannelBuffer),
			chanRunGoal:                 make(chan *datatype.ScienceGoal, maxChannelBuffer),
			chanStopPlugin:              make(chan *datatype.Plugin, maxChannelBuffer),
			chanPluginToResourceManager: make(chan *datatype.Plugin, maxChannelBuffer),
			chanNeedScheduling:          make(chan datatype.Event, maxChannelBuffer),
			chanAPIServerToGoalManager:  make(chan *datatype.ScienceGoal, maxChannelBuffer),
		},
	}
}

func (rns *RealNodeScheduler) AddGoalManager(cloudschedulerURI string) *RealNodeScheduler {
	rns.nodeScheduler.GoalManager = &NodeGoalManager{
		ScienceGoals:          make(map[string]*datatype.ScienceGoal),
		cloudSchedulerBaseURL: cloudschedulerURI,
		NodeID:                rns.nodeScheduler.NodeID,
		chanGoalQueue:         make(chan *datatype.ScienceGoal, 100),
		Simulate:              false,
		Notifier:              interfacing.NewNotifier(),
	}
	rns.nodeScheduler.GoalManager.Notifier.Subscribe(rns.nodeScheduler.chanFromGoalManager)
	return rns
}

func (rns *RealNodeScheduler) AddResourceManager(registry string, incluster bool, kubeconfig string) *RealNodeScheduler {
	registryAddress, err := url.Parse(registry)
	if err != nil {
		panic(err)
	}
	k3sClient, err := GetK3SClient(incluster, kubeconfig)
	if err != nil {
		panic(err)
	}
	metricsClient, err := GetK3SMetricsClient(incluster, kubeconfig)
	if err != nil {
		panic(err)
	}
	rns.nodeScheduler.ResourceManager = &ResourceManager{
		Namespace:     "ses",
		ECRRegistry:   registryAddress,
		Clientset:     k3sClient,
		MetricsClient: metricsClient,
		Simulate:      false,
		Notifier:      interfacing.NewNotifier(),
	}
	rns.nodeScheduler.ResourceManager.Notifier.Subscribe(rns.nodeScheduler.chanFromResourceManager)
	return rns
}

func (rns *RealNodeScheduler) AddKnowledgebase(ruleCheckerURI string) *RealNodeScheduler {
	rns.nodeScheduler.Knowledgebase = NewKnowledgeBase(rns.nodeScheduler.NodeID, ruleCheckerURI)
	return rns
}

func (rns *RealNodeScheduler) AddAPIServer() *RealNodeScheduler {
	rns.nodeScheduler.APIServer = &APIServer{
		version:       rns.nodeScheduler.Version,
		nodeScheduler: rns.nodeScheduler,
	}
	return rns
}

func (rns *RealNodeScheduler) AddLoggerToBeehive(rabbitmqURI string, rabbitmqUsername string, rabbitmqPassword string, appID string) *RealNodeScheduler {
	rns.nodeScheduler.LogToBeehive = interfacing.NewRabbitMQHandler(rabbitmqURI, rabbitmqUsername, rabbitmqPassword, appID)
	return rns
}

func (rns *RealNodeScheduler) Build() *NodeScheduler {
	return rns.nodeScheduler
}

type FakeNodeScheduler struct {
	nodeScheduler *NodeScheduler
}

func NewFakeNodeSchedulerBuilder(nodeID string, version string) *FakeNodeScheduler {
	schedulingPolicy := policy.NewSimpleSchedulingPolicy()
	return &FakeNodeScheduler{
		nodeScheduler: &NodeScheduler{
			Version:                     version,
			NodeID:                      nodeID,
			Simulate:                    true,
			SchedulingPolicy:            schedulingPolicy,
			chanContextEventToScheduler: make(chan datatype.EventPluginContext, maxChannelBuffer),
			chanFromGoalManager:         make(chan datatype.Event, maxChannelBuffer),
			chanFromResourceManager:     make(chan datatype.Event, maxChannelBuffer),
			chanRunGoal:                 make(chan *datatype.ScienceGoal, maxChannelBuffer),
			chanStopPlugin:              make(chan *datatype.Plugin, maxChannelBuffer),
			chanPluginToResourceManager: make(chan *datatype.Plugin, maxChannelBuffer),
			chanNeedScheduling:          make(chan datatype.Event, maxChannelBuffer),
			chanAPIServerToGoalManager:  make(chan *datatype.ScienceGoal, maxChannelBuffer),
		},
	}
}

func (fns *FakeNodeScheduler) AddGoalManager() *FakeNodeScheduler {
	fns.nodeScheduler.GoalManager = &NodeGoalManager{
		ScienceGoals:          make(map[string]*datatype.ScienceGoal),
		cloudSchedulerBaseURL: "",
		NodeID:                fns.nodeScheduler.NodeID,
		chanGoalQueue:         make(chan *datatype.ScienceGoal, 100),
		Simulate:              true,
		Notifier:              interfacing.NewNotifier(),
	}
	fns.nodeScheduler.GoalManager.Notifier.Subscribe(fns.nodeScheduler.chanFromGoalManager)
	return fns
}

func (fns *FakeNodeScheduler) AddResourceManager() *FakeNodeScheduler {
	fns.nodeScheduler.ResourceManager = &ResourceManager{
		Namespace:     "ses",
		ECRRegistry:   nil,
		Clientset:     nil,
		MetricsClient: nil,
		Simulate:      true,
		Notifier:      interfacing.NewNotifier(),
	}
	fns.nodeScheduler.ResourceManager.Notifier.Subscribe(fns.nodeScheduler.chanFromResourceManager)
	return fns
}

func (fns *FakeNodeScheduler) AddKnowledgebase() *FakeNodeScheduler {
	fns.nodeScheduler.Knowledgebase = NewKnowledgeBase(fns.nodeScheduler.NodeID, "")
	return fns
}

func (fns *FakeNodeScheduler) AddAPIServer() *FakeNodeScheduler {
	fns.nodeScheduler.APIServer = &APIServer{
		version:       fns.nodeScheduler.Version,
		nodeScheduler: fns.nodeScheduler,
	}
	return fns
}

func (fns *FakeNodeScheduler) Build() *NodeScheduler {
	return fns.nodeScheduler
}
