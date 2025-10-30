package gobatis

import (
	"bufio"
	"bytes"
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

func fetchScanner(value any) (*bufio.Scanner, error) {
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	case nil:
		return nil, nil
	default:
		return nil, errors.New(`gobatis: not supported type`)
	}
	return scan, nil
}

// PgLSeg postgres lseg type
type PgLSeg struct {
	P1, P2 *PgPoint
}

// Scan sql/database Scan interface
func (pg *PgLSeg) Scan(value any) error {
	scan, err := fetchScanner(value)
	if err != nil || scan == nil {
		return err
	}

	scan.Split(SplitPgRangeType)

	px := 0
	for scan.Scan() {
		text := scan.Text()
		if text != "[" {
			continue
		}

	newPoint:
		// (x, y)
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
				if p.X, err = strconv.ParseFloat(text, 64); err != nil {
					return err
				}
			} else if incr == 1 {
				if p.Y, err = strconv.ParseFloat(text, 64); err != nil {
					return err
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
			text = scan.Text()
			if text == `]` {
				if px == 0 {
					pg.P1 = p
				} else if px == 1 {
					pg.P2 = p
				}
				return nil
			}
			if text != `,` {
				continue
			}
			if px == 0 {
				pg.P1 = p
			} else if px == 1 {
				pg.P2 = p
			}
			px++
			goto newPoint
		}

		goto errorNotLSegType
	}

errorNotLSegType:
	return errors.New(`gobatis: value not lseg`)
}

// Value sql/database Value interface
func (pg *PgLSeg) Value() (driver.Value, error) {
	var builder strings.Builder
	builder.WriteString(`[`)

	var pts []*PgPoint
	if pg.P1 != nil {
		pts = append(pts, pg.P1)
	}
	if pg.P2 != nil {
		pts = append(pts, pg.P2)
	}

	for i, point := range pts {

		builder.WriteString(`(`)
		builder.WriteString(strconv.FormatFloat(point.X, 'f', -1, 64))
		builder.WriteString(`,`)
		builder.WriteString(strconv.FormatFloat(point.Y, 'f', -1, 64))
		builder.WriteString(`)`)

		if i != len(pts)-1 {
			builder.WriteString(`,`)
		}
	}

	builder.WriteString(`]`)
	return builder.String(), nil
}

// PgArrayLSeg postgres array lseg type
type PgArrayLSeg []*PgLSeg

// Scan sql/database Scan interface
func (pg *PgArrayLSeg) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgArrayLsegType)

	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}

		// "[(1,2),(3,4)]"
		// each item format as previous
	newLSeg:
		for scan.Scan() {
			if scan.Text() == `"` {
				break
			}
		}

		for scan.Scan() {
			text = scan.Text()
			if text != `[` {
				continue
			}

			lseg := &PgLSeg{}
			px := 0
		newPoint:
			for scan.Scan() {
				text = scan.Text()
				if text == `(` {
					break
				}
			}

			incr := 0
			point := &PgPoint{}
			for scan.Scan() {
				text = scan.Text()
				if text == `,` {
					continue
				}
				if incr == 0 {
					if point.X, err = strconv.ParseFloat(text, 64); err != nil {
						return err
					}
				}
				if incr == 1 {
					if point.Y, err = strconv.ParseFloat(text, 64); err != nil {
						return err
					}
					break
				}
				incr++
			}

			for scan.Scan() {
				if scan.Text() == `)` {
					break
				}
			}

			if !scan.Scan() {
				goto errorNotPgArrayLSeg
			}
			text = scan.Text()
			if text == `,` {
				if px == 0 {
					lseg.P1 = point
				} else {
					lseg.P2 = point
				}
				px++
				goto newPoint
			}
			if text == `]` {
				if px == 0 {
					lseg.P1 = point
				} else {
					lseg.P2 = point
				}
				for scan.Scan() {
					if scan.Text() == `"` {
						break
					}
				}

				for scan.Scan() {
					if scan.Text() == `,` {
						*pg = append(*pg, lseg)
						goto newLSeg
					}
					if scan.Text() == `}` {
						*pg = append(*pg, lseg)
						return nil
					}
				}
			}
			goto errorNotPgArrayLSeg
		}

	}
errorNotPgArrayLSeg:
	return errors.New(`gobatis: value not lseg[]`)
}

// Value sql/database Value interface
func (pg *PgArrayLSeg) Value() (driver.Value, error) {
	var builder strings.Builder
	builder.Grow(64)
	builder.WriteString(`{`)

	for i, seg := range *pg {
		builder.WriteString(`"[`)

		var pts []*PgPoint
		if seg.P1 != nil {
			pts = append(pts, seg.P1)
		}
		if seg.P2 != nil {
			pts = append(pts, seg.P2)
		}

		for j, pt := range pts {
			builder.WriteString(`(`)
			builder.WriteString(strconv.FormatFloat(pt.X, 'f', -1, 64))
			builder.WriteString(`,`)
			builder.WriteString(strconv.FormatFloat(pt.Y, 'f', -1, 64))
			builder.WriteString(`)`)
			if j != len(pts)-1 {
				builder.WriteString(`,`)
			}
		}

		builder.WriteString(`]"`)

		if i != len(*pg)-1 {
			builder.WriteString(`,`)
		}
	}

	builder.WriteString(`}`)
	return builder.String(), nil
}
