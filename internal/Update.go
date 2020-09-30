// Package internal with updates to nodes, inputs and outputs
package internal

import (
	"strings"

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
func (app *OnewireApp) updateSensor(nodeHWID string, sensorNode *XMLNode) {
	pub := app.pub
	rawName := sensorNode.XMLName.Local

	// only handle known input types
	sensorType, _ := SensorTypeMap[rawName]
	if sensorType == "" {
		return
	}

	output := pub.GetOutputByNodeHWID(nodeHWID, sensorType, types.DefaultOutputInstance)
	if output == nil {
		// convert OneWire EDS data type to IoTDomain output types
		rawUnit := sensorNode.Units
		output = pub.CreateOutput(nodeHWID, sensorType, types.DefaultOutputInstance)
		output.Unit = unitNameMap[rawUnit]
		pub.UpdateOutput(output)

		// writable devices also have an input
		if strings.ToLower(sensorNode.Writable) == "true" {
			pub.CreateInput(nodeHWID, types.InputType(sensorType),
				types.DefaultInputInstance, app.HandleSetInput)
		}
	}

	newVal := string(sensorNode.Content)
	pub.UpdateOutputValue(nodeHWID, sensorType, types.DefaultOutputInstance, newVal)
}

// updateDevice. A new or existing onewire device has been seen.
// If this is a new device then add.
// Existing nodes republish their property values.
func (app *OnewireApp) updateDevice(deviceOWNode *XMLNode) {
	pub := app.pub
	props, _ := app.edsAPI.ParseNodeParams(deviceOWNode)

	// EDS Nodes all have a ROMId that uniquely identifies the device on the 1-wire bus
	nodeHWID, found := props["ROMId"]
	if !found {
		// this is incomplete device data
		return
	}
	// Is this a new device?
	device := pub.GetNodeByHWID(nodeHWID)
	if device == nil {

		// crude determination of device type
		deviceType := deviceTypeMap[props["Family"]]
		if deviceType == "" {
			deviceType = types.NodeTypeUnknown // can we determine this?
		}
		pub.CreateNode(nodeHWID, deviceType)
		// device = app.pub.Nodes.AddNode(nodeID, nodes.NodeType(deviceType))
	}

	// An EDS device xml has an attribute Description that contains the product description
	// Additional properties can be found in subnodes Name, Family, ROMId, Health, Channel
	pub.UpdateNodeAttr(nodeHWID, map[types.NodeAttr]string{
		types.NodeAttrAddress:     nodeHWID,
		types.NodeAttrDescription: deviceOWNode.Description,
		types.NodeAttrModel:       props["Name"],
		"Health":                  props["Health"],
		"Channel":                 props["Channel"],
		"Resolution":              props["Resolution"],
	})
	//Publish newly discovered sensors and update the values of previously discovered properties
	for _, propXML := range deviceOWNode.Nodes {
		// TODO: create sensors first before publishing values to reduce the nr of device info postings during initial discovery
		app.updateSensor(nodeHWID, &propXML)
	}
}

// Update the one-wire gateway device status
// gwParams are the parameters as per EDS XML output
// Returns the gateway device node
func (app *OnewireApp) updateGateway(gwParams map[string]string) {
	pub := app.pub
	pub.UpdateNodeAttr(app.GatewayHWID(), map[types.NodeAttr]string{
		types.NodeAttrMAC:          gwParams["MACAddress"],
		types.NodeAttrHostname:     gwParams["HostName"],
		types.NodeAttrManufacturer: "Embedded Data Systems (EDS)",
		types.NodeAttrModel:        gwParams["DeviceName"],
	})

	// OWServer ENet specific attributes. These could be sensors if there is a need
	pub.UpdateNodeStatus(app.GatewayHWID(), map[types.NodeStatus]string{
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
