package internal

import (
	"github.com/iotdomain/iotdomain-go/publisher"
	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// AppID application name used for configuration file and default publisherID
const AppID = "onewire"

// DefaultGatewayID with nodeId of the EDS gateway. Can be overridden in config.
const DefaultGatewayID = "gateway"

// OnewireAppConfig with application state, loaded from onewire.yaml
type OnewireAppConfig struct {
	GatewayAddress string `yaml:"gatewayAddress"` // default gateway IP address
	GatewayID      string `yaml:"gatewayId"`      // default gateway node ID
}

// OnewireApp publisher app
type OnewireApp struct {
	config          *OnewireAppConfig
	pub             *publisher.Publisher
	edsAPI          *EdsAPI // EDS device access definitions and methods
	gatewayNodeAddr string  // currently running address of the gateway node
}

// GatewayDeviceID return the device ID of the gateway node
func (app *OnewireApp) GatewayDeviceID() string {
	return app.config.GatewayID
}

// SetupGatewayNode creates the gateway device node
// This set the default gateway address in its configuration
func (app *OnewireApp) SetupGatewayNode(deviceID string) *types.NodeDiscoveryMessage {
	logrus.Info("SetupGatewayNode")
	pub := app.pub

	gwAddr := pub.MakeNodeDiscoveryAddress(deviceID)
	app.gatewayNodeAddr = gwAddr

	// Create new or use existing instance
	gatewayNode := pub.CreateNode(deviceID, types.NodeTypeGateway)

	pub.UpdateNodeConfig(deviceID, types.NodeAttrAddress, &types.ConfigAttr{
		DataType:    types.DataTypeString,
		Description: "EDS Gateway IP address",
		Default:     app.config.GatewayAddress,
	})
	pub.UpdateNodeConfig(deviceID, types.NodeAttrLoginName, &types.ConfigAttr{
		DataType:    types.DataTypeString,
		Description: "Login name of the onewire gateway",
		Secret:      true, // don't include value in discovery publication
	})
	pub.UpdateNodeConfig(deviceID, types.NodeAttrPassword, &types.ConfigAttr{
		DataType:    types.DataTypeString,
		Description: "Password of the onewire gateway",
		Secret:      true, // don't include value in discovery publication
	})

	// Onewire OWS Gateway is a node with configuration for address, login name and credentials
	// Gateway nodes are only discovered when a connection is made
	return gatewayNode
}

// NewOnewireApp creates the app
// This creates a node for the gateway
func NewOnewireApp(config *OnewireAppConfig, pub *publisher.Publisher) *OnewireApp {
	app := OnewireApp{
		config: config,
		pub:    pub,
		edsAPI: &EdsAPI{},
	}
	if app.config.GatewayID == "" {
		app.config.GatewayID = DefaultGatewayID
	}
	// pub.CreateNode(DefaultGatewayID, types.NodeTypeGateway)
	pub.SetPollInterval(60, app.Poll)
	pub.SetNodeConfigHandler(app.HandleConfigCommand)
	// // Discover the node(s) and outputs. Use default for republishing discovery
	// onewirePub.SetDiscoveryInterval(0, app.Discover)
	app.SetupGatewayNode(app.config.GatewayID)

	return &app
}

// Run the publisher until the SIGTERM  or SIGINT signal is received
func Run() {
	appConfig := &OnewireAppConfig{}
	onewirePub, _ := publisher.NewAppPublisher(AppID, "", appConfig, true)

	NewOnewireApp(appConfig, onewirePub)

	onewirePub.Start()
	onewirePub.WaitForSignal()
	onewirePub.Stop()
}
