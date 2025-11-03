package gobatis

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/expr-lang/expr"

	"github.com/google/uuid"
)

var (
	ErrorElementNotSupported             = errors.New(`element not supported`)
	ErrorXmlNotValid                     = errors.New(`xml not valid`)
	ErrorElifMustFollowIfStmt            = errors.New(`elif must follow if statement`)
	ErrorElseMustFollowIfStmt            = errors.New(`else must follow if statement`)
	ErrorOtherwiseMustFollowChooseStmt   = errors.New(`otherwise must follow when statement`)
	ErrorForeachNeedCollection           = errors.New(`foreach statment need collection attr`)
	ErrorForeachNeedItem                 = errors.New(`foreach statment need item attr`)
	ErrorInputMustBeMap                  = errors.New(`input must be map`)
	ErrorForeachStatementIsNotArrayOrMap = errors.New(`foreach statement is not array or map`)
	ErrorIncludeTagNeedRefIdAttr         = errors.New(`include tag need refid attr`)

	variable   *regexp.Regexp
	multiSpace *regexp.Regexp
)

func init() {
	var err error

	// exp variables
	variable, err = regexp.Compile(`([#$]\{.*?})`)
	if err != nil {
		panic(err)
	}

	// multiSpaceExp variables
	multiSpace, err = regexp.Compile(" +")
	if err != nil {
		panic(err)
	}
}

type HandlerPayload struct {
	Input     any
	SqlMapper map[string]string

	fromChoose bool
	uuidMap    sync.Map
}

// NewUuid generate uuid for variables
func NewUuid() string {
	return `_` + strings.ReplaceAll(uuid.NewString(), `-`, ``)
}

type Handler interface {
	Evaluate(ctx context.Context, input *HandlerPayload) (string, error)
}

type BindVar struct {
	stateSql string
	args     []interface{}
	err      error
}

func (bv *BindVar) Vars() (string, []interface{}, error) {
	return bv.stateSql, bv.args, bv.err
}

type BindHandler interface {
	Bind(ctx context.Context, input *HandlerPayload) *BindVar
}

func placeHolder(typ string, count int) string {
	switch strings.ToLower(typ) {
	case `postgres`, `pg`, `pgx`, `pgx/v5`: // postgres use $1, $2, ... as placeholders for prepare statements.
		return fmt.Sprintf(`$%d`, count+1)
	case `sqlserver`, `mssql`:
		return fmt.Sprintf(`@p%d`, count+1) // for sqlserver, use @p1, @p2, ... as placeholders for prepare statements.
	case `godror`, `goracle`:
		return fmt.Sprintf(`:%d`, count+1) // for oracle, use :1, :2, ... as placeholders for prepare statements.
	default: // mysql or sqlite use ? as placeholders for prepare statements.
		return `?`
	}
}

func bindParamsToVar(ctx context.Context, m Handler, attrMap map[string]string, input *HandlerPayload) *BindVar {
	if reflect.TypeOf(input.Input).Kind() == reflect.Ptr {
		input.Input = reflect.ValueOf(input.Input).Elem().Interface()
	}

	prepareStmt, err := m.Evaluate(ctx, input)
	if err != nil {
		return &BindVar{err: err}
	}

	matches := variable.FindAllString(prepareStmt, -1)
	args := make([]interface{}, 0, len(matches))
	typeValue, _ := attrMap[TypeKey]

	for i, match := range matches {
		matchKey := strings.Trim(match, `$#{}`)
		matchValue, err := expr.Eval(matchKey, input.Input)
		if err != nil {
			return &BindVar{err: err}
		}
		if matchValue != nil && reflect.TypeOf(matchValue).Kind() == reflect.String &&
			strings.Contains(matchValue.(string), `<nil>`) {
			input.uuidMap.Range(func(key, value any) bool {
				if strings.Contains(matchKey, key.(string)) {
					matchKey = strings.ReplaceAll(matchKey, key.(string), value.(string))
				}
				return true
			})
			return &BindVar{err: fmt.Errorf("gobatis: Args not define: %s variable", matchKey)}
		}
		if strings.HasPrefix(match, `#`) {
			mv := reflect.ValueOf(matchValue)
			for mv.Kind() == reflect.Ptr {
				mv = mv.Elem()
			}

			var holders string
			_, driverInterface := matchValue.(interface {
				sql.Scanner
				driver.Valuer
			})

			if (mv.Kind() == reflect.Slice || mv.Kind() == reflect.Array) && !driverInterface {
				var holderArr = make([]string, 0, mv.Len())
				for j := 0; j < mv.Len(); j++ {
					holders := placeHolder(typeValue, i+j)
					holderArr = append(holderArr, holders)
					args = append(args, mv.Index(j).Interface())
				}
				holders = strings.Join(holderArr, `, `)
			} else {
				args = append(args, matchValue)
				holders = placeHolder(typeValue, i)
			}

			prepareStmt = strings.Replace(prepareStmt, match, holders, 1)
		} else if strings.HasPrefix(match, `$`) {
			prepareStmt = strings.ReplaceAll(prepareStmt, match, fmt.Sprintf(`%v`, matchValue))
		}
	}

	// Beautify the sql
	// if you don't need sql beautify, you can comment below two lines
	prepareStmt = strings.ReplaceAll(prepareStmt, "\n", ``)
	prepareStmt = string(multiSpace.ReplaceAll([]byte(prepareStmt), []byte{' '}))

	return &BindVar{
		stateSql: strings.TrimSpace(prepareStmt),
		args:     args,
		err:      nil,
	}
}

