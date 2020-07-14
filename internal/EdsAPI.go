// Package internal Eds API methods
package internal

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// EdsAPI EDS device API properties and methods
type EdsAPI struct {
	address   string // EDS (IP) address or filename (file://./path/to/name.xml)
	loginName string // Basic Auth login name
	password  string // Basic Auth password
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
// If edsAPI.address starts with file:// then read from file, otherwise from address
func (edsAPI *EdsAPI) ReadEds() (rootNode *XMLNode, err error) {
	if strings.HasPrefix(edsAPI.address, "file://") {
		filename := edsAPI.address[7:]
		buffer, err := ioutil.ReadFile(filename)
		if err != nil {
			logrus.Errorf("ReadEds: Unable to read EDS file from %s: %v", filename, err)
			return nil, err
		}
		err = xml.Unmarshal(buffer, &rootNode)
		return rootNode, err
	}
	// not a file, continue with http request
	edsURL := "http://" + edsAPI.address + "/details.xml"
	req, err := http.NewRequest("GET", edsURL, nil)
	req.SetBasicAuth(edsAPI.loginName, edsAPI.password)
	client := &http.Client{}
	resp, err := client.Do(req)

	// resp, err := http.Get(edsURL)
	if err != nil {
		logrus.Errorf("ReadEds: Unable to read EDS gateway from %s: %v", edsURL, err)
		return nil, err
	}
	// Decode the EDS response into XML
	dec := xml.NewDecoder(resp.Body)
	err = dec.Decode(&rootNode)
	_ = resp.Body.Close()

	return rootNode, err
}
