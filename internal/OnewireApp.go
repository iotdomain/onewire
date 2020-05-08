package internal

import (
	"github.com/hspaay/iotc.golang/iotc"
	"github.com/hspaay/iotc.golang/messenger"
	"github.com/hspaay/iotc.golang/nodes"
	"github.com/hspaay/iotc.golang/persist"
	"github.com/hspaay/iotc.golang/publisher"
	"github.com/sirupsen/logrus"
)

// AppID application name used for configuration file and default publisherID
const AppID = "onewire"

// GatewayID with nodeId of the EDS gateway
const GatewayID = "gateway"

// const zoneID = iotc.LocalZoneID

// OnewireAppConfig with application state, loaded from onewire.conf
type OnewireAppConfig struct {
	PublisherID string `yaml:"publisherid"` // default is app ID
	GatewayAddr string `yaml:"gateway"`     // default gateway IP address
}

// OnewireApp publisher app
type OnewireApp struct {
	config      *OnewireAppConfig
	pub         *publisher.Publisher
	log         *logrus.Logger
	edsAPI      EdsAPI // EDS device access definitions and methods
	gatewayAddr string // address of the gateway connectin go
}

// SetupGatewayNode creates the gateway node if it doesn't exist
func (app *OnewireApp) SetupGatewayNode(pub *publisher.Publisher) {
	app.log.Info("DiscoverNodes:")

	app.gatewayAddr = nodes.MakeNodeDiscoveryAddress(app.pub.Zone, app.config.PublisherID, GatewayID)

	gatewayNode := pub.Nodes.GetNodeByAddress(app.gatewayAddr)
	if gatewayNode == nil {
		gatewayNode := nodes.NewNode(app.pub.Zone, app.config.PublisherID, GatewayID, iotc.NodeTypeGateway)

		config := nodes.NewConfigAttr(iotc.NodeAttrAddress, iotc.DataTypeString, "EDS Gateway IP address", app.config.GatewayAddr)
		pub.Nodes.SetNodeConfig(gatewayNode.Address, config)

		config = nodes.NewConfigAttr(iotc.NodeAttrLoginName, iotc.DataTypeString, "Login name of the onewire gateway", "")
		config.Secret = true
		pub.Nodes.SetNodeConfig(gatewayNode.Address, config)

		config = nodes.NewConfigAttr(iotc.NodeAttrPassword, iotc.DataTypeString, "Secret password of the onewire gateway", "")
		config.Secret = true
		pub.Nodes.SetNodeConfig(gatewayNode.Address, config)
		pub.Nodes.UpdateNode(gatewayNode)
	}

	// Onewire OWS Gateway is a node with configuration for address, login name and credentials
	// Gateway nodes are only discovered when a connection is made
	// node
}

// OnNodeConfigHandler handles requests to update node configuration
func (app *OnewireApp) OnNodeConfigHandler(node *iotc.NodeDiscoveryMessage, config iotc.NodeAttrMap) iotc.NodeAttrMap {
	return config
}

// NewOnewireApp creates the weather app
func NewOnewireApp(config *OnewireAppConfig, pub *publisher.Publisher) *OnewireApp {
	app := OnewireApp{
		config: config,
		pub:    pub,
		log:    pub.Logger,
	}
	app.config.PublisherID = AppID
	return &app
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
func Run() {
	logger := logrus.New()
	configFolder := persist.DefaultConfigFolder
	var messengerConfig = messenger.MessengerConfig{}

	persist.LoadMessengerConfig(configFolder, &messengerConfig)
	messenger := messenger.NewMessenger(&messengerConfig, logger)

	appConfig := &OnewireAppConfig{PublisherID: AppID}
	persist.LoadAppConfig(configFolder, AppID, appConfig)

	onewirePub := publisher.NewPublisher(messengerConfig.Zone, appConfig.PublisherID, messenger)
	onewirePub.PersistNodes(configFolder, true)
	app := NewOnewireApp(appConfig, onewirePub)

	// // Discover the node(s) and outputs. Use default for republishing discovery
	// onewirePub.SetDiscoveryInterval(0, app.Discover)

	// Poll gateway and nodes every minute
	onewirePub.SetPollInterval(60, app.Poll)

	// handle update of node configuraiton
	onewirePub.SetNodeConfigHandler(app.OnNodeConfigHandler)
	// handle update of node inputs
	// onewirePub.SetNodeInputHandler(onewireApp.OnNodeInputHandler)

	onewirePub.Start()
	onewirePub.WaitForSignal()
	onewirePub.Stop()
}
