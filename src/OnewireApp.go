package src

import (
	"github.com/hspaay/iotc.golang/messaging"
	"github.com/hspaay/iotc.golang/messenger"
	"github.com/hspaay/iotc.golang/nodes"
	"github.com/hspaay/iotc.golang/persist"
	"github.com/hspaay/iotc.golang/publisher"
	"github.com/sirupsen/logrus"
)

// OnewireAppID application name used for configuration file and default publisherID
const OnewireAppID = "onewire"

// GatewayID with nodeId of the EDS gateway
const GatewayID = "gateway"

// const zoneID = messaging.LocalZoneID

// OnewireAppConfig with application state, loaded from onewire.conf
type OnewireAppConfig struct {
	PublisherID string `yaml:"publisherid"`    // default is app ID
	GatewayAddr string `yaml:"gatewayaddress"` // default gateway IP address
}

// OnewireApp publisher app
type OnewireApp struct {
	config      *OnewireAppConfig
	pub         *publisher.Publisher
	log         *logrus.Logger
	edsAPI      EdsAPI // EDS device access definitions and methods
	gatewayAddr string // gateway node address
}

// SetupGatewayNode creates the gateway node if it doesn't exist
func (app *OnewireApp) SetupGatewayNode(pub *publisher.Publisher) {
	app.log.Info("DiscoverNodes:")

	app.gatewayAddr = nodes.MakeNodeDiscoveryAddress(app.pub.ZoneID, app.config.PublisherID, GatewayID)

	gatewayNode := pub.Nodes.GetNodeByAddress(app.gatewayAddr)
	if gatewayNode == nil {
		gatewayNode := nodes.NewNode(app.pub.ZoneID, app.config.PublisherID, GatewayID, messaging.NodeTypeGateway)

		config := nodes.NewConfigAttr(messaging.NodeAttrAddress, messaging.DataTypeString, "EDS Gateway IP address", app.config.GatewayAddr)
		pub.Nodes.SetNodeConfig(gatewayNode.Address, config)

		config = nodes.NewConfigAttr(messaging.NodeAttrLoginName, messaging.DataTypeString, "Login name of the onewire gateway", "")
		config.Secret = true
		pub.Nodes.SetNodeConfig(gatewayNode.Address, config)

		config = nodes.NewConfigAttr(messaging.NodeAttrPassword, messaging.DataTypeString, "Secret password of the onewire gateway", "")
		config.Secret = true
		pub.Nodes.SetNodeConfig(gatewayNode.Address, config)
		pub.Nodes.UpdateNode(gatewayNode)
	}

	// Onewire OWS Gateway is a node with configuration for address, login name and credentials
	// Gateway nodes are only discovered when a connection is made
	// node
}

// OnNodeConfigHandler handles requests to update node configuration
func (app *OnewireApp) OnNodeConfigHandler(node *nodes.Node, config messaging.NodeAttrMap) messaging.NodeAttrMap {
	return config
}

// NewOnewireApp creates the weather app
func NewOnewireApp(config *OnewireAppConfig, pub *publisher.Publisher) *OnewireApp {
	app := OnewireApp{
		config: config,
		pub:    pub,
		log:    pub.Logger,
	}
	app.config.PublisherID = OnewireAppID
	return &app
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
func Run() {
	logger := logrus.New()
	configFolder := persist.DefaultConfigFolder
	var messengerConfig = messenger.MessengerConfig{}

	persist.LoadMessengerConfig(configFolder, &messengerConfig)
	messenger := messenger.NewMessenger(&messengerConfig, logger)

	appConfig := &OnewireAppConfig{PublisherID: OnewireAppID}
	persist.LoadAppConfig(configFolder, OnewireAppID, appConfig)

	onewirePub := publisher.NewPublisher(appConfig.PublisherID, messenger, configFolder)

	app := NewOnewireApp(appConfig, onewirePub)

	// Discover the node(s) and outputs. Use default for republishing discovery
	onewirePub.SetDiscoveryInterval(0, app.SetupGatewayNode)
	// Update the forecast once an hour
	onewirePub.SetPollInterval(3600, app.Poll)

	// handle update of node configuraiton
	onewirePub.SetNodeConfigHandler(app.OnNodeConfigHandler)
	// handle update of node inputs
	// onewirePub.SetNodeInputHandler(onewireApp.OnNodeInputHandler)

	onewirePub.Start()
	onewirePub.WaitForSignal()
	onewirePub.Stop()
}
