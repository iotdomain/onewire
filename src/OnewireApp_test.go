package src

import (
	"testing"
	"time"

	"github.com/hspaay/iotconnect.golang/messaging"
	"github.com/hspaay/iotconnect.golang/messenger"
	"github.com/hspaay/iotconnect.golang/nodes"
	"github.com/hspaay/iotconnect.golang/persist"
	"github.com/hspaay/iotconnect.golang/publisher"
	"github.com/stretchr/testify/assert"
)

const zoneID = "zone1"
const configFolder = ""
const gwAddress = "10.3.3.33"

const TestConfigFolder = "test"
const Node1Id = "node1"

var messengerConfig = &messenger.MessengerConfig{}
var appConfig = &OnewireAppConfig{}

// TestLoadConfig load a node from config
func TestLoadConfig(t *testing.T) {
	t.Log("Testing loading config")
	// create app and load its configs
	var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	err := persist.LoadAppConfig(TestConfigFolder, OnewireAppID, appConfig)
	assert.NoError(t, err)
	err = persist.LoadMessengerConfig(TestConfigFolder, messengerConfig)
	assert.NoError(t, err)

	// create publisher and load its node configuration
	pub := publisher.NewPublisher(zoneID, testMessenger, TestConfigFolder)
	var nodeMap map[string]*nodes.Node
	err = persist.LoadNodes(TestConfigFolder, appConfig.PublisherID, &nodeMap)
	pub.Nodes.UpdateNodes(nodeMap)
	assert.NoError(t, err)
	pubNode := pub.Nodes.GetNodeByID(zoneID, pub.PublisherNode.ID, messaging.PublisherNodeID)
	assert.NotNil(t, pubNode, "Missing publisher node")

	device := pub.Nodes.GetNodeByID(zoneID, pub.PublisherNode.ID, Node1Id)
	assert.NotNil(t, device, "Node 1 not loaded") // 1 device
}

// Read EDS device and check if more than 1 node is returned. A minimum of 1 is expected if the device is online with
// an additional node for each connected node.
// This requires a live gateway on the above 'gwAddress'
func TestReadEds(t *testing.T) {
	// persist.LoadMessengerConfig(TestConfigFolder, messengerConfig)
	var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	persist.LoadAppConfig(TestConfigFolder, OnewireAppID, appConfig)

	pub := publisher.NewPublisher(zoneID, testMessenger, "")

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
	var testMessenger = messenger.NewDummyMessenger(messengerConfig, nil)
	err := persist.LoadAppConfig(TestConfigFolder, OnewireAppID, appConfig)
	if !assert.NoError(t, err) {
		return
	}
	pub := publisher.NewPublisher(zoneID, testMessenger, "")
	app := NewOnewireApp(appConfig, pub)

	assert.NoError(t, err)
	pub.Start()
	assert.NoError(t, err)
	app.Poll(pub)
	time.Sleep(3 * time.Second)
	pub.Stop()
}
