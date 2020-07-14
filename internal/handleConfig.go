// Package internal handles node configuration commands
package internal

import (
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// HandleConfigCommand handles requests to update node configuration
func (app *OnewireApp) HandleConfigCommand(address string, config types.NodeAttrMap) types.NodeAttrMap {
	logrus.Infof("OnewireApp.HandleConfigCommand for %s. Ignored as this isn't supported", address)
	return nil
}
