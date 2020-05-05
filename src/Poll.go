// Package src queries the EDS for device, node and parameter information
package src

import (
	"errors"
	"fmt"
	"time"

	"github.com/hspaay/iotc.golang/messaging"
	"github.com/hspaay/iotc.golang/nodes"
	"github.com/hspaay/iotc.golang/publisher"
)

// family to device type. See also: http://owfs.sourceforge.net/simple_family.html
var deviceTypeMap = map[string]messaging.NodeType{
	"10": messaging.NodeTypeThermometer,
	"28": messaging.NodeTypeThermometer,
	"7E": messaging.NodeTypeMultiSensor,
}

// SensorTypeMap attribute name map to sensor types
var SensorTypeMap = map[string]string{
	"BarometricPressureMb": messaging.OutputTypeAtmosphericPressure,
	"Dewpoint":             messaging.OutputTypeDewpoint,
	"HeatIndex":            messaging.OutputTypeHeatIndex,
	"Humidity":             messaging.OutputTypeHumidity,
	"Humidex":              messaging.OutputTypeHumidex,
	"Light":                messaging.OutputTypeLuminance, // lux
	"Relay":                messaging.OutputTypeContact,
	"Temperature":          messaging.OutputTypeTemperature,
}
var unitNameMap = map[string]messaging.Unit{
	"PercentRelativeHumidity": messaging.UnitPercent,
	"Millibars":               messaging.UnitMillibar,
	"Centigrade":              messaging.UnitCelcius,
	"Fahrenheit":              messaging.UnitFahrenheit,
	"InchesOfMercury":         messaging.UnitMercury,
	"Lux":                     messaging.UnitLux,
	"#":                       messaging.UnitCount,
	"Volt":                    messaging.UnitVolt,
}

// updateSensor. A new or existing sensor has been seen
// If this is a new sensor, add it to the device and return true. Existing sensors return false.
// This publishes updates to the sensor value except when the sensor is configured as disabled
// Limitations:
//   This only identifies sensors with a unit. Writable is not supported.
func (app *OnewireApp) updateSensor(node *nodes.Node, sensorNode *XMLNode) {
	rawName := sensorNode.XMLName.Local

	// only handle known input types
	sensorType, _ := SensorTypeMap[rawName]
	if sensorType == "" {
		return
	}

	output := app.pub.Outputs.GetOutput(node, sensorType, messaging.DefaultOutputInstance)
	if output == nil {
		// convert OneWire EDS data type to IoTConnect output types
		rawUnit := sensorNode.Units
		output = nodes.NewOutput(node, sensorType, messaging.DefaultOutputInstance)
		output.Unit = unitNameMap[rawUnit]
		app.pub.Outputs.UpdateOutput(output)

		// writable devices also have an input
		if sensorNode.Writable == "True" {
			input := nodes.NewInput(node, sensorType, messaging.DefaultInputInstance)
			app.pub.Inputs.UpdateInput(input)
		}
	}

	newVal := string(sensorNode.Content)
	app.pub.OutputValues.UpdateOutputValue(node, sensorType, messaging.DefaultOutputInstance, newVal)
}

// updateDevice. A new or existing onewire device has been seen.
// If this is a new device then add.
// Existing nodes republish their property values.
func (app *OnewireApp) updateDevice(deviceNode *XMLNode) {
	props, _ := app.edsAPI.ParseNodeParams(deviceNode)

	// EDS Nodes all have a ROMId that uniquely identifies the device on the 1-wire bus
	id, found := props["ROMId"]
	if !found {
		return
	}
	// Is this a new device?
	device := app.pub.GetNodeByID(id)
	if device == nil {

		// crude determination of device type
		deviceType := deviceTypeMap[props["Family"]]
		if deviceType == "" {
			deviceType = messaging.NodeTypeUnknown // can we determine this?
		}
		device = nodes.NewNode(app.pub.ZoneID, app.config.PublisherID, id, deviceType)
		// device = app.pub.Nodes.AddNode(id, nodes.NodeType(deviceType))
	}

	// An EDS device xml has an attribute Description that contains the product description
	// Additional properties can be found in subnodes Name, Family, ROMId, Health, Channel
	app.pub.Nodes.SetNodeAttr(device.Address, map[messaging.NodeAttr]string{
		messaging.NodeAttrAddress:     id,
		messaging.NodeAttrDescription: deviceNode.Description,
		messaging.NodeAttrModel:       props["Name"],
		"Health":                      props["Health"],
		"Channel":                     props["Channel"],
	})
	//Publish newly discovered sensors and update the values of previously discovered properties
	for _, propXML := range deviceNode.Nodes {
		// TODO: create sensors first before publishing values to reduce the nr of device info postings during initial discovery
		app.updateSensor(device, &propXML)
	}
}

