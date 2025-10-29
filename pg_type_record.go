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

var (
	RecordReplacer        = strings.NewReplacer(`""`, `"`, `",`, ``, `")`, ``)
	RecordReverseReplacer = strings.NewReplacer(`"`, `""`)
)

// Scan sql/database Scan interface
func (pg *PgRecord) Scan(value any) error {
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	case nil:
		return nil
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
					text = scan.Text()
					builder.WriteString(text)
					if strings.HasSuffix(builder.String(), `",`) {
						*pg = append(*pg, RecordReplacer.Replace(builder.String()))
						break
					}
					if strings.HasSuffix(builder.String(), `")`) {
						*pg = append(*pg, RecordReplacer.Replace(builder.String()))
						break
					}
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
		v = RecordReverseReplacer.Replace(v)
		if bytes.IndexAny([]byte(v), `, "`) != -1 {
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
