package ltxmlharvest

import (
	"encoding/xml"
	"io"
)

// Harvest represents a single harvest.
// It implements sort.Interface
type Harvest []HarvestFragment

func (harvest Harvest) Len() int {
	return len(harvest)
}

func (harvest Harvest) Swap(i, j int) {
	harvest[i], harvest[j] = harvest[j], harvest[i]
}

func (harvest Harvest) Less(i, j int) bool {
	return harvest[i].URI < harvest[j].URI
}

// HarvestFragment represents a single document fragment within a harvest
type HarvestFragment struct {
	// ID is an internal, but unique, id of this harvest fragment
	// typically just the running id of this fragment
	ID string

	// URI is the URI of the corresponding document
	URI string

	// XHTMLContent of this document, substiuting "math" + id for formulae
	XHTMLContent string

	// List of formulae within the harvest
	Formulae []HarvestFormula
}

// HarvestFormula represents a single formula found within the harvest
type HarvestFormula struct {
	// ID of this formula
	ID string

	// Dual (Content + Presentation) MathML contained in this document
	// Content and Presentation should be linked using "xref" attributes.
	// May use "m" and "mws" namespaces.
	DualMathML string

	// Content MathML corresponding to the DualMathML above.
	// Must use the "m" namespace.
	ContentMathML string
}

const namespaceMathML = "http://www.w3.org/1998/Math/MathML"
const namespaceMWS = "http://search.mathweb.org/ns"

// WriteTo writes this harvest into writer and returns (0, error)
func (harvest Harvest) WriteTo(writer io.Writer) (n int64, err error) {
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "  ")

	// TODO: count number of bytes
	err = encoder.Encode(harvest)
	return
}

// MarshalXML marshals this harvest into xml form
func (harvest Harvest) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// <mws:harvest>
	start.Name.Local = "mws:harvest"
	start.Attr = append(start.Attr, xml.Attr{
		Name:  xml.Name{Local: "xmlns:mws"},
		Value: namespaceMWS,
	}, xml.Attr{
		Name:  xml.Name{Local: "xmlns:m"},
		Value: namespaceMathML,
	})
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// <mws:data> and <mws:expr> nodes
	for _, c := range harvest {
		if err := c.marshalXMLTo(e); err != nil {
			return err
		}
	}

	// </mws:harvest>
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// MarshalXML marshals this document into xml
func (frag HarvestFragment) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	harvest := Harvest([]HarvestFragment{frag})
	return harvest.MarshalXML(e, start)
}

func (frag HarvestFragment) marshalXMLTo(e *xml.Encoder) error {
	// <mws:data ...></mws:data>
	data := &xmlMWSDataElement{
		UUID:     frag.ID,
		ID:       frag.URI,
		Text:     frag.XHTMLContent,
		MetaData: "",
	}
	data.Formulae = make([]xmlMathElement, len(frag.Formulae))
	for i, f := range frag.Formulae {
		data.Formulae[i].URL = f.ID
		data.Formulae[i].Math = f.DualMathML
	}
	if err := e.Encode(data); err != nil {
		return err
	}

	var expr xmlMwsExprElement
	for _, f := range frag.Formulae {
		// <mws:expr ...></mws:expr>
		expr.URL = f.ID
		expr.UUID = frag.ID
		expr.Content = f.ContentMathML

		if err := e.EncodeElement(expr, xml.StartElement{Name: xml.Name{Local: "mws:expr"}}); err != nil {
			return err
		}
	}

	return nil
}

// xmlMWSDataElement represents a <mws:data> element
type xmlMWSDataElement struct {
	UUID     string `xml:"id,attr"`
	ID       string `xml:"id"`
	Text     string `xml:"text"`
	MetaData string `xml:"metadata"`
	Formulae []xmlMathElement
}

func (data xmlMWSDataElement) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// <mws:data id={uuid}>
	start.Name.Local = "mws:data"
	start.Attr = append(start.Attr, xml.Attr{
		Name:  xml.Name{Local: "mws:data_id"},
		Value: data.UUID,
	})
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// <id>{id}</id>
	if err := encodeStart(e, "id"); err != nil {
		return err
	}
	if err := encodeText(e, data.ID); err != nil {
		return err
	}
	if err := encodeEnd(e, "id"); err != nil {
		return err
	}

	// <text>{text}</text>
	if err := encodeStart(e, "text"); err != nil {
		return err
	}
	if err := encodeText(e, data.Text); err != nil {
		return err
	}
	if err := encodeEnd(e, "text"); err != nil {
		return err
	}

	// <metadata />
	if err := encodeStart(e, "metadata"); err != nil {
		return err
	}
	if err := encodeEnd(e, "metadata"); err != nil {
		return err
	}

	for _, f := range data.Formulae {
		// <math ...></math>
		if err := e.EncodeElement(f, xml.StartElement{Name: xml.Name{Local: "math"}}); err != nil {
			return err
		}
	}

	// </mws:data>
	return e.EncodeToken(xml.EndElement{Name: start.Name})
}

// xmlMathElement represets a <math> element inside <mws:data>
type xmlMathElement struct {
	URL  string `xml:"local_id,attr"`
	Math string `xml:",innerxml"`
}

// xmlMWSDataElement represents a <mws:expr> element
type xmlMwsExprElement struct {
	URL     string `xml:"url,attr"`
	UUID    string `xml:"mws:data_id,attr"`
	Content string `xml:",innerxml"`
}

func encodeStart(e *xml.Encoder, name string) error {
	return e.EncodeToken(xml.StartElement{Name: xml.Name{Local: name}})
}

func encodeText(e *xml.Encoder, data string) error {
	return e.EncodeToken(xml.CharData([]byte(data)))
}

func encodeEnd(e *xml.Encoder, name string) error {
	return e.EncodeToken(xml.EndElement{Name: xml.Name{Local: name}})
}
