package gobatis

import (
	"bytes"
	"database/sql/driver"
	"errors"
	"strings"
)

// ROW(2, '"\test', array['"\12345678901', '12345678902\\"'])
// (2,"""\\test","{""\\""\\\\12345678901"",""12345678902\\\\\\\\\\""""}", "[aaaa,bbb)")
var (
	PgNewRecordReplacer           = strings.NewReplacer(`\\`, `\`)
	PgNewArrayRecordReplacer      = strings.NewReplacer(`\\\\`, `\`, `\\\\\\\\`, `\`)
	PgNewRecordInnerReplacer      = strings.NewReplacer(`""`, `"`, `\\\\`, `\`, `\\`, ``)
	PgNewArrayRecordInnerReplacer = strings.NewReplacer(`\\\\\"\"`, `"`, `\\\\`, `\`, `\\`, ``)
	DoubleQuoteReplacer           = strings.NewReplacer(`""`, `"`)
	SlashDoubleQuoteReplacer      = strings.NewReplacer(`\"\"`, `"`, `\\`, `\`)

	RecordReverseReplacer      = strings.NewReplacer(`"`, `""`, `\`, `\\`)
	ArrayRecordReverseReplacer = strings.NewReplacer(`"`, `\\\\\"\"`, `\`, `\\\\`)
)

// PgRecord postgres record type
// caller should use this method as inner value to parse value into field.
// each field should be string, array, record, range type.
// user should convert to the real type.
//
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
	scan, err := fetchScanner(value)
	if err != nil || scan == nil {
		return err
	}
	scan.Split(SplitPgRecordType)

	for scan.Scan() {
		text := scan.Text()
		if text != `(` {
			continue
		}
	nextField:
		for scan.Scan() {
			text = scan.Text()

			field := strings.Builder{}
			field.Grow(32)
			if text == `"` {
				// inner text
				// in inner text, " should be escaped with double quote
				// in this mode, we can meet array or range type, also the normal string type values
				for scan.Scan() {
					text = scan.Text()
					switch text {
					case `{`: // array of values
						field.WriteString(text)
						for scan.Scan() {
							text = scan.Text()
							field.WriteString(text)
							if strings.HasSuffix(field.String(), `}"`) {
								for scan.Scan() {
									text = scan.Text()
									switch text {
									case `,`:
										*pg = append(*pg, DoubleQuoteReplacer.Replace(PgNewRecordInnerReplacer.Replace(field.String()[:field.Len()-1])))
										goto nextField
									case `)`:
										*pg = append(*pg, DoubleQuoteReplacer.Replace(PgNewRecordInnerReplacer.Replace(field.String()[:field.Len()-1])))
										return nil
									default:
										continue
									}
								}
								goto errorInvalidPgTypeRecord
							}
						}
						goto errorInvalidPgTypeRecord
					case `[`: // range of values
						fallthrough
					case `(`: // range of but also record type values
						field.WriteString(text)
						for scan.Scan() {
							text = scan.Text()
							field.WriteString(text)
							if strings.HasSuffix(field.String(), `]"`) ||
								strings.HasSuffix(field.String(), `)"`) {
								for scan.Scan() {
									text = scan.Text()
									switch text {
									case `,`:
										*pg = append(*pg, DoubleQuoteReplacer.Replace(PgNewRecordInnerReplacer.Replace(field.String()[:field.Len()-1])))
										goto nextField
									case `)`:
										*pg = append(*pg, DoubleQuoteReplacer.Replace(PgNewRecordInnerReplacer.Replace(field.String()[:field.Len()-1])))
										return nil
									default:
										continue
									}
								}
								goto errorInvalidPgTypeRecord
							}
						}
						goto errorInvalidPgTypeRecord
					default:
						for scan.Scan() {
							text = scan.Text()
							switch text {
							case `,`:
								if strings.HasSuffix(field.String(), `"`) {
									*pg = append(*pg, PgNewRecordReplacer.Replace(field.String()[:field.Len()-1]))
									goto nextField
								}
							case `)`:
								if strings.HasSuffix(field.String(), `"`) {
									*pg = append(*pg, PgNewRecordReplacer.Replace(field.String()[:field.Len()-1]))
									return nil
								}
							}
							field.WriteString(text)
						}
						goto errorInvalidPgTypeRecord
					}
				}
				goto errorInvalidPgTypeRecord
			} else {
				// row value
				field.WriteString(text)
			}

			// we need next string is , or )
			for scan.Scan() {
				text = scan.Text()
				switch text {
				case `,`:
					*pg = append(*pg, field.String())
					goto nextField
				case `)`:
					*pg = append(*pg, field.String())
					return nil
				default:
					continue
				}
			}
			goto errorInvalidPgTypeRecord
		}
	}

errorInvalidPgTypeRecord:
	return errors.New(`gobatis: values not valid record type or can't parse correctly.'`)
}

// Value sql/database Value interface
func (pg *PgRecord) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`(`)
	for i, v := range *pg {
		v = RecordReverseReplacer.Replace(v)
		if bytes.IndexAny([]byte(v), `, (){}[]\"`) != -1 {
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

// array[
//  row(1, 'test\\', array['12345678901\\', '12345678902']),
//  row(1, 'test', array['12345678901', '12345678902'])
//  ]::address[]
// {
//  "(1,\"test\\\\\\\\\",\"{\"\"12345678901\\\\\\\\\\\\\\\\\"\",12345678902}\")",
//  "(1,test,\"{12345678901,12345678902}\")"
// }

type PgArrayRecord []PgRecord

// Scan sql/database Scan interface
func (pg *PgArrayRecord) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitByStringWithPrefix("{}(,\")", []string{`\"`}))

	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}
	nextRecord:
		for scan.Scan() {
			text = scan.Text()
			if text != `"` {
				continue
			}

			record := PgRecord{}
			for scan.Scan() {
				text = scan.Text()
				if text != `(` {
					continue
				}

			nextField:
				for scan.Scan() {
					text = scan.Text()

					field := strings.Builder{}
					field.Grow(32)
					if text == `\"` {
						// inner text
						// in inner text, " should be escaped with double-double quote escape
						for scan.Scan() {
							text = scan.Text()
							switch text {
							case `{`: // array of values
								field.WriteString(text)
								for scan.Scan() {
									text = scan.Text()
									field.WriteString(text)
									if strings.HasSuffix(field.String(), `}\"`) {
										for scan.Scan() {
											text = scan.Text()
											switch text {
											case `,`:
												record = append(record, SlashDoubleQuoteReplacer.Replace(PgNewArrayRecordInnerReplacer.Replace(field.String()[:field.Len()-2])))
												goto nextField
											case `)`:
												record = append(record, SlashDoubleQuoteReplacer.Replace(PgNewArrayRecordInnerReplacer.Replace(field.String()[:field.Len()-2])))
												goto endOrNext
											default:
												continue
											}
										}
										goto errorInvalidArrayRecordType
									}
								}
								goto errorInvalidArrayRecordType
							case `[`: // range of values
								fallthrough
							case `(`: // range of but also record type values
								field.WriteString(text)
								for scan.Scan() {
									text = scan.Text()
									field.WriteString(text)
									if strings.HasSuffix(field.String(), `]\"`) ||
										strings.HasSuffix(field.String(), `)\"`) {
										for scan.Scan() {
											text = scan.Text()
											switch text {
											case `,`:
												record = append(record, SlashDoubleQuoteReplacer.Replace(PgNewArrayRecordInnerReplacer.Replace(field.String()[:field.Len()-2])))
												goto nextField
											case `)`:
												record = append(record, SlashDoubleQuoteReplacer.Replace(PgNewArrayRecordInnerReplacer.Replace(field.String()[:field.Len()-2])))
												goto endOrNext
											default:
												continue
											}
										}
										goto errorInvalidArrayRecordType
									}
								}
								goto errorInvalidArrayRecordType
							default:
								field.WriteString(text)
								for scan.Scan() {
									text = scan.Text()
									switch text {
									case `,`:
										if strings.HasSuffix(field.String(), `\"`) {
											record = append(record, PgNewArrayRecordReplacer.Replace(field.String()[:field.Len()-2]))
											goto nextField
										}
									case `)`:
										if strings.HasSuffix(field.String(), `\"`) {
											record = append(record, PgNewArrayRecordReplacer.Replace(field.String()[:field.Len()-2]))
											goto endOrNext
										}
									}
									field.WriteString(text)
								}
								goto errorInvalidArrayRecordType
							}
						}
						goto errorInvalidArrayRecordType
					} else {
						// row value
						field.WriteString(text)
					}

					// we need next string is , or )
					for scan.Scan() {
						text = scan.Text()
						switch text {
						case `,`:
							record = append(record, PgNewArrayRecordReplacer.Replace(field.String()))
							goto nextField
						case `)`:
							record = append(record, PgNewArrayRecordReplacer.Replace(field.String()))
							goto endOrNext
						default:
							continue
						}
					}

				endOrNext:
					for scan.Scan() {
						text = scan.Text()
						if text != `"` {
							continue
						}

						for scan.Scan() {
							text = scan.Text()
							switch text {
							case `,`:
								*pg = append(*pg, record)
								goto nextRecord
							case `}`:
								*pg = append(*pg, record)
								return nil
							default:
								continue
							}
						}
					}
					goto errorInvalidArrayRecordType
				}
			}

			goto errorInvalidArrayRecordType
		}
	}

errorInvalidArrayRecordType:
	return errors.New(`gobatis: values not valid array record type or can't parse correctly.'`)
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
			if bytes.IndexAny([]byte(item), `, (){}[]\"`) != -1 {
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
