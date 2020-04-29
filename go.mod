module iotconnect.onewire

go 1.13

require (
	github.com/hspaay/iotconnect.golang v0.0.0-20200416041144-e5d7862c6985
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/sirupsen/logrus v1.5.0
	github.com/stretchr/testify v1.5.1
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	golang.org/x/sys v0.0.0-20200413165638-669c56c373c4 // indirect
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

// Temporary for testing iotconnect.golang
replace github.com/hspaay/iotconnect.golang => ../iotconnect.golang
