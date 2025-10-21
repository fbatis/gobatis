package gobatis

import (
	"context"
	"encoding/xml"
)

// Update children can be one of: CharData, If, Elif, Else, Choose, Where, Foreach, Trim, Otherwise, Include, Sql,
type Update struct {
	Children []interface{}
	Attrs    []xml.Attr
	AttrsMap map[string]string
	Sql      []*Sql

	Text string `xml:",chardata"`
}

func NewUpdate() *Update {
	return &Update{
		Attrs:    []xml.Attr{},
		AttrsMap: make(map[string]string),
		Sql:      make([]*Sql, 0, 5),
	}
}

func (m *Update) Evaluate(ctx context.Context, input *HandlerPayload) (string, error) {
	return intervalEvaluate(ctx, m.Children, input)
}

func (m *Update) Bind(ctx context.Context, input *HandlerPayload) *BindVar {
	return bindParamsToVar(ctx, m, m.AttrsMap, input)
}

func (m *Update) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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
