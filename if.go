package gobatis

import (
	"context"
	"encoding/xml"
)

const (
	TestKey            = `test`
	PrefixOverridesKey = `prefixOverrides`
	PrefixKey          = `prefix`
	CollectionKey      = `collection`
	ItemKey            = `item`
	SeparatorKey       = `separator`
	IdKey              = `id`
	TypeKey            = `type`
	IndexKey           = `index`
)

type If struct {
	Children []interface{}
	Attrs    []xml.Attr
	AttrsMap map[string]string
	Sql      []*Sql
}

func NewIf() *If {
	return &If{
		Attrs:    []xml.Attr{},
		AttrsMap: make(map[string]string, 32),
		Sql:      []*Sql{},
	}
}

func (m *If) Evaluate(ctx context.Context, input *HandlerPayload) (string, error) {
	return intervalEvaluate(ctx, m.Children, input)
}

func (m *If) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	m.Attrs = start.Attr
	for _, attr := range m.Attrs {
		m.AttrsMap[XmlName(attr.Name).Name()] = attr.Value
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