// exprEvaluate statement and get true or false
func exprEvaluate(exprString string, input interface{}) (bool, error) {
	prog, err := expr.Compile(exprString)
	if err != nil {
		return false, err
	}
	output, err := expr.Run(prog, input)
	if err != nil {
		return false, err
	}
	switch output {
	case nil:
		return false, err
	case true:
		return true, err
	default:
		return false, err
	}
}

// intervalEvaluate used for caculate the xml chardata if condition ok.
func intervalEvaluate(ctx context.Context, children []interface{}, input *HandlerPayload) (string, error) {
	var builder strings.Builder
	builder.Grow(128)

	var ok bool
	var err error
	var ifExist bool
	var whenExist bool

	for _, child := range children {
	redo:
		switch v := child.(type) {
		case *If:
			ifExist = true
			if testNode, exist := v.AttrsMap[TestKey]; exist && testNode != `` {
				if ok, err = exprEvaluate(testNode, input.Input); err != nil {
					return ``, err
				} else if ok {
					if innerText, err := intervalEvaluate(ctx, v.Children, input); err != nil {
						return ``, err
					} else {
						builder.WriteString(` ` + innerText + ` `)
					}
				}
			}
		case *Elif:
			if !ifExist {
				return ``, ErrorElifMustFollowIfStmt
			}

			if ok {
				continue
			}
			if testNode, exist := v.AttrsMap[TestKey]; exist && testNode != `` {
				if ok, err = exprEvaluate(testNode, input.Input); err != nil {
					return ``, err
				} else if ok {
					if innerText, err := intervalEvaluate(ctx, v.Children, input); err != nil {
						return ``, err
					} else {
						builder.WriteString(` ` + innerText + ` `)
					}
				}
			}
		case *Else:
			if !ifExist {
				return ``, ErrorElseMustFollowIfStmt
			}

			if ok {
				continue
			}
			if innerText, err := intervalEvaluate(ctx, v.Children, input); err != nil {
				return ``, err
			} else {
				builder.WriteString(` ` + innerText + ` `)
			}
		case *Choose:
			input.fromChoose = true
			if innerText, err := intervalEvaluate(ctx, v.Children, input); err != nil {
				return ``, err
			} else {
				builder.WriteString(` ` + innerText + ` `)
			}
			input.fromChoose = false
		case *When:
			if input.fromChoose && whenExist && ok {
				continue
			}
			whenExist = true
			if testNode, exist := v.AttrsMap[TestKey]; exist && testNode != `` {
				if ok, err = exprEvaluate(testNode, input.Input); err != nil {
					return ``, err
				} else if ok {
					if innerText, err := intervalEvaluate(ctx, v.Children, input); err != nil {
						return ``, err
					} else {
						builder.WriteString(` ` + innerText + ` `)
					}
				}
			}
		case *Otherwise:
			if !whenExist {
				return ``, ErrorOtherwiseMustFollowChooseStmt
			}
			if ok {
				continue
			}
			if innerText, err := intervalEvaluate(ctx, v.Children, input); err != nil {
				return ``, err
			} else {
				builder.WriteString(` ` + innerText + ` `)
			}
		case *Foreach:
			var collection string
			var item string
			var separator string
			var arrayIndexKey string

			if collection, ok = v.AttrsMap[CollectionKey]; !ok {
				return ``, ErrorForeachNeedCollection
			}
			if item, ok = v.AttrsMap[ItemKey]; !ok {
				return ``, ErrorForeachNeedItem
			}
			separator, _ = v.AttrsMap[SeparatorKey]
			arrayIndex, _ := v.AttrsMap[IndexKey]
			arrayIndex = strings.TrimSpace(arrayIndex)

			value, err := expr.Eval(collection, input.Input)
			if err != nil {
				return ``, err
			}
			t := reflect.TypeOf(value)
			if t == nil {
				continue
			}

			for t.Kind() == reflect.Ptr {
				t = t.Elem()
			}

			val := reflect.ValueOf(value)

			if reflect.TypeOf(input.Input).Kind() != reflect.Map {
				return ``, ErrorInputMustBeMap
			}
			inputMap := reflect.ValueOf(input.Input)
			for inputMap.Kind() == reflect.Ptr {
				inputMap = inputMap.Elem()
			}

			switch t.Kind() {
			case reflect.Slice, reflect.Array:
				var textArr []string
				for i := 0; i < val.Len(); i++ {
					collectionKey := fmt.Sprintf(`%s[%d]`, collection, i)
					sliceItem, err := expr.Eval(collectionKey, input.Input)
					if err != nil {
						return ``, err
					}
					// Save the original value of the item key
					itemValue := reflect.ValueOf(item)
					previousItemValue := inputMap.MapIndex(itemValue)
					// Save the original value of the index key
					arrayIndexValue := reflect.ValueOf(arrayIndex)
					previousArrayIndexValue := inputMap.MapIndex(arrayIndexValue)

					inputMap.SetMapIndex(itemValue, reflect.ValueOf(sliceItem))
					if arrayIndex != `` {
						arrayIndexKey = NewUuid()
						inputMap.SetMapIndex(reflect.ValueOf(arrayIndexKey), reflect.ValueOf(i))
						inputMap.SetMapIndex(arrayIndexValue, reflect.ValueOf(i))
						input.uuidMap.Store(arrayIndexKey, arrayIndex)
					}
					var newText string
					if newText, err = intervalEvaluate(ctx, v.Children, &HandlerPayload{
						Input:      inputMap.Interface(),
						SqlMapper:  input.SqlMapper,
						fromChoose: input.fromChoose,
					}); err != nil {
						return ``, err
					}

					// restore the item & index value if in original input value
					inputMap.SetMapIndex(itemValue, previousItemValue)
					if arrayIndex != `` {
						// restore array index value from original input value
						inputMap.SetMapIndex(arrayIndexValue, previousArrayIndexValue)
						matches := variable.FindAllString(newText, -1)
						for _, match := range matches {
							newText = strings.NewReplacer(match, strings.NewReplacer(
								arrayIndex,
								arrayIndexKey,
							).Replace(match)).Replace(newText)
						}
					}
					matches := variable.FindAllString(newText, -1)
					for _, match := range matches {
						newText = strings.NewReplacer(
							match,
							strings.NewReplacer(
								item,
								collectionKey,
							).Replace(match),
						).Replace(newText)
						input.uuidMap.Store(collectionKey, item)
					}
					textArr = append(textArr, newText)
				}
				builder.WriteString(` ` + strings.Join(textArr, separator) + ` `)
			default:
				return ``, ErrorForeachStatementIsNotArrayOrMap
			}
		case *Trim:
			if innerText, err := intervalEvaluate(ctx, v.Children, input); err != nil {
				return ``, err
			} else {
				innerText = strings.TrimSpace(innerText)

				if prefixOverrides, ok := v.AttrsMap[PrefixOverridesKey]; ok {
					for _, prefix := range strings.Split(prefixOverrides, `|`) {
						prefix := strings.ToLower(strings.TrimSpace(prefix))
						innerTextLower := strings.ToLower(innerText)
						if strings.HasPrefix(innerTextLower, prefix) {
							innerText = innerText[len(prefix):]
						}
					}
				}
				if prefix, ok := v.AttrsMap[PrefixKey]; ok && prefix != `` {
					builder.WriteString(prefix + ` `)
				}

				builder.WriteString(` ` + innerText + ` `)
			}
		case *Include:
			if v.RefId == `` {
				return ``, ErrorIncludeTagNeedRefIdAttr
			}
			if sqlMapper, ok := input.SqlMapper[v.RefId]; ok {
				sqlMapper = strings.NewReplacer(
					fmt.Sprintf(`${%s}`, v.Alias),
					v.Value,
				).Replace(sqlMapper)
				builder.WriteString(` ` + sqlMapper + ` `)
			} else {
				return ``, fmt.Errorf(`mapper: sql mapper with id: %s not found`, v.RefId)
			}
		case *Where:
			if innerText, err := intervalEvaluate(ctx, v.Children, input); err != nil {
				return ``, err
			} else {
				innerText = strings.TrimSpace(innerText)
				innerTextLower := strings.ToLower(innerText)
				if strings.HasPrefix(innerTextLower, `and`) {
					innerText = innerText[3:]
				}
				if strings.HasPrefix(innerTextLower, `or`) {
					innerText = innerText[2:]
				}
				if strings.TrimSpace(innerText) != `` {
					builder.WriteString(` WHERE ` + innerText + ` `)
				}
			}
		case *Sql:
			input.SqlMapper[v.Id] = v.Text
		case *interface{}:
			if child == nil {
				continue
			}
			child = *v
			goto redo
		case xml.CharData:
			builder.Write(v)
		default:
			return ``, ErrorElementNotSupported
		}
	}

	return builder.String(), nil
}

