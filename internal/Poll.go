// Package internal queries the EDS for device, node and parameter information
package internal

import (
	"errors"
	"fmt"
	"time"

	"github.com/hspaay/iotc.golang/iotc"
	"github.com/hspaay/iotc.golang/publisher"
)

// family to device type. See also: http://owfs.sourceforge.net/simple_family.html
// Todo: get from config file so it is easy to update
var deviceTypeMap = map[string]iotc.NodeType{
	"10": iotc.NodeTypeThermometer,
	"28": iotc.NodeTypeThermometer,
	"7E": iotc.NodeTypeMultisensor,
}

// SensorTypeMap attribute name map to sensor types
var SensorTypeMap = map[string]string{
	"BarometricPressureMb": iotc.OutputTypeAtmosphericPressure,
	"DewPoint":             iotc.OutputTypeDewpoint,
	"HeatIndex":            iotc.OutputTypeHeatIndex,
	"Humidity":             iotc.OutputTypeHumidity,
	"Humidex":              iotc.OutputTypeHumidex,
	"Light":                iotc.OutputTypeLuminance, // lux
	"RelayState":           iotc.OutputTypeRelay,
	"Temperature":          iotc.OutputTypeTemperature,
}
var unitNameMap = map[string]iotc.Unit{
	"PercentRelativeHumidity": iotc.UnitPercent,
	"Millibars":               iotc.UnitMillibar,
	"Centigrade":              iotc.UnitCelcius,
	"Fahrenheit":              iotc.UnitFahrenheit,
	"InchesOfMercury":         iotc.UnitMercury,
	"Lux":                     iotc.UnitLux,
	"#":                       iotc.UnitCount,
	"Volt":                    iotc.UnitVolt,
}

// updateSensor. A new or existing device sensor has been seen
// If this is a new sensor, add it to the device and return true. Existing sensors return false.
// This publishes updates to the sensor value except when the sensor is configured as disabled
// Limitations:
//   This only identifies sensors with a unit. Writable is not supported.
func (app *OnewireApp) updateSensor(nodeID string, sensorNode *XMLNode) {
	rawName := sensorNode.XMLName.Local

	// only handle known input types
	sensorType, _ := SensorTypeMap[rawName]
	if sensorType == "" {
		return
	}

	output := app.pub.GetOutputByType(nodeID, sensorType, iotc.DefaultOutputInstance)
	if output == nil {
		// convert OneWire EDS data type to IoTConnect output types
		rawUnit := sensorNode.Units
		output = app.pub.NewOutput(nodeID, sensorType, iotc.DefaultOutputInstance)
		output.Unit = unitNameMap[rawUnit]
		app.pub.Outputs.UpdateOutput(output)

		// writable devices also have an input
		if sensorNode.Writable == "True" {
			app.pub.NewInput(nodeID, sensorType, iotc.DefaultInputInstance)
		}
	}

	newVal := string(sensorNode.Content)
	app.pub.OutputValues.UpdateOutputValue(output.Address, newVal)
}

// updateDevice. A new or existing onewire device has been seen.
// If this is a new device then add.
// Existing nodes republish their property values.
func (app *OnewireApp) updateDevice(deviceOWNode *XMLNode) {
	props, _ := app.edsAPI.ParseNodeParams(deviceOWNode)

	// EDS Nodes all have a ROMId that uniquely identifies the device on the 1-wire bus
	nodeID, found := props["ROMId"]
	if !found {
		return
	}
	// Is this a new device?
	device := app.pub.GetNodeByID(nodeID)
	if device == nil {

		// crude determination of device type
		deviceType := deviceTypeMap[props["Family"]]
		if deviceType == "" {
			deviceType = iotc.NodeTypeUnknown // can we determine this?
		}
		app.pub.NewNode(nodeID, deviceType)
		// device = app.pub.Nodes.AddNode(nodeID, nodes.NodeType(deviceType))
	}

	// An EDS device xml has an attribute Description that contains the product description
	// Additional properties can be found in subnodes Name, Family, ROMId, Health, Channel
	app.pub.SetNodeAttr(nodeID, map[iotc.NodeAttr]string{
		iotc.NodeAttrAddress:     nodeID,
		iotc.NodeAttrDescription: deviceOWNode.Description,
		iotc.NodeAttrModel:       props["Name"],
		"Health":                 props["Health"],
		"Channel":                props["Channel"],
		"Resolution":             props["Resolution"],
	})
	//Publish newly discovered sensors and update the values of previously discovered properties
	for _, propXML := range deviceOWNode.Nodes {
		// TODO: create sensors first before publishing values to reduce the nr of device info postings during initial discovery
		app.updateSensor(nodeID, &propXML)
	}
}

