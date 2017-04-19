package ade

import (
	"bytes"
	"encoding/xml"
	"strings"
)

type xmlElement interface {
	GetXMLName() xml.Name
}
type XMLAtom struct {
	XMLName xml.Name `xml:"atom"`
	Name    string   `xml:"name,attr"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:"value,attr"`
}

type XMLContainer struct {
	XMLName  xml.Name `xml:"container"`
	Name     string   `xml:"name,attr"`
	Children []*Atom
}

func (a *Atom) AsXmlObject() xmlElement {
	if a.Type() == "CONT" {
		return XMLContainer{Name: a.Name(), Children: a.Children()}
	}
	return XMLAtom{Name: a.Name(), Type: a.Type(), Value: a.ValueString()}
}

// XMLName() funcs exist only to satisfy a common interface that can hold both Atoms
// and Containers.
func (elt XMLAtom) GetXMLName() xml.Name {
	return elt.XMLName
}
func (elt XMLContainer) GetXMLName() xml.Name {
	return elt.XMLName
}

// MarshalXML writes an Atom to containerxml, rooted in the startElement.
func (a *Atom) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	err := e.Encode(a.AsXmlObject())
	if err != nil {
		return err
	}
	return nil
}

// Create a complete XML document in containerxml format with the given atom as
// the root.
func AtomToXMLDocumentText(root *Atom) ([]byte, error) {
	cxmlStart := `<containerxml version="1" xmlns="http://www.bycast.com/schemas/XML-container-1.0.0">` + "\n"
	cxmlEnd := "\n</containerxml>\n"
	var buf bytes.Buffer

	// Write xml header, containerxml start tag
	buf.WriteString(xml.Header)
	buf.WriteString(cxmlStart)

	// Write Atom data
	if moreBytes, e := xml.MarshalIndent(root, "\t", "\t"); e != nil {
		return nil, e
	} else {
		buf.Write(moreBytes)
	}

	// Write containerxml close tag
	buf.WriteString(cxmlEnd)

	// Postprocess to match existing C ADE xml handling, so unit tests will match

	// Replace atom closing tags with self-closing tags
	s := strings.Replace(buf.String(), "></atom>", "/>", -1)

	// Un-escape single quotes
	s = strings.Replace(s, "&#39;", "'", -1)

	return []byte(s), nil
}

func (a *Atom) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	//	vX := strucT{ X, Y, Z int }{}
	//	d.DecodeElement(&vX, &start)
	//
	//	*a = CopyVector{vX.X, vX.Y, vX.Z}

	return nil
}
