package gobatis

import (
	"database/sql/driver"
	"errors"
	"strconv"
	"strings"
)

// PgPath postgres path type
type PgPath struct {
	Points []*PgPoint
	Open   bool
}

// Scan sql/database Scan interface
func (pg *PgPath) Scan(value any) error {
	scan, err := fetchScanner(value)
	if scan == nil || err != nil {
		return err
	}
	scan.Split(SplitPgRangeType)

	for scan.Scan() {
		text := scan.Text()
		if text == `(` {
			pg.Open = false
			goto newPoint
		}
		if text == `[` {
			pg.Open = true
			goto newPoint
		}
		goto errorNotValidPgPath

	newPoint:
		point := &PgPoint{}
		for scan.Scan() {
			if scan.Text() != `(` {
				continue
			}

			incr := 0
			for scan.Scan() {
				text = scan.Text()
				if text == `,` {
					continue
				}
				if incr == 0 {
					if point.X, err = strconv.ParseFloat(text, 64); err != nil {
						return err
					}
				} else if incr == 1 {
					if point.Y, err = strconv.ParseFloat(text, 64); err != nil {
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
				switch scan.Text() {
				case `,`:
					pg.Points = append(pg.Points, point)
					goto newPoint
				case `)`:
					if pg.Open {
						goto errorNotValidPgPath
					}
					pg.Points = append(pg.Points, point)
					return nil
				case `]`:
					if !pg.Open {
						goto errorNotValidPgPath
					}
					pg.Points = append(pg.Points, point)
					return nil
				}
			}
			goto errorNotValidPgPath
		}
	}
errorNotValidPgPath:
	return errors.New(`gobatis: value not path`)
}

// Value sql/database Value interface
func (pg *PgPath) Value() (driver.Value, error) {
	builder := strings.Builder{}
	builder.Grow(64)
	if pg.Open {
		builder.WriteString(`[`)
	} else {
		builder.WriteString(`(`)
	}

	for i, point := range pg.Points {

		builder.WriteString(`(`)
		builder.WriteString(strconv.FormatFloat(point.X, 'f', -1, 64))
		builder.WriteString(`,`)
		builder.WriteString(strconv.FormatFloat(point.Y, 'f', -1, 64))
		builder.WriteString(`)`)

		if i != len(pg.Points)-1 {
			builder.WriteString(`,`)
		}
	}

	if pg.Open {
		builder.WriteString(`]`)
	} else {
		builder.WriteString(`)`)
	}

	return builder.String(), nil
}

// PgArrayPath postgres path[] type
type PgArrayPath []*PgPath

// Scan sql/database Scan interface
func (pg *PgArrayPath) Scan(value any) error {
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
	newPath:
		for scan.Scan() {
			text = scan.Text()
			if text != `"` {
				continue
			}

			path := &PgPath{}
			for scan.Scan() {
				text := scan.Text()
				if text == `[` {
					path.Open = true
					goto newPoint
				}
				if text == `(` {
					path.Open = false
					goto newPoint
				}
				goto errorNotValidPgArrayPath

			newPoint:
				point := &PgPoint{}
				for scan.Scan() {
					if scan.Text() != `(` {
						continue
					}

					incr := 0
					for scan.Scan() {
						text = scan.Text()
						if text == `,` {
							continue
						}
						if incr == 0 {
							if point.X, err = strconv.ParseFloat(text, 64); err != nil {
								return err
							}
						} else if incr == 1 {
							if point.Y, err = strconv.ParseFloat(text, 64); err != nil {
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
						switch scan.Text() {
						case `,`:
							path.Points = append(path.Points, point)
							goto newPoint
						case `)`:
							if path.Open {
								goto errorNotValidPgArrayPath
							}
							path.Points = append(path.Points, point)
							goto endOrNext
						case `]`:
							if !path.Open {
								goto errorNotValidPgArrayPath
							}
							path.Points = append(path.Points, point)
							goto endOrNext
						}
					}

				endOrNext:
					for scan.Scan() {
						text = scan.Text()
						if text == `"` {
							continue
						}
						if text == `,` {
							*pg = append(*pg, path)
							goto newPath
						}
						if text == `}` {
							*pg = append(*pg, path)
							return nil
						}
					}
					goto errorNotValidPgArrayPath
				}
			}

		}
	}
errorNotValidPgArrayPath:
	return errors.New(`gobatis: value not path[]`)
}

// Value sql/database Value interface
func (pg *PgArrayPath) Value() (driver.Value, error) {
	builder := strings.Builder{}
	builder.Grow(64)
	builder.WriteString(`{`)

	for i, path := range *pg {
		if path.Open {
			builder.WriteString(`"[`)
		} else {
			builder.WriteString(`"(`)
		}

		for i, point := range path.Points {

			builder.WriteString(`(`)
			builder.WriteString(strconv.FormatFloat(point.X, 'f', -1, 64))
			builder.WriteString(`,`)
			builder.WriteString(strconv.FormatFloat(point.Y, 'f', -1, 64))
			builder.WriteString(`)`)

			if i != len(path.Points)-1 {
				builder.WriteString(`,`)
			}
		}

		if path.Open {
			builder.WriteString(`]"`)
		} else {
			builder.WriteString(`)"`)
		}
		if i != len(*pg)-1 {
			builder.WriteString(`,`)
		}
	}

	builder.WriteString(`}`)
	return builder.String(), nil
}
