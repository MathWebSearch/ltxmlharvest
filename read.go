package ltxmlharvest

import (
	"bytes"
	"errors"
	"io"
	"log"

	"github.com/beevik/etree"
)

// ReadXHTML reads xhtml from reader and returns a HarvestFragment
func ReadXHTML(reader io.Reader) (fragment HarvestFragment, err error) {
	// read the xhtml from the source document
	// and bail out when there is an error
	doc := etree.NewDocument()
	if _, err := doc.ReadFrom(reader); err != nil {
		return fragment, err
	}

	// find all the <math> elements within the document
	// and treat them like a formula!
	for element := range findMath(&doc.Element) {

		// parse it as a formula or log an error

		f, err := ReadFormula(element)
		if err != nil {
			log.Println(err)
			continue
		}
		fragment.Formulae = append(fragment.Formulae, f)

		// replace the formula with "math" + id in the document
		// this will enable usage in TemaSearch in the future
		parent := element.Parent()
		index := element.Index()
		parent.RemoveChildAt(index)
		parent.InsertChildAt(index, etree.NewText("math"+f.ID))
	}

	// write out the xhtml content
	// so that elasticsearch could index it!
	fragment.XHTMLContent = innerText(doc)

	return
}

// ReadFormula parses a formula based on element
func ReadFormula(math *etree.Element) (HarvestFormula, error) {
	var annotation *etree.Element
	for _, ax := range math.FindElements("./semantics/annotation-xml") {
		if getAttr(ax, "encoding") == "MathML-Content" {
			annotation = ax
			break
		}
	}

	if annotation == nil {
		return HarvestFormula{}, errors.New("ReadFormula: Missing Content MathML")
	}

	ID := getAttr(math, "id")
	if ID == "" {
		return HarvestFormula{}, errors.New("ReadFormula: Missing Content ID")
	}

	return HarvestFormula{
		ID:            getAttr(math, "id"),
		DualMathML:    outerXML(math),
		ContentMathML: innerXML(annotation),
	}, nil
}

func getAttr(element *etree.Element, name string) string {
	for _, attr := range element.Attr {
		if attr.Key == name {
			return attr.Value
		}
	}

	return ""
}

// findMath recursively finds <m:math> elements inside root
func findMath(root *etree.Element) <-chan *etree.Element {
	res := make(chan *etree.Element)
	go func() {
		defer close(res)
		findMathRecursive(root, res)
	}()
	return res
}

// findMathRecursive recursively finds "math" elements and writes them to write
func findMathRecursive(element *etree.Element, write chan<- *etree.Element) {
	// it's a math element!
	if element.NamespaceURI() == namespaceMathML && element.Tag == "math" {
		write <- element
		return
	}

	// search all the children
	for _, c := range element.ChildElements() {
		findMathRecursive(c, write)
	}
}

func innerText(doc *etree.Document) string {
	var buffer bytes.Buffer
	innerTextRecursive(&doc.Element, &buffer)
	return buffer.String()
}

func innerTextRecursive(element *etree.Element, writer io.Writer) {
	io.WriteString(writer, element.Text())
	for _, c := range element.Child {
		// Element, CharData, Comment, Directive, or ProcInst.
		switch t := c.(type) {
		case *etree.Element:
			innerTextRecursive(t, writer)
		case *etree.CharData:
			io.WriteString(writer, t.Data)
		}
	}
}

func innerXML(element *etree.Element) string {
	doc := etree.NewDocument()
	for _, c := range element.Copy().Child {
		doc.AddChild(c)
	}

	var buffer bytes.Buffer
	doc.WriteTo(&buffer)
	return buffer.String()
}

func outerXML(element *etree.Element) string {
	doc := etree.NewDocument()
	doc.AddChild(element.Copy())

	var buffer bytes.Buffer
	doc.WriteTo(&buffer)
	return buffer.String()
}
