module iotd.onewire

go 1.13

require (
	github.com/iotdomain/iotdomain-go v0.0.0-20200618210420-9f2a2ec8914f
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/sys v0.0.0-20200615200032-f1bc736245b1 // indirect
)

// Temporary for testing iotdomain-go
replace github.com/iotdomain/iotdomain-go => ../iotdomain-go
