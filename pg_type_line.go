package gobatis

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

// PgLine postgres line type
type PgLine struct {
	A, B, C float64
}

// Scan sql/database Scan interface
func (pg *PgLine) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgArrayType)

	for scan.Scan() {
		// begin
		text := scan.Text()
		if text != `{` {
			goto errorNotPoint
		}

		if !scan.Scan() {
			goto errorNotPoint
		}
		text = scan.Text()
		// x point
		pg.A, err = strconv.ParseFloat(text, 64)
		if err != nil {
			return err
		}

		// , separator
		if !scan.Scan() {
			goto errorNotPoint
		}
		text = scan.Text()
		if text != `,` {
			goto errorNotPoint
		}

		if !scan.Scan() {
			goto errorNotPoint
		}
		text = scan.Text()
		// B point
		pg.B, err = strconv.ParseFloat(text, 64)
		if err != nil {
			return err
		}

		// , separator
		if !scan.Scan() {
			goto errorNotPoint
		}
		text = scan.Text()
		if text != `,` {
			goto errorNotPoint
		}

		if !scan.Scan() {
			goto errorNotPoint
		}
		text = scan.Text()
		// C point
		pg.C, err = strconv.ParseFloat(text, 64)
		if err != nil {
			return err
		}

		// end
		if !scan.Scan() {
			goto errorNotPoint
		}
		text = scan.Text()

		if text != `}` {
			goto errorNotPoint
		}
		return nil
	}
errorNotPoint:
	return errors.New(`gobatis: value not line`)
}

// Value sql/database Value interface
func (pg *PgLine) Value() (driver.Value, error) {
	var b strings.Builder
	b.WriteString(`{`)
	b.WriteString(strconv.FormatFloat(pg.A, 'f', -1, 64))
	b.WriteString(`,`)
	b.WriteString(strconv.FormatFloat(pg.B, 'f', -1, 64))
	b.WriteString(`,`)
	b.WriteString(strconv.FormatFloat(pg.C, 'f', -1, 64))
	b.WriteString(`}`)
	return b.String(), nil
}

// PgArrayLine postgres array line type
type PgArrayLine []*PgLine

// Scan sql/database Scan interface
func (pg *PgArrayLine) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgArrayLineType)

	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}
	parseLine:
		for scan.Scan() {
			text = scan.Text()
			if text != `"` {
				continue
			}

			line := &PgLine{}
			for scan.Scan() {
				text = scan.Text()
				if text != `{` {
					continue
				}

				incr := 0
				for scan.Scan() {
					text = scan.Text()
					if text == `,` {
						continue
					}
					if incr == 0 {
						if line.A, err = strconv.ParseFloat(text, 64); err != nil {
							return err
						}
					} else if incr == 1 {
						if line.B, err = strconv.ParseFloat(text, 64); err != nil {
							return err
						}
					} else if incr == 2 {
						if line.C, err = strconv.ParseFloat(text, 64); err != nil {
							return err
						}
						break
					}
					incr++
				}

				if !scan.Scan() {
					goto errorNotLine
				}
				text = scan.Text()
				if text == `}` {
					break
				}
				goto errorNotLine
			}

			if !scan.Scan() {
				goto errorNotLine
			}
			text = scan.Text()
			if text != `"` {
				goto errorNotLine
			}

			if !scan.Scan() {
				goto errorNotLine
			}
			text = scan.Text()
			if text == `,` {
				*pg = append(*pg, line)
				goto parseLine
			}
			if text == `}` {
				*pg = append(*pg, line)
				return nil
			}
		}

	}
errorNotLine:
	return errors.New(`gobatis: value not line[]`)
}

// Value sql/database Value interface
func (pg *PgArrayLine) Value() (driver.Value, error) {
	var builder strings.Builder
	builder.WriteString(`{`)

	for i, line := range *pg {
		builder.WriteString(`"{`)

		builder.WriteString(strconv.FormatFloat(line.A, 'f', -1, 64))
		builder.WriteString(`,`)
		builder.WriteString(strconv.FormatFloat(line.B, 'f', -1, 64))
		builder.WriteString(`,`)
		builder.WriteString(strconv.FormatFloat(line.C, 'f', -1, 64))

		builder.WriteString(`}"`)
		if i != len(*pg)-1 {
			builder.WriteString(`,`)
		}
	}

	builder.WriteString(`}`)
	return builder.String(), nil
}
