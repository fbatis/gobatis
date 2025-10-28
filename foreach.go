package gobatis

import (
	"encoding/xml"
	"strings"
)

type Foreach struct {
	Children []interface{}
	Attrs    []xml.Attr
	AttrsMap map[string]string
	Sql      []*Sql
}

func NewForeach() *Foreach {
	return &Foreach{
		Attrs:    []xml.Attr{},
		AttrsMap: make(map[string]string, 32),
		Sql:      []*Sql{},
	}
}

func (m *Foreach) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {

	m.Attrs = start.Attr
	for _, attr := range start.Attr {
		m.AttrsMap[XmlName(attr.Name).Name()] = strings.TrimSpace(attr.Value)
	}

	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			if ele, err := parseElementEntry(d, &el); err != nil {
				return err
			} else {
				m.Children = append(m.Children, ele)
			}
		case xml.CharData:
			m.Children = append(m.Children, el.Copy())
		case xml.EndElement:
			return nil
		case xml.Comment, xml.ProcInst, xml.Directive:
		}
	}
}
