package ltxmlharvest

// Harvest represents a single harvest
type Harvest []HarvestFragment

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
