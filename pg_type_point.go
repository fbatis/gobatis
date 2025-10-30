package gobatis

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

// PgPoint postgres point type
type PgPoint struct {
	X float64
	Y float64
}

// Scan sql/database Scan interface
func (pg *PgPoint) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgPointType)

	for scan.Scan() {
		// begin
		text := scan.Text()
		if text != `(` {
			goto errorNotPoint
		}

		if !scan.Scan() {
			goto errorNotPoint
		}
		text = scan.Text()
		// x point
		pg.X, err = strconv.ParseFloat(text, 64)
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
		// y point
		pg.Y, err = strconv.ParseFloat(text, 64)
		if err != nil {
			return err
		}

		// end
		if !scan.Scan() {
			goto errorNotPoint
		}
		text = scan.Text()

		if text != `)` {
			goto errorNotPoint
		}
		return nil
	}
errorNotPoint:
	return errors.New(`gobatis: value not point`)
}

// Value sql/database Value interface
func (pg *PgPoint) Value() (driver.Value, error) {
	var b strings.Builder
	b.WriteString(`(`)
	b.WriteString(strconv.FormatFloat(pg.X, 'f', -1, 64))
	b.WriteString(`,`)
	b.WriteString(strconv.FormatFloat(pg.Y, 'f', -1, 64))
	b.WriteString(`)`)
	return b.String(), nil
}

// PgArrayPoint postgres point[] type
type PgArrayPoint []*PgPoint

// Scan sql/database Scan interface
func (pg *PgArrayPoint) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgArrayPathType)

	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}
	newPoint:
		for scan.Scan() {
			text = scan.Text()
			if text != `"` {
				continue
			}

			p := &PgPoint{}
			for scan.Scan() {
				// begin
				text := scan.Text()
				if text != `(` {
					goto errorNotValidPgArrayPoint
				}

				if !scan.Scan() {
					goto errorNotValidPgArrayPoint
				}
				text = scan.Text()
				// x point
				p.X, err = strconv.ParseFloat(text, 64)
				if err != nil {
					return err
				}

				// , separator
				if !scan.Scan() {
					goto errorNotValidPgArrayPoint
				}
				text = scan.Text()
				if text != `,` {
					goto errorNotValidPgArrayPoint
				}

				if !scan.Scan() {
					goto errorNotValidPgArrayPoint
				}
				text = scan.Text()
				// y point
				p.Y, err = strconv.ParseFloat(text, 64)
				if err != nil {
					return err
				}

				// end
				if !scan.Scan() {
					goto errorNotValidPgArrayPoint
				}
				text = scan.Text()

				if text != `)` {
					goto errorNotValidPgArrayPoint
				}

				for scan.Scan() {
					text = scan.Text()
					if text == `"` {
						break
					}
				}

				for scan.Scan() {
					switch scan.Text() {
					case `,`:
						*pg = append(*pg, p)
						goto newPoint
					case `}`:
						*pg = append(*pg, p)
						return nil
					default:
						goto errorNotValidPgArrayPoint
					}
				}
			}
		}
	}

errorNotValidPgArrayPoint:
	return errors.New(`gobatis: value not point[]`)
}

// Value sql/database Value interface
func (pg *PgArrayPoint) Value() (driver.Value, error) {
	b := strings.Builder{}
	b.Grow(32)
	b.WriteString(`{`)
	for i, p := range *pg {
		b.WriteString(`"`)
		b.WriteString(`(`)
		b.WriteString(strconv.FormatFloat(p.X, 'f', -1, 64))
		b.WriteString(`,`)
		b.WriteString(strconv.FormatFloat(p.Y, 'f', -1, 64))
		b.WriteString(`)`)
		b.WriteString(`"`)
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`}`)
	return b.String(), nil
}
