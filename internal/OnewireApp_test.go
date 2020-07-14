package internal

import (
	"testing"
	"time"

	"github.com/iotdomain/iotdomain-go/messaging"
	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/stretchr/testify/assert"
)

// const gwAddress = "10.3.3.33"

const cacheFolder = "../test/cache"
const configFolder = "../test"
const Node1Id = DefaultGatewayID

var messengerConfig = &messaging.MessengerConfig{Domain: "test"}
var appConfig = &OnewireAppConfig{}

// TestLoadConfig load a node from config
func TestLoadConfig(t *testing.T) {
	pub, err := publisher.NewAppPublisher(AppID, configFolder, cacheFolder, appConfig, true)
	assert.NoError(t, err, "Failed creating AppPublisher")
	assert.Equal(t, "10.3.3.33", appConfig.GatewayAddress)
	assert.Equal(t, "onewire", pub.PublisherID())
	pub.Start()
	// create gateway
	_ = NewOnewireApp(appConfig, pub)

	// create publisher and load its node configuration
	allNodes := pub.GetNodes()
	assert.GreaterOrEqual(t, len(allNodes), 1, "Expected at least 1 node")

	device := pub.GetNodeByID(Node1Id)
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
}

// Read EDS device and check if more than 1 node is returned. A minimum of 1 is expected if the device is online with
// an additional node for each connected node.
// This requires a live gateway on the above 'gwAddress'
func TestReadEdsFromGateway(t *testing.T) {
	pub, err := publisher.NewAppPublisher(AppID, configFolder, cacheFolder, appConfig, true)
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
}

// Parse the nodes xml file and test for correct results
func TestParseNodeFile(t *testing.T) {
	pub, err := publisher.NewAppPublisher(AppID, configFolder, cacheFolder, appConfig, false)
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
	gwNode := pub.GetNodeByID(DefaultGatewayID)
	assert.Len(t, gwNode.Attr, 4, "Expected 4 attributes in gateway node")
	assert.Len(t, gwNode.Status, 10, "Expected 10 status attributes in gateway node")

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

func TestPollOnce(t *testing.T) {
	pub, err := publisher.NewAppPublisher(AppID, configFolder, cacheFolder, appConfig, false)
	if !assert.NoError(t, err) {
		return
	}
	app := NewOnewireApp(appConfig, pub)
	app.SetupGatewayNode(pub)

	assert.NoError(t, err)
	pub.Start()

	app.Poll(pub)
	time.Sleep(3 * time.Second)

	pub.Stop()
}
