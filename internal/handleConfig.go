// Package internal handles node configuration commands
package internal

import "github.com/hspaay/iotc.golang/iotc"

// HandleConfigCommand handles requests to update node configuration
func (app *OnewireApp) HandleConfigCommand(node *iotc.NodeDiscoveryMessage, config iotc.NodeAttrMap) iotc.NodeAttrMap {
	app.logger.Infof("OnewireApp.HandleConfigCommand for %s. Ignored as this isn't supported", node.Address)
	return nil
}
