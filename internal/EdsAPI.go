// Package internal Eds API methods
package internal

import (
	"encoding/xml"
	"net/http"

	"github.com/sirupsen/logrus"
)

// EdsAPI EDS device API properties and methods
type EdsAPI struct {
	address   string // EDS IP address
	loginName string // Basic Auth login name
	password  string // Basic Auth password
	log       *logrus.Logger
}

// XMLNode XML parsing node. Pure magic...
//--- https://stackoverflow.com/questions/30256729/how-to-traverse-through-xml-data-in-golang
type XMLNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:"-"`
	Content []byte     `xml:",innerxml"`
	Nodes   []XMLNode  `xml:",any"`
	// Possible attributes for subnodes, depending on the property name
	Description string `xml:"Description,attr"`
	Writable    string `xml:"Writable,attr"`
	Units       string `xml:"Units,attr"`
}

// UnmarshalXML parse xml
func (n *XMLNode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	n.Attrs = start.Attr
	type node XMLNode

	return d.DecodeElement((*node)(n), &start)
}

// ParseNodeParams Parse node and subnodes from EDS response message and return it as a nested key-value map
// returns a root node parameters and sub-nodes (gatewawy connected nodes)
func (edsAPI *EdsAPI) ParseNodeParams(xmlNode *XMLNode) (map[string]string, []XMLNode) {
	var params = make(map[string]string)
	var subNodes = make([]XMLNode, 0)
	for _, node := range xmlNode.Nodes {
		name := node.XMLName.Local
		content := string(node.Content)

		// if the xmlnode has no subnodes then it is a parameter describing the current node
		if len(node.Nodes) == 0 {
			params[name] = content
		} else {
			// The node contains subnodes which contains one or more sensors.
			subNodes = append(subNodes, node)
		}
	}
	return params, subNodes
}

// ReadEds reads EDS gateway and return the result as an XML node
func (edsAPI *EdsAPI) ReadEds() (rootNode *XMLNode, err error) {
	edsURL := "http://" + edsAPI.address + "/details.xml"

	resp, err := http.Get(edsURL)
	if err != nil {
		edsAPI.log.Warnf("pollDevice: Unable to read EDS gateway from %s: %v", edsURL, err)
		return nil, err
	}
	// Decode the EDS response into XML
	dec := xml.NewDecoder(resp.Body)
	//var rootNode XmlNode
	_ = dec.Decode(&rootNode)
	_ = resp.Body.Close()
	return rootNode, nil
}