// Mapper all data mapper into Mapper struct
type Mapper struct {
	Select []*Select
	Update []*Update
	Insert []*Insert
	Delete []*Delete
	Sql    []*Sql

	Attrs   []xml.Attr
	AttrMap map[string]string
}

type XmlName xml.Name

func (xn XmlName) Name() string {
	if xn.Space == `` {
		return xn.Local
	}
	return xn.Space + `:` + xn.Local
}

// parseElementEntry Parse element into struct
func parseElementEntry(d *xml.Decoder, t interface{}) (any, error) {
	var stmt any

	switch tok := t.(type) {
	case *xml.StartElement:
		elementName := XmlName(tok.Name)
		switch strings.ToLower(elementName.Name()) {
		case `if`:
			stmt = NewIf()
		case `elif`:
			stmt = NewElif()
		case `else`:
			stmt = NewElse()
		case `choose`:
			stmt = NewChoose()
		case `when`:
			stmt = NewWhen()
		case `where`:
			stmt = NewWhere()
		case `foreach`:
			stmt = NewForeach()
		case `trim`:
			stmt = NewTrim()
		case `include`:
			stmt = NewInclude()
		case `otherwise`:
			stmt = NewOtherwise()
		case `sql`:
			stmt = NewSql()
		default:
			return nil, ErrorElementNotSupported
		}
		if err := d.DecodeElement(stmt, tok); err != nil {
			return nil, err
		}
		return &stmt, nil
	}

	return nil, ErrorElementNotSupported
}

