package ltxmlharvest

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/beevik/etree"
)

func (f *HarvestFragment) ReadFrom(reader io.Reader) (n int64, err error) {
	// read the xhtml from the source document
	// and bail out when there is an error
	doc := etree.NewDocument()
	if n, err = doc.ReadFrom(reader); err != nil {
		return
	}

	// find all the <math> elements within the document
	// and treat them like a formula!
	for element := range findMathNodes(&doc.Element) {

		// parse it as a formula or log an error

		ff, err := ReadFormula(element)
		if err != nil { // if we can't read a formula, skip it!
			continue
		}
		f.Formulae = append(f.Formulae, ff)

		// replace the formula with "math" + id in the document
		// this will enable usage in TemaSearch in the future
		parent := element.Parent()
		index := element.Index()
		parent.RemoveChildAt(index)
		parent.InsertChildAt(index, etree.NewText("math"+ff.ID))
	}

	// write out the xhtml content
	// so that elasticsearch could index it!
	f.XHTMLContent = documentToText(doc)

	return
}

// ReadFormula parses a formula based on element
func ReadFormula(math *etree.Element) (HarvestFormula, error) {
	var annotation *etree.Element
	for _, ax := range math.FindElements("./semantics/annotation-xml") {
		if getElementAttr(ax, "encoding", "") == "MathML-Content" {
			annotation = ax
			break
		}
	}

	if annotation == nil {
		return HarvestFormula{}, errors.New("ReadFormula: Missing Content MathML")
	}

	ID := getElementAttr(math, "id", "")
	if ID == "" {
		return HarvestFormula{}, errors.New("ReadFormula: Missing Content ID")
	}

	return HarvestFormula{
		ID:            getElementAttr(math, "id", ""),
		DualMathML:    elementToXML(math),
		ContentMathML: elementToText(annotation),
	}, nil
}

// getElementAttr gets an xhtml attribute, or default if it doesn't exist
func getElementAttr(element *etree.Element, name, dflt string) string {
	for _, attr := range element.Attr {
		if attr.Key == name {
			return attr.Value
		}
	}

	return dflt
}

// findMathNodes recursively finds <m:math> elements inside root
func findMathNodes(root *etree.Element) <-chan *etree.Element {
	res := make(chan *etree.Element)
	go func() {
		defer close(res)
		findMathRecursive(root, res)
	}()
	return res
}

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

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// documentToText returns text found inside a document
func documentToText(doc *etree.Document) string {
	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bufferPool.Put(buffer)

	writeElementText(&doc.Element, buffer)
	return squashedString(buffer.Bytes())
}

// elementToText returns text inside an element
func elementToText(element *etree.Element) string {
	doc := etree.NewDocument()
	for _, c := range element.Copy().Child {
		doc.AddChild(c)
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bufferPool.Put(buffer)

	doc.WriteTo(buffer)
	return squashedString(buffer.Bytes())
}

func writeElementText(element *etree.Element, writer io.Writer) {
	io.WriteString(writer, element.Text())
	io.WriteString(writer, " ")
	for _, c := range element.Child {
		switch t := c.(type) {
		case *etree.Element:
			writeElementText(t, writer)
		case *etree.CharData:
			io.WriteString(writer, t.Data)
		}
	}
}

var spaceRegex = regexp.MustCompile(`\s+`)

// replace multiple spaces by one and return a string
func squashedString(bytes []byte) string {
	s := spaceRegex.ReplaceAll(bytes, []byte(" "))
	return strings.TrimSpace(string(s))
}

// elementToXML returns xml making up an element
func elementToXML(element *etree.Element) string {
	doc := etree.NewDocument()
	doc.AddChild(element.Copy())

	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bufferPool.Put(buffer)

	doc.WriteTo(buffer)
	return buffer.String()
}
