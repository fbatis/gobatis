package gobatis

import (
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
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
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
	return errors.New(`gobatis: value not record`)
}

// Value sql/database Value interface
func (pg *PgRecord) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`(`)
	for i, v := range *pg {
		v = RecordReverseReplacer.Replace(v)
		if bytes.IndexAny([]byte(v), `, ()\"`) != -1 {
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

// PgArrayRecord postgresql array record
type PgArrayRecord [][]string

var (
	ArrayRecordReplacer        = strings.NewReplacer(`\"\"`, `"`, `\",`, ``, `\")`, ``, `\\\\`, `\`)
	ArrayRecordReverseReplacer = strings.NewReplacer(`"`, `\"\"`, `\`, `\\\\`)
)

// Scan sql/database Scan interface
func (pg *PgArrayRecord) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}

	scan.Split(SplitPgArrayRecordType)
	var recordItem []string
	var detail strings.Builder

	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			goto errorRecordArrayType
		}

	nextRecord:
		recordItem = nil
		for scan.Scan() {
			text = scan.Text()
			if text != `"` {
				continue
			}

			if !scan.Scan() {
				goto errorRecordArrayType
			}
			text = scan.Text()
			if text != `(` {
				goto errorRecordArrayType
			}

			for scan.Scan() {
				text = scan.Text()

				if text == `\` {
					detail.Reset()
					for scan.Scan() {
						text = scan.Text()
						if text != `"` {
							continue
						}

						// text body
						for scan.Scan() {
							text = scan.Text()
							detail.WriteString(text)

							if strings.HasSuffix(detail.String(), `\",`) {
								goto addItem
							} else if strings.HasSuffix(detail.String(), `\")`) {
								goto addItem
							}
						}
						goto errorRecordArrayType
					}
				}
			addItem:
				if text == `,` {
					recordItem = append(recordItem, ArrayRecordReplacer.Replace(detail.String()))
					detail.Reset()
					continue
				}
				if text == `)` {
					recordItem = append(recordItem, ArrayRecordReplacer.Replace(detail.String()))
					detail.Reset()
					break
				}

				detail.WriteString(text)
			}

			for scan.Scan() {
				text = scan.Text()
				if text == `"` {
					continue
				}
				if text == `,` {
					*pg = append(*pg, recordItem)
					goto nextRecord
				}
				if text == `}` {
					break
				}
			}
		}

		if text == `}` {
			*pg = append(*pg, recordItem)
			return nil
		}
	}
errorRecordArrayType:
	return errors.New(`gobatis: value not record[]`)
}

// Value sql/database Value interface
func (pg *PgArrayRecord) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(64)
	b.WriteString(`{`)

	for i, v := range *pg {
		b.WriteString(`"(`)
		for j, item := range v {
			item = ArrayRecordReverseReplacer.Replace(item)
			if bytes.IndexAny([]byte(item), `, ()\"`) != -1 {
				item = `\"` + item + `\"`
			}
			b.WriteString(item)
			if j != len(v)-1 {
				b.WriteString(`,`)
			}
		}
		b.WriteString(`)"`)
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}

	b.WriteString(`}`)
	return b.String(), nil
}