func (m *Mapper) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if m.AttrMap == nil {
		m.AttrMap = make(map[string]string, 32)
	}
	m.Attrs = start.Attr
	for _, attr := range m.Attrs {
		m.AttrMap[XmlName(attr.Name).Name()] = attr.Value
	}

	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch el := tok.(type) {
		case xml.StartElement:
			elementName := XmlName(el.Name)
			switch strings.ToLower(elementName.Name()) {
			case `select`:
				var selectSt = NewSelect()
				if err := d.DecodeElement(selectSt, &el); err != nil {
					return err
				}
				m.Select = append(m.Select, selectSt)
			case `update`:
				var updateSt = NewUpdate()
				if err := d.DecodeElement(updateSt, &el); err != nil {
					return err
				}
				m.Update = append(m.Update, updateSt)
			case `insert`:
				var insertSt = NewInsert()
				if err := d.DecodeElement(insertSt, &el); err != nil {
					return err
				}
				m.Insert = append(m.Insert, insertSt)
			case `delete`:
				var deleteSt = NewDelete()
				if err := d.DecodeElement(deleteSt, &el); err != nil {
					return err
				}
				m.Delete = append(m.Delete, deleteSt)
			case `sql`:
				var sql Sql
				if err := d.DecodeElement(&sql, &el); err != nil {
					return err
				}
				m.Sql = append(m.Sql, &sql)
			}
		case xml.CharData:
		case xml.EndElement:
			if (XmlName(el.Name)).Name() == XmlName(start.Name).Name() {
				return nil // 错误结束
			}
			return ErrorXmlNotValid
		case xml.Comment, xml.ProcInst, xml.Directive:
		}
	}
}

// ParseMapperFromBuffer parse xml mapper from buffer
func ParseMapperFromBuffer(xmlContent []byte) (*Mapper, error) {
	var mappers Mapper
	err := xml.Unmarshal(xmlContent, &mappers)
	if err != nil {
		return nil, err
	}
	return &mappers, nil
}
