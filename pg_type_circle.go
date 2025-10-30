package gobatis

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

// PgCircle postgres circle type
type PgCircle struct {
	Center *PgPoint
	Radius float64
}

// Scan sql/database Scan interface
func (pg *PgCircle) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgCircleType)

	for scan.Scan() {
		text := scan.Text()
		if text != `<` {
			continue
		}

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
			} else if incr == 1 {
				if p.Y, err = AsFloat(text); err != nil {
					return err
				}
				break
			}
			incr++
		}

		pg.Center = p
		for scan.Scan() {
			text = scan.Text()
			if text == `)` {
				break
			}
		}

		for scan.Scan() {
			text = scan.Text()
			if text == `,` {
				break
			}
		}

		if !scan.Scan() {
			goto errorNotValidPgCircle
		}
		if pg.Radius, err = AsFloat(scan.Text()); err != nil {
			return err
		}

		if !scan.Scan() {
			goto errorNotValidPgCircle
		}
		if text := scan.Text(); text != `>` {
			goto errorNotValidPgCircle
		}
		return nil
	}

errorNotValidPgCircle:
	return errors.New(`gobatis: value not circle`)
}

// Value sql/database Value interface
func (pg *PgCircle) Value() (driver.Value, error) {
	b := strings.Builder{}
	b.Grow(32)

	b.WriteString(`<(`)
	b.WriteString(strconv.FormatFloat(pg.Center.X, 'f', -1, 64))
	b.WriteString(`,`)
	b.WriteString(strconv.FormatFloat(pg.Center.Y, 'f', -1, 64))
	b.WriteString(`),`)
	b.WriteString(strconv.FormatFloat(pg.Radius, 'f', -1, 64))
	b.WriteString(`>`)

	return b.String(), nil
}

// PgArrayCircle postgres circle[] type
type PgArrayCircle []*PgCircle

// Scan sql/database Scan interface
func (pg *PgArrayCircle) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgArrayCircleType)

	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}
	newCircle:
		for scan.Scan() {
			text = scan.Text()
			if text != `"` {
				continue
			}

			circle := &PgCircle{}
			for scan.Scan() {
				text := scan.Text()
				if text != `<` {
					continue
				}

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
					} else if incr == 1 {
						if p.Y, err = AsFloat(text); err != nil {
							return err
						}
						break
					}
					incr++
				}

				circle.Center = p
				for scan.Scan() {
					text = scan.Text()
					if text == `)` {
						break
					}
				}

				for scan.Scan() {
					text = scan.Text()
					if text == `,` {
						break
					}
				}

				if !scan.Scan() {
					goto errorNotValidPgArrayCircle
				}
				if circle.Radius, err = AsFloat(scan.Text()); err != nil {
					return err
				}

				if !scan.Scan() {
					goto errorNotValidPgArrayCircle
				}
				if text := scan.Text(); text != `>` {
					goto errorNotValidPgArrayCircle
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
						*pg = append(*pg, circle)
						goto newCircle
					case `}`:
						*pg = append(*pg, circle)
						return nil
					default:
						goto errorNotValidPgArrayCircle
					}
				}
			}
		}
	}

errorNotValidPgArrayCircle:
	return errors.New(`gobatis: value not circle[]`)
}

// Value sql/database Value interface
func (pg *PgArrayCircle) Value() (driver.Value, error) {
	b := strings.Builder{}
	b.Grow(64)

	b.WriteString(`{`)

	for i, circle := range *pg {
		b.WriteString(`"`)

		b.WriteString(`<(`)
		b.WriteString(strconv.FormatFloat(circle.Center.X, 'f', -1, 64))
		b.WriteString(`,`)
		b.WriteString(strconv.FormatFloat(circle.Center.Y, 'f', -1, 64))
		b.WriteString(`),`)
		b.WriteString(strconv.FormatFloat(circle.Radius, 'f', -1, 64))
		b.WriteString(`>`)

		b.WriteString(`"`)
		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}

	b.WriteString(`}`)
	return b.String(), nil
}
