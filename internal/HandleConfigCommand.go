// Package internal handles node configuration commands
package internal

import (
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// HandleConfigCommand handles requests to update node configuration
// There are currently no node configurations to update to onewire
func (app *OnewireApp) HandleConfigCommand(nodeHWID string, config types.NodeAttrMap) {
	logrus.Infof("OnewireApp.HandleConfigCommand for %s.", nodeHWID)
	// for now accept all configuration
	app.pub.UpdateNodeConfigValues(nodeHWID, config)
}
