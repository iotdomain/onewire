module iotc.onewire

go 1.13

require (
	github.com/hspaay/iotc.golang v0.0.0-20200416041144-e5d7862c6985
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.0
	golang.org/x/sys v0.0.0-20200413165638-669c56c373c4 // indirect
)

// Temporary for testing iotc.golang
replace github.com/hspaay/iotc.golang => ../iotc.golang
