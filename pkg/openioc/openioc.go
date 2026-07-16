// Package openioc builds OpenIOC 1.1 documents from indicator items.
package openioc

import (
	"encoding/xml"
	"time"

	"github.com/google/uuid"
)

// Namespace is the OpenIOC 1.1 XML namespace.
const Namespace = "http://openioc.org/schemas/OpenIOC_1.1"

// Document is an OpenIOC 1.1 document.
type Document struct {
	XMLName       xml.Name  `xml:"OpenIOC"`
	Namespace     string    `xml:"xmlns,attr"`
	ID            string    `xml:"id,attr"`
	LastModified  time.Time `xml:"last-modified,attr"`
	PublishedDate time.Time `xml:"published-date,attr"`

	Metadata Metadata    `xml:"metadata"`
	Criteria []Indicator `xml:"criteria>Indicator"`
}

// Metadata holds OpenIOC document metadata. Element names follow the schema's
// snake_case naming.
type Metadata struct {
	ShortDescription string    `xml:"short_description,omitempty"`
	Keywords         string    `xml:"keywords,omitempty"`
	AuthoredBy       string    `xml:"authored_by"`
	AuthoredDate     time.Time `xml:"authored_date"`
}

// Indicator is a logical grouping of indicator items joined by an operator.
type Indicator struct {
	ID       string          `xml:"id,attr"`
	Operator string          `xml:"operator,attr"`
	Items    []IndicatorItem `xml:"IndicatorItem"`
}

// IndicatorItem is a single match condition. preserve-case and negate are
// required by the OpenIOC 1.1 schema; both default to false for exports.
type IndicatorItem struct {
	ID           string  `xml:"id,attr"`
	Condition    string  `xml:"condition,attr"`
	PreserveCase bool    `xml:"preserve-case,attr"`
	Negate       bool    `xml:"negate,attr"`
	Context      Context `xml:"Context"`
	Content      Content `xml:"Content"`
}

// Context describes what is being matched.
type Context struct {
	Document string `xml:"document,attr"`
	Search   string `xml:"search,attr"`
	Type     string `xml:"type,attr"`
}

// Content holds the matched value. Value is emitted as escaped character data.
type Content struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

// New creates an OpenIOC 1.1 document with a single top-level OR Indicator,
// ready for items to be appended via AddItem.
func New(author string, now time.Time) *Document {
	return &Document{
		Namespace:     Namespace,
		ID:            uuid.NewString(),
		LastModified:  now,
		PublishedDate: now,
		Metadata: Metadata{
			AuthoredBy:   author,
			AuthoredDate: now,
		},
		Criteria: []Indicator{{
			ID:       uuid.NewString(),
			Operator: "OR",
		}},
	}
}

// AddItem appends an indicator item to the document's top-level criteria.
func (d *Document) AddItem(condition string, ctx Context, contentType, value string) {
	d.Criteria[0].Items = append(d.Criteria[0].Items, IndicatorItem{
		Condition: condition,
		ID:        uuid.NewString(),
		Context:   ctx,
		Content:   Content{Type: contentType, Value: value},
	})
}
