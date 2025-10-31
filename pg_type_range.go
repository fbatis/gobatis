package gobatis

import (
	"database/sql/driver"
	"errors"
	"strings"
)

const (
	// TsTzDateTimeFormat postgres timestamp with time zone format
	TsTzDateTimeFormat = `2006-01-02 15:04:05-07`

	// TsDateTimeFormat postgres timestamp format
	TsDateTimeFormat = `2006-01-02 15:04:05`
)

// PgRange postgres range type
// support int4range, int8range, numrange, tsrange, tstzrange, daterange
type PgRange struct {
	ContainFrom bool
	From        string

	ContainTo bool
	To        string
}

// Scan sql/database Scan interface
func (pg *PgRange) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}

	pg.ContainFrom = false
	scan.Split(SplitPgRangeType)
	for scan.Scan() {
		text := scan.Text()
		if text != `[` && text != `(` {
			continue
		}
		// start
		if text == `[` {
			pg.ContainFrom = true
		}

		// from value
		if !scan.Scan() {
			goto errorRangeType
		}
		text = scan.Text()
		if text == `,` {
			goto parseTo
		}
		pg.From = text

		// separator ,
		if !scan.Scan() {
			goto errorRangeType
		}
		_ = scan.Text()
	parseTo:
		// to value
		if !scan.Scan() {
			goto errorRangeType
		}
		text = scan.Text()
		if text == `)` || text == `]` {
			goto parseEnd
		}
		pg.To = scan.Text()

		// end
		if !scan.Scan() {
			goto errorRangeType
		}
	parseEnd:
		switch scan.Text() {
		case `)`:
			pg.ContainTo = false
			return nil
		case `]`:
			pg.ContainTo = true
			return nil
		default:
		}
	}
errorRangeType:
	return errors.New(`gobatis: value not the form of range`)
}

// Value sql/database Value interface
func (pg *PgRange) Value() (driver.Value, error) {
	var b strings.Builder
	b.Grow(32)

	if pg.ContainFrom {
		b.WriteString(`[`)
	} else {
		b.WriteString(`(`)
	}

	b.WriteString(pg.From)
	b.WriteString(`,`)
	b.WriteString(pg.To)
	if pg.ContainTo {
		b.WriteString(`]`)
	} else {
		b.WriteString(`)`)
	}
	return b.String(), nil
}

// PgArrayRange postgres range[], contains `int4range`, `int8range`, `numrange`, `tsrange`, `tstzrange`, `daterange` type
type PgArrayRange []*PgRange

// Scan sql/database Scan interface
func (pg *PgArrayRange) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgArrayRangeType)

	for scan.Scan() {
		text := scan.Text()
		if text != `{` {
			continue
		}
	newRange:
		for scan.Scan() {
			text = scan.Text()
			if text != `"` {
				continue
			}

			pgRange := &PgRange{}
			for scan.Scan() {
				text := scan.Text()
				if text != `[` && text != `(` {
					continue
				}
				// start
				if text == `[` {
					pgRange.ContainFrom = true
				}

				// from value
				if !scan.Scan() {
					goto errorNotValidPgArrayRange
				}
				text = scan.Text()
				if text == `,` {
					goto parseTo
				}
				pgRange.From = text

				// separator ,
				if !scan.Scan() {
					goto errorNotValidPgArrayRange
				}
				_ = scan.Text()
			parseTo:
				// to value
				if !scan.Scan() {
					goto errorNotValidPgArrayRange
				}
				text = scan.Text()
				if text == `)` || text == `]` {
					goto parseEnd
				}
				pgRange.To = scan.Text()

				// end
				if !scan.Scan() {
					goto errorNotValidPgArrayRange
				}
			parseEnd:
				switch scan.Text() {
				case `)`:
					pgRange.ContainTo = false
				case `]`:
					pgRange.ContainTo = true
				default:
					goto errorNotValidPgArrayRange
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
						*pg = append(*pg, pgRange)
						goto newRange
					case `}`:
						*pg = append(*pg, pgRange)
						return nil
					default:
						goto errorNotValidPgArrayRange
					}
				}
			}
		}
	}
errorNotValidPgArrayRange:
	return errors.New("gobatis: value not range[]")
}

// Value sql/database Value interface
func (pg *PgArrayRange) Value() (driver.Value, error) {
	b := strings.Builder{}
	b.Grow(32)
	b.WriteString(`{`)

	for i, v := range *pg {

		if v.ContainFrom {
			b.WriteString(`"[`)
		} else {
			b.WriteString(`"(`)
		}

		b.WriteString(v.From)
		b.WriteString(`,`)
		b.WriteString(v.To)
		if v.ContainTo {
			b.WriteString(`]"`)
		} else {
			b.WriteString(`)"`)
		}

		if i != len(*pg)-1 {
			b.WriteString(`,`)
		}
	}

	b.WriteString(`}`)
	return b.String(), nil
}
