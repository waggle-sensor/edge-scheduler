package cloudscheduler

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/sagecontinuum/ses/pkg/datatype"
	"github.com/sagecontinuum/ses/pkg/logger"
	yaml "gopkg.in/yaml.v2"
)

// MetaHandler structs nodes and plugins with their meta information
type MetaHandler struct {
	nodes   []datatype.Node
	plugins []datatype.Plugin
}

// NewMetaHandler returns an instance of meta handler
func NewMetaHandler(dataDir string) (*MetaHandler, error) {
	loadedNodes, err := getNodesFromDirectory(fmt.Sprint(dataDir, "/nodes"))
	if err != nil {
		logger.Error.Printf("Failed to load nodes:%s", err.Error())
	}
	loadedPlugins, err := getPluginsFromDirectory(fmt.Sprint(dataDir, "/plugins"))
	if err != nil {
		logger.Error.Printf("Failed to load plugins: %s", err.Error())
	}
	return &MetaHandler{
		nodes:   loadedNodes,
		plugins: loadedPlugins,
	}, nil
}

// GetPlugin returns the plugin of given name and version
func (mh *MetaHandler) GetPlugin(name string, version string) (datatype.Plugin, error) {
	for _, plugin := range mh.plugins {
		if plugin.Name == name &&
			plugin.PluginSpec.Version == version {
			return plugin, nil
		}
	}
	return datatype.Plugin{}, fmt.Errorf("could not find plugin %s:%s", name, version)
}

// GetNodesByTags returns a list of nodes matched with given tags
func (mh *MetaHandler) GetNodesByTags(tags []string) (nodesFound []datatype.Node) {
	for _, node := range mh.nodes {
		for _, tag := range tags {
			for _, nodeTag := range node.Tags {
				if tag == nodeTag {
					exists := nodeExistInArray(node, nodesFound)
					if !exists {
						nodesFound = append(nodesFound, node)
					}
					break
				}
			}
		}
	}
	return
}

// GetPluginsByTags returns a list of plugins matched with given tags
func (mh *MetaHandler) GetPluginsByTags(tags []string) (pluginsFound []datatype.Plugin) {
	for _, plugin := range mh.plugins {
		for _, tag := range tags {
			for _, pluginTag := range plugin.Tags {
				if tag == pluginTag {
					exists := pluginExistInArray(plugin, pluginsFound)
					if !exists {
						pluginsFound = append(pluginsFound, plugin)
					}
					break
				}
			}
		}
	}
	return
}

func getNodesFromDirectory(path string) (nodes []datatype.Node, err error) {
	nodeFiles, _ := filepath.Glob(fmt.Sprint(path, "/*.yaml"))
	for _, filePath := range nodeFiles {
		dat, _ := ioutil.ReadFile(filePath)
		var node datatype.Node
		_ = yaml.Unmarshal(dat, &node)
		nodes = append(nodes, node)
		logger.Debug.Printf("Node %s is loaded from %s", node.Name, filePath)
	}
	return
}

func getPluginsFromDirectory(path string) (plugins []datatype.Plugin, err error) {
	nodeFiles, _ := filepath.Glob(fmt.Sprint(path, "/*.yaml"))
	for _, filePath := range nodeFiles {
		dat, _ := ioutil.ReadFile(filePath)
		var plugin datatype.Plugin
		_ = yaml.Unmarshal(dat, &plugin)
		plugins = append(plugins, plugin)
		logger.Debug.Printf("Plugin %s is loaded from %s", plugin.Name, filePath)
	}
	return
}

func nodeExistInArray(node datatype.Node, nodes []datatype.Node) bool {
	for _, nodeInarray := range nodes {
		if nodeInarray.Name == node.Name {
			return true
		}
	}
	return false
}

func pluginExistInArray(plugin datatype.Plugin, plugins []datatype.Plugin) bool {
	for _, pluginInArray := range plugins {
		if pluginInArray.Name == plugin.Name &&
			pluginInArray.PluginSpec.Version == plugin.PluginSpec.Version {
			return true
		}
	}
	return false
}
