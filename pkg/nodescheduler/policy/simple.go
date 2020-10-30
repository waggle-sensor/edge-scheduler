package policy

func NoSchedulingStrategy(plugins map[string]PluginConfig, currentPlugins []string) (prioritizedPlugins []string) {
	// Run them all!
	for k, _ := range plugins {
		prioritizedPlugins = append(prioritizedPlugins, k)
	}
	return
}
