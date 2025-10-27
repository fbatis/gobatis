package gobatis

import (
	"bufio"
	"bytes"
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
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	default:
		return errors.New(`gobatis: not supported type`)
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
	return errors.New(`gobatis: value not array type`)
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
