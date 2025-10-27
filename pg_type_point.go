package gobatis

import (
	"bufio"
	"bytes"
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
	var scan *bufio.Scanner
	switch v := value.(type) {
	case []byte:
		scan = bufio.NewScanner(bytes.NewReader(v))
	case string:
		scan = bufio.NewScanner(strings.NewReader(v))
	default:
		return errors.New(`gobatis: not supported type`)
	}
	scan.Split(SplitPgPointType)
	var err error

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
	return errors.New(`gobatis: value not point type`)
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
