package cloudscheduler

import (
	"bufio"
	"fmt"
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
	Plugins          map[string]datatype.PluginManifest
	PluginsWhitelist map[string]bool
	Nodes            map[string]datatype.NodeManifest
}

func NewJobValidator(config *CloudSchedulerConfig) *JobValidator {
	return &JobValidator{
		dataPath:         config.DataDir,
		pluginDBURL:      config.ECRURL,
		nodeDBURL:        config.NodeManifestURL,
		Plugins:          make(map[string]datatype.PluginManifest),
		PluginsWhitelist: make(map[string]bool),
		Nodes:            make(map[string]datatype.NodeManifest),
	}
}

func (jv *JobValidator) GetNodeManifest(nodeName string) *datatype.NodeManifest {
	if n, exist := jv.Nodes[nodeName]; exist {
		return &n
	} else {
		return nil
	}
}

func (jv *JobValidator) GetPluginManifest(pluginImage string, updateDBIfNotExist bool) *datatype.PluginManifest {
	if p, exist := jv.Plugins[pluginImage]; exist {
		return &p
	} else {
		if updateDBIfNotExist {
			newP, err := jv.GetPluginManifestFromECR(pluginImage)
			if err != nil {
				logger.Error.Printf("failed to fetch plugin manifest: %s", err.Error())
				return nil
			} else {
				jv.Plugins[newP.ID] = *newP
				return newP
			}
		}
		return nil
	}
}

// IsPluginNameValid checks if given plugin name is valid.
// Plugin name must follow RFC 1123.
// Reference: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names
func (jv *JobValidator) IsPluginNameValid(name string) bool {
	// the maximum length allowed is 256, but the scheduler may use several characters
	// to indicate job ID when it names a plugin, thus reduce length of user plugins to 200
	if len(name) > 200 {
		return false
	}
	var validNamePattern = regexp.MustCompile("^[a-z0-9-]+$")
	return validNamePattern.MatchString(name)
}

// LoadDatabase loads node and plugin manifests
// this function should be called only at initialization
func (jv *JobValidator) LoadDatabase() error {
	jv.Plugins = make(map[string]datatype.PluginManifest)
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
		jv.Plugins[p.ID] = p
	}

	jv.Nodes = make(map[string]datatype.NodeManifest)
	// IMPROVEMENT: we may want to load node manifest from files first
	// in case the node manifest pull fails due to an error comuunicating with manifest server
	// nodeFiles, err := ioutil.ReadDir(path.Join(jv.dataPath, "nodes"))
	// if err != nil {
	// 	return err
	// }
	// for _, nodeFile := range nodeFiles {
	// 	nodeFilePath := path.Join(jv.dataPath, "nodes", nodeFile.Name())
	// 	raw, err := os.ReadFile(nodeFilePath)
	// 	if err != nil {
	// 		logger.Debug.Printf("Failed to read %s:%s", nodeFilePath, err.Error())
	// 		continue
	// 	}
	// 	var n datatype.NodeManifest
	// 	err = json.Unmarshal(raw, &n)
	// 	if err != nil {
	// 		logger.Debug.Printf("Failed to parse %s:%s", nodeFilePath, err.Error())
	// 		continue
	// 	}
	// 	jv.Nodes[n.Name] = &n
	// }
	if jv.nodeDBURL == "" {
		return fmt.Errorf("no nodeDB URL is given")
	}
	req = interfacing.NewHTTPRequest(jv.nodeDBURL)
	resp, err = req.RequestGet("manifests/", nil, additionalHeader)
	if err != nil {
		return err
	}
	decoder, err = req.ParseJSONHTTPResponse(resp)
	if err != nil {
		return err
	}
	nodes := []datatype.NodeManifest{}
	err = decoder.Decode(&nodes)
	if err != nil {
		return err
	}
	for _, n := range nodes {
		jv.Nodes[n.VSN] = n
	}

	return nil
}

func (jv *JobValidator) LoadPluginWhitelist() {
	jv.PluginsWhitelist = make(map[string]bool)
	whitelistFilePath := path.Join(jv.dataPath, "plugins.whitelist")
	if file, err := os.OpenFile(whitelistFilePath, os.O_CREATE|os.O_RDONLY, 0644); err == nil {
		fileScanner := bufio.NewScanner(file)
		for fileScanner.Scan() {
			jv.AddPluginWhitelist(fileScanner.Text())
		}
	} else {
		logger.Error.Printf("failed to create or open %q: %s", whitelistFilePath, err.Error())
	}
}

func (jv *JobValidator) AddPluginWhitelist(whitelist string) {
	if whitelist != "" {
		jv.PluginsWhitelist[whitelist] = true
	}
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

// GetPluginManifestFromECR attempts to find the plugin from edge code repository.
// If found, it adds the plugin in the in-memory DB
func (jv *JobValidator) GetPluginManifestFromECR(pluginImage string) (*datatype.PluginManifest, error) {
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
