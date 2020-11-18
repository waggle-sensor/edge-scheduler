package policy

import (
	"github.com/sagecontinuum/ses/pkg/datatype"
)

// SimpleSchedulingPolicy simply returns the list as is
func SimpleSchedulingPolicy(plugins []*datatype.Plugin, availableResource datatype.Resource) (prioritizedPlugins []*datatype.Plugin) {
	return plugins
}
