package gobatis

import (
	"bufio"
	"bytes"
	"database/sql/driver"
	"errors"
	"strings"
)

// PgRecord postgres record type
// caller should use this method as inner value to parse value into field.
// like this:
//
//	type SomeRecord struct {
//		value gobatis.Record
//		FieldOne string `pg:"field_one"`
//		FieldTwo string `pg:"field_two"`
//	}
//
//	func (sr *SomeRecord) Scan(value any) error {
//		err := sr.value.scan(value)
//		if err != nil {
//		  return err
//		}
//		sr.FieldOne = sr.value[0]
//		sr.FieldTwo = sr.value[1]
//		return nil
//	}
//
//	func (sr *SomeRecord) Value() (driver.Value, error) {
//		sr.value = nil
//		sr.value = append(sr.value, sr.FieldOne, sr.FieldTwo)
//	    return sr.value.Value()
//	}
type PgRecord []string

// Scan sql/database Scan interface
func (pg *PgRecord) Scan(value any) error {
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	default:
		return errors.New(`gobatis: not supported type`)
	}

	var builder strings.Builder
	builder.Grow(32)
	scan.Split(SplitPgRecordType)
	for scan.Scan() {
		text := scan.Text()
		if text != `(` {
			continue
		}

		for scan.Scan() {
			text = scan.Text()

			if text == `"` {
				builder.Reset()
				for scan.Scan() {
					innserText := scan.Text()
					if innserText != `"` {
						builder.WriteString(innserText)
						continue
					}
					*pg = append(*pg, builder.String())
					break
				}
				continue
			}

			if text == `,` {
				continue
			}
			if text == `)` {
				break
			}
			*pg = append(*pg, text)
		}

		if text == `)` {
			return nil
		}
	}
	return errors.New(`gobatis: value not array type`)
}

// Value sql/database Value interface
func (pg *PgRecord) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`(`)
	for i, v := range *pg {
		if strings.Contains(v, `,`) {
			v = `"` + v + `"`
		}
		b.WriteString(v)
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`)`)
	return b.String(), nil
}
