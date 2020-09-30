// Package internal handles input set command
package internal

import (
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// HandleSetInput handles requests to update input value
// TODO: support for controlling onewire inputs
func (app *OnewireApp) HandleSetInput(
	input *types.InputDiscoveryMessage, sender string, value string) {
	if input == nil {
		logrus.Errorf("HandleSetInput: input is nil")
		return
	}
	logrus.Warnf("OnewireApp.HandleSetInput for %s. Ignored as this isn't supported", input.Address)
}
