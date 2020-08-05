// Package internal queries the EDS for device, node and parameter information
package internal

import (
	"errors"
	"fmt"
	"time"

	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// Poll the EDS gateway for updates to nodes and sensors
func (app *OnewireApp) Poll(pub *publisher.Publisher) {
	// read the EDS gateway amd update the gateway state when disconnected
	gwID := app.config.GatewayID

	edsAPI := app.edsAPI
	edsAPI.address, _ = pub.GetNodeConfigString(gwID, types.NodeAttrAddress, app.config.GatewayAddress)
	edsAPI.loginName, _ = pub.GetNodeConfigString(gwID, types.NodeAttrLoginName, "")
	edsAPI.password, _ = pub.GetNodeConfigString(gwID, types.NodeAttrPassword, "")
	if edsAPI.address == "" {
		err := errors.New("a Gateway address has not been configured")
		logrus.Infof(err.Error())
		pub.UpdateNodeErrorStatus(gwID, types.NodeRunStateError, err.Error())
		return
	}
	startTime := time.Now()
	rootNode, err := edsAPI.ReadEds()
	endTime := time.Now()
	latency := endTime.Sub(startTime)

	if err != nil {
		err := fmt.Errorf("unable to connect to the gateway at %s", edsAPI.address)
		logrus.Infof(err.Error())
		pub.UpdateNodeErrorStatus(gwID, types.NodeRunStateError, err.Error())
		return
	}
	pub.UpdateNodeErrorStatus(gwID, types.NodeRunStateReady, "")

	// (re)discover the nodes on the gateway
	gwParams, deviceNodes := edsAPI.ParseNodeParams(rootNode)
	app.updateGateway(gwParams)
	pub.UpdateNodeStatus(gwID, map[types.NodeStatus]string{
		types.NodeStatusRunState:    string(types.NodeRunStateReady),
		types.NodeStatusLastError:   "",
		types.NodeStatusLatencyMSec: fmt.Sprintf("%d", latency.Milliseconds()),
	})

	// (re)discover any new sensor nodes and publish when changed
	for _, node := range deviceNodes {
		app.updateDevice(&node)
	}
}
