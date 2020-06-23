# Onewire Publisher

This publisher reads sensor information from the onewire EDS OWServer V2 gateway and publishes it on the message bus 
following the IoTDomain standard.


To build and/or install see the [iotdomain-go README](https://github.com/iotdomain/iotdomain-go)

## Dependencies

This publisher does not have any further dependencies, other than listed in the iotdomain-go README.md


## Configuration

Edit onewire.yaml with the default EDS OWServer V2 address and login credentials. This is optional as the onewire publisher includes a gateway node that can be configured with this information over the message bus (using a UI).