// Update the one-wire gateway device status
// gwParams are the parameters as per EDS XML output
// Returns the gateway device node
func (app *OnewireApp) updateGateway(gwParams map[string]string) {

	app.pub.SetNodeAttr(app.config.GatewayID, map[iotc.NodeAttr]string{
		iotc.NodeAttrMAC:          gwParams["MACAddress"],
		iotc.NodeAttrHostname:     gwParams["HostName"],
		iotc.NodeAttrManufacturer: "Embedded Data Systems (EDS)",
		iotc.NodeAttrModel:        gwParams["DeviceName"],
	})

	// OWServer ENet specific attributes. These could be sensors if there is a need
	app.pub.SetNodeStatus(app.config.GatewayID, map[iotc.NodeStatus]string{
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
	// return gwNode
}

// Poll the EDS gateway for updates to nodes and sensors
func (app *OnewireApp) Poll(pub *publisher.Publisher) {
	// read the EDS gateway amd update the gateway state when disconnected
	nodeList := app.pub.Nodes
	gwID := app.config.GatewayID

	edsAPI := app.edsAPI
	edsAPI.address, _ = pub.GetNodeConfigValue(gwID, iotc.NodeAttrAddress)
	edsAPI.address, _ = pub.GetNodeConfigValue(gwID, iotc.NodeAttrAddress)
	edsAPI.loginName, _ = pub.GetNodeConfigValue(gwID, iotc.NodeAttrLoginName)
	edsAPI.password, _ = pub.GetNodeConfigValue(gwID, iotc.NodeAttrPassword)
	if edsAPI.address == "" {
		err := errors.New("a Gateway address has not been configured")
		app.log.Infof(err.Error())
		pub.SetNodeErrorStatus(gwID, iotc.NodeRunStateError, err.Error())
		return
	}
	startTime := time.Now()
	rootNode, err := edsAPI.ReadEds()
	endTime := time.Now()
	latency := endTime.Sub(startTime)

	if err != nil {
		err := fmt.Errorf("unable to connect to the gateway at %s", edsAPI.address)
		app.log.Infof(err.Error())
		pub.SetNodeErrorStatus(gwID, iotc.NodeRunStateError, err.Error())
		return
	}
	pub.SetNodeErrorStatus(gwID, iotc.NodeRunStateReady, "")

	// (re)discover the nodes on the gateway
	gwParams, deviceNodes := edsAPI.ParseNodeParams(rootNode)
	app.updateGateway(gwParams)
	pub.SetNodeStatus(gwID, map[iotc.NodeStatus]string{
		iotc.NodeStatusRunState:    string(iotc.NodeRunStateReady),
		iotc.NodeStatusLastError:   "",
		iotc.NodeStatusLatencyMSec: fmt.Sprintf("%d", latency.Milliseconds()),
	})

	// (re)discover any new sensor nodes and publish when changed
	for _, node := range deviceNodes {
		app.updateDevice(&node)
	}
	// in case configuration changes

	// in case configuration changes
	node := app.pub.PublisherNode()
	newPollInterval, err := nodeList.GetNodeConfigInt(node.Address, iotc.NodeAttrPollInterval)
	if err == nil {
		app.pub.SetPollInterval(newPollInterval, app.Poll)
	}
}
