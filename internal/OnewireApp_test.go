package internal

import (
	"testing"
	"time"

	"github.com/hspaay/iotc.golang/iotc"
	"github.com/hspaay/iotc.golang/messenger"
	"github.com/hspaay/iotc.golang/persist"
	"github.com/hspaay/iotc.golang/publisher"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const configFolder = ""
const gwAddress = "10.3.3.33"

const TestConfigFolder = "../test"
const Node1Id = GatewayID

var messengerConfig = &messenger.MessengerConfig{Zone: "test"}
var appConfig = &OnewireAppConfig{PublisherID: AppID}

// TestLoadConfig load a node from config
func TestLoadConfig(t *testing.T) {
	t.Log("Testing loading config")
	// create app and load its configs
	var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	err := persist.LoadMessengerConfig(TestConfigFolder, messengerConfig)
	assert.NoError(t, err)

	err = persist.LoadAppConfig(TestConfigFolder, AppID, appConfig)
	assert.NoError(t, err)

	// create publisher and load its node configuration
	pub := publisher.NewPublisher(messengerConfig.Zone, appConfig.PublisherID, testMessenger)
	err = pub.SetPersistNodes(TestConfigFolder, false)
	assert.NoError(t, err)
	assert.Len(t, pub.Nodes.GetAllNodes(), 2, "Expected 2 nodes")

	pubNode := pub.Nodes.GetNodeByID(messengerConfig.Zone, pub.ID(), iotc.PublisherNodeID)
	assert.NotNil(t, pubNode, "Missing publisher node")

	device := pub.Nodes.GetNodeByID(messengerConfig.Zone, pub.ID(), Node1Id)
	assert.NotNil(t, device, "Node 1 not loaded") // 1 device
}

// Read EDS test data from file
func TestReadEdsFromFile(t *testing.T) {
	edsAPI := EdsAPI{
		address: "file://../test/owserver-details.xml",
		log:     logrus.New(),
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
	// persist.LoadMessengerConfig(TestConfigFolder, messengerConfig)
	var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	persist.LoadAppConfig(TestConfigFolder, AppID, appConfig)

	pub := publisher.NewPublisher(messengerConfig.Zone, appConfig.PublisherID, testMessenger)

	edsAPI := EdsAPI{
		address: gwAddress,
		log:     pub.Logger,
	}
	rootNode, err := edsAPI.ReadEds()
	assert.NoError(t, err)
	assert.NotNil(t, rootNode, "Expected root node")
	assert.True(t, len(rootNode.Nodes) > 1, "Expected multiple nodes")
}

// Parse the nodes xml file and test for correct results
func TestParseNodeFile(t *testing.T) {
	var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	persist.LoadAppConfig(TestConfigFolder, AppID, appConfig)

	pub := publisher.NewPublisher(messengerConfig.Zone, appConfig.PublisherID, testMessenger)
	app := NewOnewireApp(appConfig, pub)

	edsAPI := EdsAPI{
		address: "file://../test/owserver-details.xml",
		log:     pub.Logger,
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
	gwNode := pub.GetNodeByID(GatewayID)
	assert.Len(t, gwNode.Attr, 4, "Missing gateway parameters in node")
	assert.Len(t, gwNode.Status, 10, "Missing gateway status attributes in node")

	// (re)discover any new sensor nodes and publish when changed
	for _, node := range deviceNodes {
		app.updateDevice(&node)
	}
	nodeList := pub.Nodes.GetAllNodes()
	assert.Len(t, nodeList, 5, "Missing nodes, expect publisher, gateway and 3 device nodes")

	// There is one relay which is an input
	inputList := pub.Inputs.GetAllInputs()
	assert.Len(t, inputList, 1, "Unexpected EDS node inputs")

	outputList := pub.Outputs.GetAllOutputs()
	assert.Len(t, outputList, 10, "Missing EDS node outputs")
}

func TestPollOnce(t *testing.T) {
	persist.LoadMessengerConfig(TestConfigFolder, messengerConfig)
	// var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	var testMessenger = messenger.NewMessenger(messengerConfig, nil)
	err := persist.LoadAppConfig(TestConfigFolder, AppID, appConfig)
	if !assert.NoError(t, err) {
		return
	}
	pub := publisher.NewPublisher(messengerConfig.Zone, appConfig.PublisherID, testMessenger)
	app := NewOnewireApp(appConfig, pub)
	pub.SetPersistNodes(TestConfigFolder, false)
	app.SetupGatewayNode(pub)

	assert.NoError(t, err)
	pub.Start()

	app.Poll(pub)
	time.Sleep(3 * time.Second)

	pub.Stop()
}
