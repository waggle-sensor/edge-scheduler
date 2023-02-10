package cloudscheduler

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type JobValidator struct {
	dataPath         string
	Plugins          map[string]*datatype.PluginManifest
	PluginsWhitelist map[string]bool
	Nodes            map[string]*datatype.NodeManifest
}

func NewJobValidator(dataPath string) *JobValidator {
	return &JobValidator{
		dataPath:         dataPath,
		Plugins:          make(map[string]*datatype.PluginManifest),
		PluginsWhitelist: make(map[string]bool),
		Nodes:            make(map[string]*datatype.NodeManifest),
	}
}

func (jv *JobValidator) GetNodeManifest(nodeName string) *datatype.NodeManifest {
	if n, exist := jv.Nodes[nodeName]; exist {
		return n
	} else {
		return nil
	}
}

func (jv *JobValidator) GetPluginManifest(pluginImage string) *datatype.PluginManifest {
	if p, exist := jv.Plugins[pluginImage]; exist {
		return p
	} else {
		return nil
	}
}

func (jv *JobValidator) LoadDatabase() error {
	nodeFiles, err := ioutil.ReadDir(path.Join(jv.dataPath, "nodes"))
	if err != nil {
		return err
	}
	for _, nodeFile := range nodeFiles {
		nodeFilePath := path.Join(jv.dataPath, "nodes", nodeFile.Name())
		raw, err := os.ReadFile(nodeFilePath)
		if err != nil {
			logger.Debug.Printf("Failed to read %s:%s", nodeFilePath, err.Error())
			continue
		}
		var n datatype.NodeManifest
		err = json.Unmarshal(raw, &n)
		if err != nil {
			logger.Debug.Printf("Failed to parse %s:%s", nodeFilePath, err.Error())
			continue
		}
		jv.Nodes[n.Name] = &n
	}
	pluginFiles, err := ioutil.ReadDir(path.Join(jv.dataPath, "plugins"))
	if err != nil {
		return err
	}
	for _, pluginFile := range pluginFiles {
		pluginFilePath := path.Join(jv.dataPath, "plugins", pluginFile.Name())
		raw, err := os.ReadFile(pluginFilePath)
		if err != nil {
			logger.Debug.Printf("Failed to read %s:%s", pluginFilePath, err.Error())
			continue
		}
		var p datatype.PluginManifest
		err = json.Unmarshal(raw, &p)
		if err != nil {
			logger.Debug.Printf("Failed to parse %s:%s", pluginFilePath, err.Error())
			continue
		}
		jv.Plugins[p.Image] = &p
	}
	return nil
}

func (jv *JobValidator) LoadPluginWhitelist() {
	whitelistFilePath := path.Join(jv.dataPath, "plugins.whitelist")
	if file, err := os.OpenFile(whitelistFilePath, os.O_CREATE|os.O_RDONLY, 0644); err == nil {
		fileScanner := bufio.NewScanner(file)
		for fileScanner.Scan() {
			jv.PluginsWhitelist[fileScanner.Text()] = true
		}
	} else {
		logger.Error.Printf("failed to create or open %q: %s", whitelistFilePath, err.Error())
	}
}

func (jv *JobValidator) AddPluginWhitelist(whitelist string) {
	jv.PluginsWhitelist[whitelist] = true
}

func (jv *JobValidator) RemovePluginWhitelist(whitelist string) {
	if _, found := jv.PluginsWhitelist[whitelist]; found {
		delete(jv.PluginsWhitelist, whitelist)
	}
}

func (jv *JobValidator) ListPluginWhitelist() (l []string) {
	for whitelist := range jv.PluginsWhitelist {
		l = append(l, whitelist)
	}
	return
}

func (jv *JobValidator) WritePluginWhitelist() {
	whitelistFilePath := path.Join(jv.dataPath, "plugins.whitelist")
	if file, err := os.OpenFile(whitelistFilePath, os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		for whitelist, _ := range jv.PluginsWhitelist {
			file.WriteString(whitelist + "\n")
		}
	} else {
		logger.Error.Printf("failed to create or open %q: %s", whitelistFilePath, err.Error())
	}
}

// GetNodeNamesByTags returns a list of node names matched with given tags
func (jv *JobValidator) GetNodeNamesByTags(tags []string) (nodesFound []string) {
	if len(tags) == 0 {
		return
	}
	for _, node := range jv.Nodes {
		if node.MatchTags(tags, true) {
			nodesFound = append(nodesFound, node.Name)
		}
	}
	return
}
