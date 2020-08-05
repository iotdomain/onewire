module onewire

go 1.13

require (
	github.com/iotdomain/iotdomain-go v0.0.0-20200623050445-f9200737c15b
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
)

// Temporary for testing iotdomain-go
replace github.com/iotdomain/iotdomain-go => ../iotdomain-go
