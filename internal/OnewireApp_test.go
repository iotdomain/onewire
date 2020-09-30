package internal

import (
	"os"
	"testing"
	"time"

	"github.com/iotdomain/iotdomain-go/messaging"
	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// const gwAddress = "10.3.3.33"

const configFolder = "../test"
const Node1HWID = types.NodeIDGateway

var messengerConfig = &messaging.MessengerConfig{Domain: "test"}
var appConfig = &OnewireAppConfig{}

// TestLoadConfig load a node from config
func TestLoadConfig(t *testing.T) {
	os.Remove("../test/onewire-nodes.json")
	pub, err := publisher.NewAppPublisher(AppID, configFolder, appConfig, "", true)
	assert.NoError(t, err, "Failed creating AppPublisher")
	assert.Equal(t, "10.3.3.33", appConfig.GatewayAddress)
	assert.Equal(t, "onewire", pub.PublisherID())
	pub.Start()
	// create gateway
	_ = NewOnewireApp(appConfig, pub)

	// create publisher and load its node configuration
	allNodes := pub.GetNodes()
	assert.GreaterOrEqual(t, len(allNodes), 1, "Expected at least 1 node")

	device := pub.GetNodeByHWID(Node1HWID)
	assert.NotNil(t, device, "Node 1 not loaded") // 1 device
	pub.Stop()
}

// Read EDS test data from file
func TestReadEdsFromFile(t *testing.T) {
	edsAPI := EdsAPI{
		address: "file://../test/owserver-details.xml",
	}
	rootNode, err := edsAPI.ReadEds()
	assert.NoError(t, err)
	assert.NotNil(t, rootNode, "Expected root node")
	assert.True(t, len(rootNode.Nodes) == 20, "Expected 20 parameters and nested")

	// error case, unknown file
	edsAPI.address = "file://../doesnotexist.xml"
	rootNode, err = edsAPI.ReadEds()
	assert.Error(t, err)

}

// Read EDS device and check if more than 1 node is returned. A minimum of 1 is expected if the device is online with
// an additional node for each connected node.
// NOTE: This requires a live gateway on the above 'gwAddress'
func TestReadEdsFromGateway(t *testing.T) {
	pub, err := publisher.NewAppPublisher(AppID, configFolder, appConfig, configFolder, true)
	assert.NoError(t, err, "Failed creating AppPublisher")
	pub.Start()

	edsAPI := EdsAPI{
		address: appConfig.GatewayAddress,
	}
	rootNode, err := edsAPI.ReadEds()
	assert.NoError(t, err, "Failed reading EDS gateway")
	assert.NotNil(t, rootNode, "Expected root node")
	assert.GreaterOrEqual(t, len(rootNode.Nodes), 3, "Expected at least 3 nodes")
	pub.Stop()

	// error case - bad gateway
	// error case, unknown file
	edsAPI.address = "http://localhost/doesnotexist.xml"
	rootNode, err = edsAPI.ReadEds()
	assert.Error(t, err)
}

// Parse the nodes xml file and test for correct results
func TestParseNodeFile(t *testing.T) {
	// remove cached nodes first
	os.Remove("../test/onewire-nodes.json")
	pub, err := publisher.NewAppPublisher(AppID, configFolder, appConfig, "", false)
	pub.Start()
	app := NewOnewireApp(appConfig, pub)

	edsAPI := EdsAPI{
		address: "file://../test/owserver-details.xml",
	}
	rootNode, err := edsAPI.ReadEds()
	if !assert.NoError(t, err) {
		return
	}
	// The test file has gateway parameters and 3 connected nodes
	gwParams, deviceNodes := edsAPI.ParseNodeParams(rootNode)
	assert.Len(t, gwParams, 17, "Expected multiple gateway parameters")
	assert.Lenf(t, deviceNodes, 3, "Expected 3 gateway nodes")

	// Parameters should turn into node attributes
	app.updateGateway(gwParams)
	gwNode := pub.GetNodeByHWID(types.NodeIDGateway)
	assert.GreaterOrEqual(t, len(gwNode.Attr), 4, "Expected 4 or more attributes in gateway node")
	nrAttr := len(gwNode.Status)
	assert.GreaterOrEqual(t, nrAttr, 10, "Expected 10 status attributes in gateway node")

	// (re)discover any new sensor nodes and publish when changed
	for _, node := range deviceNodes {
		app.updateDevice(&node)
	}
	nodeList := pub.GetNodes()
	assert.Len(t, nodeList, 4, "Missing nodes, expect gateway and 3 device nodes")

	// There is one relay which is an input
	inputList := pub.GetInputs()
	assert.Len(t, inputList, 1, "Unexpected EDS node inputs")

	outputList := pub.GetOutputs()
	assert.Len(t, outputList, 10, "Missing EDS node outputs")
	pub.Stop()

}

func TestHandleConfigInput(t *testing.T) {
	pub, _ := publisher.NewAppPublisher(AppID, configFolder, appConfig, "", false)
	app := NewOnewireApp(appConfig, pub)
	// gwNode := app.SetupGatewayNode()

	// error cases - set nil config
	logrus.Infof("Testing config error cases")
	config := types.NodeAttrMap{}
	app.HandleConfigCommand("", config)

	// error case - set nil input
	app.HandleSetInput(nil, "nosender", "novalue")

	// error case - set input but nothing to set
	gwID := app.GatewayHWID()
	input := pub.CreateInput(gwID, types.InputTypeUnknown, types.DefaultInputInstance, nil)
	app.HandleSetInput(input, "nosender", "novalue")
}

func TestPollOnce(t *testing.T) {
	os.Remove("../test/onewire-nodes.json")
	pub, err := publisher.NewAppPublisher(AppID, configFolder, appConfig, "", false)
	pub.SetSigningOnOff(false)
	if !assert.NoError(t, err) {
		return
	}
	app := NewOnewireApp(appConfig, pub)
	// app.SetupGatewayNode()

	assert.NoError(t, err)
	pub.Start()

	app.Poll(pub)
	time.Sleep(3 * time.Second)

	pub.Stop()

	// error cases - don't panic when polling without address
	os.Remove("../test/onewire-nodes.json")
	pub, err = publisher.NewAppPublisher(AppID, configFolder, appConfig, "", false)
	appConfig.GatewayAddress = ""
	app.config.GatewayAddress = ""
	app = NewOnewireApp(appConfig, pub)
	app.Poll(pub)

	// error cases - don't panic when the gateway address is bad
	app.config.GatewayAddress = "bad"
	app.Poll(pub)
}
