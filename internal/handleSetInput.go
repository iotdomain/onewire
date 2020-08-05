// Package internal handles input set command
package internal

import (
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// HandleSetInput handles requests to update input value
// this is not yet supported
func (app *OnewireApp) HandleSetInput(
	input *types.InputDiscoveryMessage, sender string, value string) {
	logrus.Infof("OnewireApp.HandleSetInput for %s. Ignored as this isn't supported", input.Address)
}
