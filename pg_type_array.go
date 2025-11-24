package gobatis

import (
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
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
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
	return errors.New(`gobatis: value not int[]`)
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
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
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
	return errors.New(`gobatis: value not float[]`)
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

var (
	ArrayStringReplacer        = strings.NewReplacer(`\"`, `"`, `\\`, `\`)
	ArrayStringReverseReplacer = strings.NewReplacer(`"`, `\"`)
)

// Scan sql/database Scan interface
func (pg *PgArrayString) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
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
					if strings.HasSuffix(builder.String(), `\`) {
						builder.WriteString(innerText)
						continue
					}
					*pg = append(*pg, ArrayStringReplacer.Replace(builder.String()))
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
	return errors.New(`gobatis: value not text/varchar/char[]`)
}

// Value sql/database Value interface
func (pg *PgArrayString) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`{`)
	for i, v := range *pg {
		v = ArrayStringReverseReplacer.Replace(v)
		if bytes.IndexAny([]byte(v), `,"`) != -1 {
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
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
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
	return errors.New(`gobatis: value not bool[]`)
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


// PgVectorFloat postgresql array float
// Prices []float64
// which Prices was PgArrayFloat type
type PgVectorFloat []float64

// Scan sql/database Scan interface
func (pg *PgVectorFloat) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}

	scan.Split(SplitPgVectorType)
	for scan.Scan() {
		text := scan.Text()
		if text != `[` {
			continue
		}

		for scan.Scan() {
			text = scan.Text()
			if text == `,` {
				continue
			}
			if text == `]` {
				break
			}

			num, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return err
			}
			*pg = append(*pg, num)
		}

		if text == `]` {
			return nil
		}
	}
	return errors.New(`gobatis: value not float[]`)
}

// Value sql/database Value interface
func (pg *PgVectorFloat) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(len(*pg) * 3)
	b.WriteString(`[`)
	for i, v := range *pg {
		b.WriteString(strconv.FormatFloat(v, 'f', -2, 64))
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`]`)
	return b.String(), nil
}

