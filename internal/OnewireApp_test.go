package internal

import (
	"testing"
	"time"

	"github.com/hspaay/iotc.golang/iotc"
	"github.com/hspaay/iotc.golang/messenger"
	"github.com/hspaay/iotc.golang/nodes"
	"github.com/hspaay/iotc.golang/persist"
	"github.com/hspaay/iotc.golang/publisher"
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
	pub := publisher.NewPublisher(messengerConfig.Zone, appConfig.PublisherID, testMessenger, TestConfigFolder)
	var nodeList []*nodes.Node
	err = persist.LoadNodes(TestConfigFolder, appConfig.PublisherID, &nodeList)
	assert.NoError(t, err)
	assert.Len(t, nodeList, 2, "Expected 2 nodes")
	pub.Nodes.UpdateNodes(nodeList)

	pubNode := pub.Nodes.GetNodeByID(messengerConfig.Zone, pub.ID(), iotc.PublisherNodeID)
	assert.NotNil(t, pubNode, "Missing publisher node")

	device := pub.Nodes.GetNodeByID(messengerConfig.Zone, pub.ID(), Node1Id)
	assert.NotNil(t, device, "Node 1 not loaded") // 1 device
}

// Read EDS device and check if more than 1 node is returned. A minimum of 1 is expected if the device is online with
// an additional node for each connected node.
// This requires a live gateway on the above 'gwAddress'
func TestReadEds(t *testing.T) {
	// persist.LoadMessengerConfig(TestConfigFolder, messengerConfig)
	var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	persist.LoadAppConfig(TestConfigFolder, AppID, appConfig)

	pub := publisher.NewPublisher(messengerConfig.Zone, appConfig.PublisherID, testMessenger, "")

	edsAPI := EdsAPI{
		address: gwAddress,
		log:     pub.Logger,
	}
	rootNode, err := edsAPI.ReadEds()
	assert.NoError(t, err)
	assert.NotNil(t, rootNode, "Expected root node")
	assert.True(t, len(rootNode.Nodes) > 1, "Expected multiple nodes")
}

func TestPollOnce(t *testing.T) {
	persist.LoadMessengerConfig(TestConfigFolder, messengerConfig)
	// var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	var testMessenger = messenger.NewMqttMessenger(messengerConfig, nil)
	err := persist.LoadAppConfig(TestConfigFolder, AppID, appConfig)
	if !assert.NoError(t, err) {
		return
	}
	pub := publisher.NewPublisher(messengerConfig.Zone, appConfig.PublisherID, testMessenger, TestConfigFolder)
	app := NewOnewireApp(appConfig, pub)
	app.SetupGatewayNode(pub)

	assert.NoError(t, err)
	pub.Start()
	assert.NoError(t, err)
	app.Poll(pub)
	time.Sleep(3 * time.Second)
	pub.Stop()
}
