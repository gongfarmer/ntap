package ade

import (
	"bytes"
	"encoding/xml"
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
func (elt XMLContainer) ToAtom() (a *Atom) {
	return
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

	return buf.Bytes(), nil
}

func XMLDocumentToAtom(data []byte) (a *Atom, e error) {
	e = xml.Unmarshal(data, a)
	if e != nil {
		return nil, e
	}
	return
}

// MarshalXML writes an Atom to containerxml, rooted in the startElement.
func (a *Atom) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	err := e.Encode(a.AsXmlObject())
	if err != nil {
		return err
	}
	return nil
}

// MarshalXML writes an Atom to containerxml, rooted in the startElement.
func (a *Atom) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var x XMLContainer
	err := d.Decode(&x)
	a = x.ToAtom()
	if err != nil {
		return err
	}
	return nil
}

// // Postprocess to match existing C ADE xml handling, so unit tests will match.
// // This only exists to make the output match ADE, the XML is already
// // well-formed before this.
// func normalizeToADEStyle(buf []byte) []byte {
//
// 	// Replace atom closing tags with self-closing tags
// 	s := strings.Replace(string(buf), "></atom>", "/>", -1)
//
// 	// Replace container closing tags on empty containers with self-closing tags
// 	s = strings.Replace(s, "></container>", "/>", -1)
//
// 	// Use those instead of these
// 	s = strings.Replace(s, `\\`, `\`, -1)
// 	s = strings.Replace(s, `\x09`, `&#9;`, -1)
// 	s = strings.Replace(s, `\n`, `&#10;`, -1)
// 	s = strings.Replace(s, `\r`, `&#13;`, -1)
// 	s = strings.Replace(s, `\&#34;`, "&quot;", -1)
// 	s = strings.Replace(s, "&#39;", "'", -1)
//
// 	return []byte(s)
// }
