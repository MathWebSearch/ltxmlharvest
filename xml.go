package ltxmlharvest

import "encoding/xml"

const namespaceMathML = "http://www.w3.org/1998/Math/MathML"
const namespaceMWS = "http://search.mathweb.org/ns"

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