// Update the one-wire gateway device status
// gwParams are the parameters as per EDS XML output
// Returns the gateway device node
func (app *OnewireApp) updateGateway(gwParams map[string]string) *nodes.Node {
	gwNode := app.pub.Nodes.GetNodeByAddress(app.gatewayAddr)
	app.pub.Nodes.SetNodeAttr(app.gatewayAddr, map[messaging.NodeAttr]string{
		messaging.NodeAttrMAC:          gwParams["MACAddress"],
		messaging.NodeAttrHostname:     gwParams["HostName"],
		messaging.NodeAttrManufacturer: "Embedded Data Systems (EDS)",
		messaging.NodeAttrModel:        gwParams["DeviceName"],
	})

	// OWServer ENet specific attributes. These could be sensors if there is a need
	app.pub.Nodes.SetNodeStatus(app.gatewayAddr, map[messaging.NodeStatus]string{
		"DevicesConnected":         gwParams["DevicesConnected"],
		"DevicesConnectedChannel1": gwParams["DevicesConnectedChannel1"],
		"DevicesConnectedChannel2": gwParams["DevicesConnectedChannel2"],
		"DevicesConnectedChannel3": gwParams["DevicesConnectedChannel3"],
		"DataErrorsChannel1":       gwParams["DataErrorsChannel1"],
		"DataErrorsChannel2":       gwParams["DataErrorsChannel2"],
		"DataErrorsChannel3":       gwParams["DataErrorsChannel3"],
		"VoltageChannel1":          gwParams["VoltageChannel1"],
		"VoltageChannel2":          gwParams["VoltageChannel2"],
		"VoltageChannel3":          gwParams["VoltageChannel3"],
	})
	return gwNode
}

// Poll the EDS gateway for updates to nodes and sensors
func (app *OnewireApp) Poll(pub *publisher.Publisher) {
	// read the EDS gateway amd update the gateway state when disconnected
	nodeList := app.pub.Nodes
	gwNode := app.pub.Nodes.GetNodeByAddress(app.gatewayAddr)
	if gwNode == nil {
		app.log.Error("Poll: gateway node not created")
		return
	}
	edsAPI := app.edsAPI
	edsAPI.address, _ = gwNode.GetConfigValue(messaging.NodeAttrAddress)
	edsAPI.loginName, _ = gwNode.GetConfigValue(messaging.NodeAttrLoginName)
	edsAPI.password, _ = gwNode.GetConfigValue(messaging.NodeAttrPassword)
	if edsAPI.address == "" {
		err := errors.New("a Gateway address has not been configured")
		app.log.Infof(err.Error())
		nodeList.SetErrorStatus(gwNode, err.Error())
		return
	}
	startTime := time.Now()
	rootNode, err := edsAPI.ReadEds()
	endTime := time.Now()
	latency := endTime.Sub(startTime)

	if err != nil {
		err := fmt.Errorf("unable to connect to the gateway at %s", edsAPI.address)
		app.log.Infof(err.Error())
		nodeList.SetErrorStatus(gwNode, err.Error())
		return
	}
	// (re)discover the nodes on the gateway
	gwParams, deviceNodes := edsAPI.ParseNodeParams(rootNode)
	app.updateGateway(gwParams)
	nodeList.SetNodeRunState(gwNode.Address, messaging.NodeRunStateReady)
	nodeList.SetNodeStatus(gwNode.Address, map[messaging.NodeStatus]string{
		messaging.NodeStatusLatencyMSec: fmt.Sprintf("%d", latency*time.Millisecond),
	})

	// (re)discover any new sensor nodes and publish when changed
	for _, node := range deviceNodes {
		app.updateDevice(&node)
	}
	// in case configuration changes

	// in case configuration changes
	newPollInterval, err := app.pub.PublisherNode.GetConfigInt(messaging.NodeAttrPollInterval)
	if err == nil {
		app.pub.SetPollInterval(newPollInterval, app.Poll)
	}
}
