package gobatis

import (
	"bufio"
	"bytes"
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

// PgArrayInt postgresql array int
// Ids []int
// which Ids was PgArrayInt type
type PgArrayInt []int64

// Scan sql/database Scan interface
func (pg *PgArrayInt) Scan(value any) error {
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	default:
		return errors.New(`gobatis: not supported type`)
	}

	scan.Split(SplitPgArrayType)
	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}

		for scan.Scan() {
			text = scan.Text()
			if text == `,` {
				continue
			}
			if text == `}` {
				break
			}

			num, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return err
			}
			*pg = append(*pg, num)
		}

		if text == `}` {
			return nil
		}
	}
	return errors.New(`gobatis: value not array type`)
}

// Value sql/database Value interface
func (pg *PgArrayInt) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`{`)
	for i, v := range *pg {
		b.WriteString(strconv.FormatInt(v, 10))
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`}`)
	return b.String(), nil
}

// PgArrayFloat postgresql array float
// Prices []float64
// which Prices was PgArrayFloat type
type PgArrayFloat []float64

// Scan sql/database Scan interface
func (pg *PgArrayFloat) Scan(value any) error {
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	default:
		return errors.New(`gobatis: not supported type`)
	}

	scan.Split(SplitPgArrayType)
	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}

		for scan.Scan() {
			text = scan.Text()
			if text == `,` {
				continue
			}
			if text == `}` {
				break
			}

			num, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return err
			}
			*pg = append(*pg, num)
		}

		if text == `}` {
			return nil
		}
	}
	return errors.New(`gobatis: value not array type`)
}

// Value sql/database Value interface
func (pg *PgArrayFloat) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`{`)
	for i, v := range *pg {
		b.WriteString(strconv.FormatFloat(v, 'f', -2, 64))
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`}`)
	return b.String(), nil
}

// PgArrayString postgresql array string
// Names []text
// which Names was PgArrayString type
type PgArrayString []string

// Scan sql/database Scan interface
func (pg *PgArrayString) Scan(value any) error {
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	default:
		return errors.New(`gobatis: not supported type`)
	}

	scan.Split(SplitPgArrayStringType)
	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}

		var builder = strings.Builder{}
		builder.Grow(64)

		for scan.Scan() {
			text = scan.Text()

			if text == `"` {
				builder.Reset()
				for scan.Scan() {
					innerText := scan.Text()
					if innerText != `"` {
						builder.WriteString(innerText)
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

			if text == `}` {
				break
			}

			*pg = append(*pg, text)
		}

		if text == `}` {
			return nil
		}
	}
	return errors.New(`gobatis: value not array type`)
}

// Value sql/database Value interface
func (pg *PgArrayString) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`{`)
	for i, v := range *pg {
		if strings.Contains(v, `,`) {
			v = `"` + v + `"`
		}
		b.WriteString(v)
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`}`)
	return b.String(), nil
}

// PgArrayBool postgresql array bool
// Exists []bool
// which Exists was PgArrayBool type
type PgArrayBool []bool

// Scan sql/database Scan interface
func (pg *PgArrayBool) Scan(value any) error {
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	default:
		return errors.New(`gobatis: not supported type`)
	}

	scan.Split(SplitPgArrayType)
	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}

		for scan.Scan() {
			text = scan.Text()
			if text == `,` {
				continue
			}
			if text == `}` {
				break
			}

			switch text {
			case `t`, `true`:
				*pg = append(*pg, true)
			case `f`, `false`:
				*pg = append(*pg, false)
			default:
				goto errorNotArrayBoolType
			}
		}

		if text == `}` {
			return nil
		}
	}
errorNotArrayBoolType:
	return errors.New(`gobatis: value not array type`)
}

// Value sql/database Value interface
func (pg *PgArrayBool) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`{`)
	for i, v := range *pg {
		switch v {
		case true:
			b.WriteString(`t`)
		case false:
			b.WriteString(`f`)
		}
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`}`)
	return b.String(), nil
}

// PgArrayRecord postgresql array record
type PgArrayRecord [][]string

// Scan sql/database Scan interface
func (pg *PgArrayRecord) Scan(value any) error {
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	default:
		return errors.New(`gobatis: not supported type`)
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

		nextItem:
			for scan.Scan() {
				text = scan.Text()

				if text == `\` {
					detail.Reset()
					for scan.Scan() {
						text = scan.Text()
						if text == `"` {
							if !strings.HasSuffix(detail.String(), `\`) {
								continue
							}
							recordItem = append(recordItem, detail.String()[:len(detail.String())-1])
							goto nextItem
						}
						detail.WriteString(text)
					}
				}

				if text == `,` {
					continue
				}
				if text == `)` {
					break
				}
				recordItem = append(recordItem, text)
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
	return errors.New(`gobatis: value not record[] type`)
}

// Value sql/database Value interface
func (pg *PgArrayRecord) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(64)
	b.WriteString(`{`)

	for i, v := range *pg {
		b.WriteString(`"(`)
		for j, item := range v {
			if strings.Contains(item, `,`) {
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
