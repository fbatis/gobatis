package gobatis

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

// PgBox postgres box type
type PgBox struct {
	RightTop   *PgPoint
	LeftBottom *PgPoint
}

// Scan sql/database Scan interface
func (pg *PgBox) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgPointType)

newPoint:
	for scan.Scan() {
		text := scan.Text()
		if text != `(` {
			continue
		}

		// x, y
		pt := &PgPoint{}
		incr := 0
		for scan.Scan() {
			text = scan.Text()
			if text == `,` {
				continue
			}
			if incr == 0 {
				if pt.X, err = strconv.ParseFloat(text, 64); err != nil {
					return err
				}
			} else if incr == 1 {
				if pt.Y, err = strconv.ParseFloat(text, 64); err != nil {
					return err
				}
				break
			}
			incr++
		}

		// )
		for scan.Scan() {
			if scan.Text() == `)` {
				break
			}
		}

		// ,
		for scan.Scan() {
			if scan.Text() == `,` {
				pg.RightTop = pt
				goto newPoint
			}
		}
		if !scan.Scan() {
			pg.LeftBottom = pt
			return nil
		}
		goto errorNotValidPgBox
	}

errorNotValidPgBox:
	return errors.New(`gobatis: value not box`)
}

// Value sql/database Value interface
func (pg *PgBox) Value() (driver.Value, error) {
	builder := strings.Builder{}
	var pts []*PgPoint
	if pg.RightTop != nil {
		pts = append(pts, pg.RightTop)
	}
	if pg.LeftBottom != nil {
		pts = append(pts, pg.LeftBottom)
	}

	for i, pt := range pts {
		builder.WriteString(`(`)
		builder.WriteString(strconv.FormatFloat(pt.X, 'f', -1, 64))
		builder.WriteString(`,`)
		builder.WriteString(strconv.FormatFloat(pt.Y, 'f', -1, 64))
		builder.WriteString(`)`)

		if i != len(pts)-1 {
			builder.WriteString(`,`)
		}
	}

	return builder.String(), nil
}

// PgArrayBox postgres array box type
type PgArrayBox []*PgBox

// Scan sql/database Scan interface
func (pg *PgArrayBox) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgArrayBoxType)

	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}
	newBox:
		box := &PgBox{}
	newPoint:
		for scan.Scan() {
			text := scan.Text()
			if text != `(` {
				continue
			}

			// x, y
			pt := &PgPoint{}
			incr := 0
			for scan.Scan() {
				text = scan.Text()
				if text == `,` {
					continue
				}
				if incr == 0 {
					if pt.X, err = strconv.ParseFloat(text, 64); err != nil {
						return err
					}
				} else if incr == 1 {
					if pt.Y, err = strconv.ParseFloat(text, 64); err != nil {
						return err
					}
					break
				}
				incr++
			}

			// )
			for scan.Scan() {
				if scan.Text() == `)` {
					break
				}
			}

			// ,
			for scan.Scan() {
				text = scan.Text()
				if text == `,` {
					box.RightTop = pt
					goto newPoint
				}
				if text == `;` {
					box.LeftBottom = pt
					*pg = append(*pg, box)
					goto newBox
				}
				if text == `}` {
					box.LeftBottom = pt
					*pg = append(*pg, box)
					return nil
				}
			}
			goto errorNotValidPgArrayBox
		}

	}
errorNotValidPgArrayBox:
	return errors.New(`gobatis: value not box[]`)
}

// Value sql/database Value interface
func (pg *PgArrayBox) Value() (driver.Value, error) {
	builder := strings.Builder{}
	builder.Grow(64)
	builder.WriteString(`{`)

	for i, box := range *pg {

		var pts []*PgPoint
		if box.RightTop != nil {
			pts = append(pts, box.RightTop)
		}
		if box.LeftBottom != nil {
			pts = append(pts, box.LeftBottom)
		}

		for i, pt := range pts {
			builder.WriteString(`(`)
			builder.WriteString(strconv.FormatFloat(pt.X, 'f', -1, 64))
			builder.WriteString(`,`)
			builder.WriteString(strconv.FormatFloat(pt.Y, 'f', -1, 64))
			builder.WriteString(`)`)

			if i != len(pts)-1 {
				builder.WriteString(`,`)
			}
		}

		if i != len(*pg)-1 {
			builder.WriteString(`;`)
		}
	}

	builder.WriteString(`}`)
	return builder.String(), nil
}
