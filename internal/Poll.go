// Package internal queries the EDS for device, node and parameter information
package internal

import (
	"errors"
	"fmt"
	"time"

	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
)

// family to device type. See also: http://owfs.sourceforge.net/simple_family.html
// Todo: get from config file so it is easy to update
var deviceTypeMap = map[string]types.NodeType{
	"10": types.NodeTypeThermometer,
	"28": types.NodeTypeThermometer,
	"7E": types.NodeTypeMultisensor,
}

// SensorTypeMap attribute name map to sensor types
var SensorTypeMap = map[string]types.OutputType{
	"BarometricPressureMb": types.OutputTypeAtmosphericPressure,
	"DewPoint":             types.OutputTypeDewpoint,
	"HeatIndex":            types.OutputTypeHeatIndex,
	"Humidity":             types.OutputTypeHumidity,
	"Humidex":              types.OutputTypeHumidex,
	"Light":                types.OutputTypeLuminance, // lux
	"RelayState":           types.OutputTypeRelay,
	"Temperature":          types.OutputTypeTemperature,
}
var unitNameMap = map[string]types.Unit{
	"PercentRelativeHumidity": types.UnitPercent,
	"Millibars":               types.UnitMillibar,
	"Centigrade":              types.UnitCelcius,
	"Fahrenheit":              types.UnitFahrenheit,
	"InchesOfMercury":         types.UnitMercury,
	"Lux":                     types.UnitLux,
	"#":                       types.UnitCount,
	"Volt":                    types.UnitVolt,
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

	output := app.pub.GetOutputByType(nodeID, sensorType, types.DefaultOutputInstance)
	if output == nil {
		// convert OneWire EDS data type to IoTDomain output types
		rawUnit := sensorNode.Units
		output = app.pub.NewOutput(nodeID, sensorType, types.DefaultOutputInstance)
		output.Unit = unitNameMap[rawUnit]
		app.pub.Outputs.UpdateOutput(output)

		// writable devices also have an input
		if sensorNode.Writable == "True" {
			app.pub.NewInput(nodeID, types.InputType(sensorType), types.DefaultInputInstance)
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
			deviceType = types.NodeTypeUnknown // can we determine this?
		}
		app.pub.NewNode(nodeID, deviceType)
		// device = app.pub.Nodes.AddNode(nodeID, nodes.NodeType(deviceType))
	}

	// An EDS device xml has an attribute Description that contains the product description
	// Additional properties can be found in subnodes Name, Family, ROMId, Health, Channel
	app.pub.SetNodeAttr(nodeID, map[types.NodeAttr]string{
		types.NodeAttrAddress:     nodeID,
		types.NodeAttrDescription: deviceOWNode.Description,
		types.NodeAttrModel:       props["Name"],
		"Health":                  props["Health"],
		"Channel":                 props["Channel"],
		"Resolution":              props["Resolution"],
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

	app.pub.SetNodeAttr(app.config.GatewayID, map[types.NodeAttr]string{
		types.NodeAttrMAC:          gwParams["MACAddress"],
		types.NodeAttrHostname:     gwParams["HostName"],
		types.NodeAttrManufacturer: "Embedded Data Systems (EDS)",
		types.NodeAttrModel:        gwParams["DeviceName"],
	})

	// OWServer ENet specific attributes. These could be sensors if there is a need
	app.pub.SetNodeStatus(app.config.GatewayID, map[types.NodeStatus]string{
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
	gwID := app.config.GatewayID

	edsAPI := app.edsAPI
	edsAPI.address, _ = pub.GetNodeConfigString(gwID, types.NodeAttrAddress, app.config.GatewayAddress)
	edsAPI.loginName, _ = pub.GetNodeConfigString(gwID, types.NodeAttrLoginName, "")
	edsAPI.password, _ = pub.GetNodeConfigString(gwID, types.NodeAttrPassword, "")
	if edsAPI.address == "" {
		err := errors.New("a Gateway address has not been configured")
		app.logger.Infof(err.Error())
		pub.SetNodeErrorStatus(gwID, types.NodeRunStateError, err.Error())
		return
	}
	startTime := time.Now()
	rootNode, err := edsAPI.ReadEds()
	endTime := time.Now()
	latency := endTime.Sub(startTime)

	if err != nil {
		err := fmt.Errorf("unable to connect to the gateway at %s", edsAPI.address)
		app.logger.Infof(err.Error())
		pub.SetNodeErrorStatus(gwID, types.NodeRunStateError, err.Error())
		return
	}
	pub.SetNodeErrorStatus(gwID, types.NodeRunStateReady, "")

	// (re)discover the nodes on the gateway
	gwParams, deviceNodes := edsAPI.ParseNodeParams(rootNode)
	app.updateGateway(gwParams)
	pub.SetNodeStatus(gwID, map[types.NodeStatus]string{
		types.NodeStatusRunState:    string(types.NodeRunStateReady),
		types.NodeStatusLastError:   "",
		types.NodeStatusLatencyMSec: fmt.Sprintf("%d", latency.Milliseconds()),
	})

	// (re)discover any new sensor nodes and publish when changed
	for _, node := range deviceNodes {
		app.updateDevice(&node)
	}
}
