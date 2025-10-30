package gobatis

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

// PgPolygon postgres polygon type
type PgPolygon []*PgPoint

// Scan sql/database Scan interface
func (pg *PgPolygon) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgPointType)

	for scan.Scan() {
		text := scan.Text()
		if text != `(` {
			continue
		}
	newPoint:
		for scan.Scan() {
			text = scan.Text()
			if text == `(` {
				break
			}
		}

		incr := 0
		p := &PgPoint{}
		for scan.Scan() {
			text = scan.Text()
			if text == `,` {
				continue
			}
			if incr == 0 {
				if p.X, err = AsFloat(text); err != nil {
					return err
				}
			} else {
				if p.Y, err = AsFloat(text); err != nil {
					return nil
				}
				break
			}
			incr++
		}

		for scan.Scan() {
			text = scan.Text()
			if text == `)` {
				break
			}
		}

		for scan.Scan() {
			switch scan.Text() {
			case `,`:
				*pg = append(*pg, p)
				goto newPoint
			case `)`:
				*pg = append(*pg, p)
				return nil
			default:
				goto errorNotValidPolygon
			}
		}
	}
errorNotValidPolygon:
	return errors.New(`gobatis: value not polygon`)
}

// Value sql/database Value interface
func (pg *PgPolygon) Value() (driver.Value, error) {
	b := strings.Builder{}
	b.Grow(64)

	b.WriteString(`(`)
	for i, point := range *pg {

		b.WriteString(`(`)
		b.WriteString(strconv.FormatFloat(point.X, 'f', -1, 64))
		b.WriteString(`,`)
		b.WriteString(strconv.FormatFloat(point.Y, 'f', -1, 64))
		b.WriteString(`)`)

		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}
	b.WriteString(`)`)
	return b.String(), nil
}

// PgArrayPolygon postgres polygon[] type
type PgArrayPolygon []*PgPolygon

// Scan sql/database Scan interface
func (pg *PgArrayPolygon) Scan(value any) error {
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

	newPolygon:
		polygon := PgPolygon{}
		for scan.Scan() {
			text = scan.Text()
			if text != `"` {
				continue
			}

			for scan.Scan() {
				text := scan.Text()
				if text != `(` {
					continue
				}
			newPoint:
				for scan.Scan() {
					text = scan.Text()
					if text == `(` {
						break
					}
				}

				incr := 0
				p := &PgPoint{}
				for scan.Scan() {
					text = scan.Text()
					if text == `,` {
						continue
					}
					if incr == 0 {
						if p.X, err = AsFloat(text); err != nil {
							return err
						}
					} else {
						if p.Y, err = AsFloat(text); err != nil {
							return nil
						}
						break
					}
					incr++
				}

				for scan.Scan() {
					text = scan.Text()
					if text == `)` {
						break
					}
				}

				for scan.Scan() {
					switch scan.Text() {
					case `,`:
						polygon = append(polygon, p)
						goto newPoint
					case `)`:
						polygon = append(polygon, p)
						goto endOrNext
					default:
						goto errorNotValidPolygon
					}
				}
			endOrNext:
				for scan.Scan() {
					text = scan.Text()
					if text == `"` {
						continue
					}
					if text == `,` {
						*pg = append(*pg, &polygon)
						goto newPolygon
					}
					if text == `}` {
						*pg = append(*pg, &polygon)
						return nil
					}
				}
				goto errorNotValidPolygon
			}
		}
	}

errorNotValidPolygon:
	return errors.New(`gobatis: value not polygon[]`)
}

// Value sql/database Value interface
func (pg *PgArrayPolygon) Value() (driver.Value, error) {
	b := strings.Builder{}
	b.Grow(64)

	b.WriteString(`{`)

	for i, polygon := range *pg {

		b.WriteString(`"`)
		b.WriteString(`(`)
		for i, point := range *polygon {

			b.WriteString(`(`)
			b.WriteString(strconv.FormatFloat(point.X, 'f', -1, 64))
			b.WriteString(`,`)
			b.WriteString(strconv.FormatFloat(point.Y, 'f', -1, 64))
			b.WriteString(`)`)

			if i != len(*polygon)-1 {
				b.WriteString(`,`)
			}
		}
		b.WriteString(`)`)
		b.WriteString(`"`)

		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}

	b.WriteString(`}`)

	return b.String(), nil
}
