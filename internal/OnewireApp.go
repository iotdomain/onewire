package internal

import (
	"github.com/hspaay/iotc.golang/iotc"
	"github.com/hspaay/iotc.golang/nodes"
	"github.com/hspaay/iotc.golang/publisher"
	"github.com/sirupsen/logrus"
)

// AppID application name used for configuration file and default publisherID
const AppID = "onewire"

// GatewayID with nodeId of the EDS gateway
const GatewayID = "gateway"

// const zoneID = iotc.LocalZoneID

// OnewireAppConfig with application state, loaded from onewire.yaml
type OnewireAppConfig struct {
	PublisherID    string `yaml:"publisherId"`    // default publisher is app ID
	GatewayAddress string `yaml:"gatewayAddress"` // default gateway IP address
	GatewayID      string `yaml:"gatewayId"`      // default gateway node ID
}

// OnewireApp publisher app
type OnewireApp struct {
	config          *OnewireAppConfig
	pub             *publisher.Publisher
	logger          *logrus.Logger
	edsAPI          *EdsAPI // EDS device access definitions and methods
	gatewayNodeAddr string  // currently running address of the gateway node
}

// SetupGatewayNode creates the gateway node if it doesn't exist
// This set the default gateway address in its configuration
func (app *OnewireApp) SetupGatewayNode(pub *publisher.Publisher) {
	app.logger.Info("SetupGatewayNode")
	nodeList := pub.Nodes
	gwID := GatewayID

	gwAddr := nodes.MakeNodeDiscoveryAddress(app.pub.GetZone(), app.config.PublisherID, GatewayID)
	app.gatewayNodeAddr = gwAddr

	gatewayNode := pub.GetNodeByID(gwID)
	if gatewayNode == nil {
		pub.NewNode(gwID, iotc.NodeTypeGateway)
	}
	pub.NewNodeConfig(gwID, iotc.NodeAttrAddress, iotc.DataTypeString, "EDS Gateway IP address", app.config.GatewayAddress)

	config := nodes.NewNodeConfig(iotc.NodeAttrLoginName, iotc.DataTypeString, "Login name of the onewire gateway", "")
	config.Secret = true
	nodeList.UpdateNodeConfig(gwAddr, config)

	config = nodes.NewNodeConfig(iotc.NodeAttrPassword, iotc.DataTypeString, "Secret password of the onewire gateway", "")
	config.Secret = true
	nodeList.UpdateNodeConfig(gwAddr, config)
	// pub.Nodes.UpdateNode(gatewayNode)

	// Onewire OWS Gateway is a node with configuration for address, login name and credentials
	// Gateway nodes are only discovered when a connection is made
	// node
}

// NewOnewireApp creates the app
// This creates a node for the gateway
func NewOnewireApp(config *OnewireAppConfig, pub *publisher.Publisher) *OnewireApp {
	app := OnewireApp{
		config:          config,
		pub:             pub,
		logger:          pub.Logger,
		gatewayNodeAddr: nodes.MakeNodeDiscoveryAddress(pub.GetZone(), config.PublisherID, GatewayID),
		edsAPI:          &EdsAPI{},
	}
	app.config.PublisherID = AppID
	app.edsAPI.log = pub.Logger
	pub.SetPollInterval(60, app.Poll)
	// pub.SetNodeInputHandler(app.HandleInputCommand)
	pub.SetNodeConfigHandler(app.HandleConfigCommand)
	// // Discover the node(s) and outputs. Use default for republishing discovery
	// onewirePub.SetDiscoveryInterval(0, app.Discover)

	return &app
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
func Run() {
	appConfig := &OnewireAppConfig{PublisherID: AppID}
	onewirePub, _ := publisher.NewAppPublisher(AppID, "", appConfig, true)

	app := NewOnewireApp(appConfig, onewirePub)
	app.SetupGatewayNode(onewirePub)

	onewirePub.Start()
	onewirePub.WaitForSignal()
	onewirePub.Stop()
}
