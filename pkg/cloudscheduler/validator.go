package cloudscheduler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
	"github.com/waggle-sensor/edge-scheduler/pkg/interfacing"
	"github.com/waggle-sensor/edge-scheduler/pkg/logger"
)

type JobValidator struct {
	dataPath         string
	pluginDBURL      string
	nodeDBURL        string
	Plugins          map[string]*datatype.PluginManifest
	PluginsWhitelist map[string]bool
	Nodes            map[string]*datatype.NodeManifest
}

func NewJobValidator(config *CloudSchedulerConfig) *JobValidator {
	return &JobValidator{
		dataPath:         config.DataDir,
		pluginDBURL:      config.ECRURL,
		nodeDBURL:        config.NodeManifestURL,
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

func (jv *JobValidator) GetPluginManifest(pluginImage string, updateDBIfNotExist bool) *datatype.PluginManifest {
	if p, exist := jv.Plugins[pluginImage]; exist {
		return p
	} else {
		if updateDBIfNotExist {
			newP, err := jv.AttemptToFindPluginManifest(pluginImage)
			if err != nil {
				logger.Error.Printf("failed to fetch plugin manifest: %s", err.Error())
				return nil
			} else {
				jv.Plugins[newP.ID] = newP
				return newP
			}
		}
		return nil
	}
}

// LoadDatabase loads node and plugin manifests
// this function should be called only at initialization
func (jv *JobValidator) LoadDatabase() error {
	jv.Plugins = make(map[string]*datatype.PluginManifest)
	// IMPROVEMENT: we may want to load plugin manifest from files first
	// in case the plugin manifest pull fails due to an error comuunicating with ECR

	// pluginFiles, err := ioutil.ReadDir(path.Join(jv.dataPath, "plugins"))
	// if err != nil {
	// 	return err
	// }
	// for _, pluginFile := range pluginFiles {
	// 	pluginFilePath := path.Join(jv.dataPath, "plugins", pluginFile.Name())
	// 	raw, err := os.ReadFile(pluginFilePath)
	// 	if err != nil {
	// 		logger.Debug.Printf("Failed to read %s:%s", pluginFilePath, err.Error())
	// 		continue
	// 	}
	// 	var p datatype.PluginManifest
	// 	err = json.Unmarshal(raw, &p)
	// 	if err != nil {
	// 		logger.Debug.Printf("Failed to parse %s:%s", pluginFilePath, err.Error())
	// 		continue
	// 	}
	// 	jv.Plugins[p.ID] = &p
	// }
	// cleaning up cached plugins

	if jv.pluginDBURL == "" {
		return fmt.Errorf("no pluginDB URL is given")
	}
	req := interfacing.NewHTTPRequest(jv.pluginDBURL)
	additionalHeader := map[string]string{
		"Accept": "application/json",
	}
	resp, err := req.RequestGet("api/apps", nil, additionalHeader)
	if err != nil {
		return err
	}
	decoder, err := req.ParseJSONHTTPResponse(resp)
	if err != nil {
		return err
	}
	plugins := struct {
		Data []datatype.PluginManifest `json:"data"`
	}{}
	err = decoder.Decode(&plugins)
	if err != nil {
		return err
	}
	for _, p := range plugins.Data {
		jv.Plugins[p.ID] = &p
	}

	jv.Nodes = make(map[string]*datatype.NodeManifest)
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

	return nil
}

func (jv *JobValidator) LoadPluginWhitelist() {
	jv.PluginsWhitelist = make(map[string]bool)
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

func (jv *JobValidator) IsPluginWhitelisted(pluginImage string) bool {
	for l := range jv.PluginsWhitelist {
		if matched, _ := regexp.MatchString(l, pluginImage); matched {
			return true
		}
	}
	return false
}

func (jv *JobValidator) WritePluginWhitelist() {
	whitelistFilePath := path.Join(jv.dataPath, "plugins.whitelist")
	if file, err := os.OpenFile(whitelistFilePath, os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		for whitelist := range jv.PluginsWhitelist {
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

// AttemptToFindPluginManifest attempts to find the plugin from edge code repository.
// If found, it adds the plugin in the in-memory DB
func (jv *JobValidator) AttemptToFindPluginManifest(pluginImage string) (*datatype.PluginManifest, error) {
	req := interfacing.NewHTTPRequest(jv.pluginDBURL)
	additionalHeader := map[string]string{
		"Accept": "application/json",
	}
	sp := strings.Split(pluginImage, "/")
	if len(sp) < 2 {
		return nil, fmt.Errorf("%s must consist of domain, plugin name, and version", pluginImage)
	}
	pluginNameVersion := strings.Split(path.Base(sp[len(sp)-1]), ":")
	if len(pluginNameVersion) != 2 {
		return nil, fmt.Errorf("%s must consist of plugin name, and version", sp[len(sp)-1])
	}
	subString := path.Join("api/apps", sp[len(sp)-2], pluginNameVersion[0], pluginNameVersion[1])
	resp, err := req.RequestGet(subString, nil, additionalHeader)
	if err != nil {
		return nil, err
	}
	decoder, err := req.ParseJSONHTTPResponse(resp)
	if err != nil {
		return nil, err
	}
	var p datatype.PluginManifest
	err = decoder.Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
