package gobatis

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"
)

var (
	ErrorNotFound              = errors.New(`record not found`)
	ErrorInvalidScanRowType    = errors.New(`ScanRow: non-pointer of dest`)
	ErrorInvalidScanSliceType  = errors.New(`ScanSlice: non-pointer of dest`)
	ErrorNowRowsFound          = errors.New(`no rows found`)
	ErrorMapperCallFirst       = errors.New(`gobatis: Mapper() must be invoked, before Bind`)
	ErrorExecuteFailedWithType = errors.New(`gobatis: Execute: invalid type`)

	timeType     = reflect.TypeOf(time.Time{})
	timePtrType  = reflect.TypeOf(&time.Time{})
	nullTimeType = reflect.TypeOf(sql.NullTime{})
)

// BatisInput Args bind variables
type BatisInput = map[string]interface{}

// Gobatis Args bind variables
type Gobatis = map[string]interface{}

// Args bind variables
type Args = map[string]interface{}

const (
	mapperSelect = iota
	mapperInsert
	mapperUpdate
	mapperDelete
)

const (
	LogLevelDebug = iota
	LogLevelInfo
	LogLevelError
)

type Logger interface {
	Log(ctx context.Context, level int, format string, args ...any)
}

type DB struct {
	// sql database handler
	db *sql.DB
	tx *sql.Tx

	// xml mapper
	selectMapper map[string]*Select
	insertMapper map[string]*Insert
	updateMapper map[string]*Update
	deleteMapper map[string]*Delete
	sqlMapper    map[string]string

	// error information.
	LastInserId  int64
	RowsAffected int64
	Error        error

	// mapper
	mapper     BindHandler
	mapperType int

	// bindVars
	bindVars *BindVar

	// rows scanner
	rows *sql.Rows

	// logger
	logger Logger

	// context use
	ctx context.Context

	// open driver name
	driverName string
}

// WithLogger set logger
func WithLogger(logger Logger) func(*DB) {
	return func(db *DB) {
		db.logger = logger
	}
}

// WithMapper set mapper from outspace
func WithMapper(mapper *Mapper) func(*DB) {
	return func(db *DB) {
		if mapper == nil {
			return
		}
		if _, ok := mapper.AttrMap[TypeKey]; !ok && len(db.driverName) != 0 {
			mapper.AttrMap[TypeKey] = db.driverName
		}
		db.mapSelect(mapper)
		db.mapInsert(mapper)
		db.mapUpdate(mapper)
		db.mapDelete(mapper)
		for _, sqlMapper := range mapper.Sql {
			db.sqlMapper[sqlMapper.Id] = sqlMapper.Text
		}
	}
}

// Open database
func Open(driverName, dataSourceName string, opts ...func(*DB)) (*DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	ret := &DB{
		db:           db,
		tx:           nil,
		selectMapper: make(map[string]*Select, 32),
		insertMapper: make(map[string]*Insert, 32),
		updateMapper: make(map[string]*Update, 32),
		deleteMapper: make(map[string]*Delete, 32),
		sqlMapper:    make(map[string]string, 32),
		Error:        nil,
		mapperType:   0,
		rows:         nil,
		ctx:          context.TODO(),
		driverName:   strings.ToLower(driverName),
	}
	for _, opt := range opts {
		opt(ret)
	}
	return ret, nil
}

// OpenWithEmbedFs open database with embed.FS
func OpenWithEmbedFs(
	driverName, dataSourceName string,
	fs embed.FS,
	directory string,
	opts ...func(*DB),
) (*DB, error) {
	if db, err := Open(driverName, dataSourceName, opts...); err != nil {
		return nil, err
	} else {
		directories, err := fs.ReadDir(directory)
		if err != nil {
			return nil, err
		}
		err = db.parseXmlMapper(fs, directory, directories)
		return db, err
	}
}

func (b *DB) Clone() *DB {
	return &DB{
		db:           b.db,
		tx:           b.tx,
		selectMapper: b.selectMapper,
		insertMapper: b.insertMapper,
		updateMapper: b.updateMapper,
		deleteMapper: b.deleteMapper,
		sqlMapper:    b.sqlMapper,
		mapper:       b.mapper,
		mapperType:   b.mapperType,
		Error:        b.Error,
		rows:         b.rows,
		ctx:          b.ctx,
		bindVars:     b.bindVars,
		logger:       b.logger,
	}
}

// parseXmlMapper parse xml mapper from .xml file
// each file end with .xml will be parsed
// and add to mapper with the key of the file xml id
func (b *DB) parseXmlMapper(fs embed.FS, directory string, directories []fs.DirEntry) error {
	for _, file := range directories {
		if file.IsDir() {
			continue
		}

		if !strings.HasSuffix(file.Name(), `.xml`) {
			continue
		}

		filePath := filepath.Join(directory, file.Name())
		filePath = strings.ReplaceAll(filePath, `\`, `/`)
		mapperData, err := fs.ReadFile(filePath)
		if err != nil {
			return err
		}

		mapperData, err = preprocessXMLReplace(mapperData)
		if err != nil {
			return err
		}

		mappers, err := ParseMapperFromBuffer(mapperData)
		if err != nil {
			return err
		}

		if _, ok := mappers.AttrMap[TypeKey]; !ok && len(b.driverName) != 0 {
			mappers.AttrMap[TypeKey] = b.driverName
		}

		b.mapSelect(mappers)
		b.mapInsert(mappers)
		b.mapUpdate(mappers)
		b.mapDelete(mappers)

		for _, sqlMapper := range mappers.Sql {
			b.sqlMapper[sqlMapper.Id] = sqlMapper.Text
		}
	}

	return nil
}

func (b *DB) mapSelect(mappers *Mapper) {
	for i, selectMapper := range mappers.Select {
		if mapperTypeValue, ok := mappers.AttrMap[TypeKey]; ok {
			if tv, ok := selectMapper.AttrsMap[TypeKey]; !ok || strings.TrimSpace(tv) == `` {
				selectMapper.AttrsMap[TypeKey] = mapperTypeValue
			}
		}
		if value, ok := selectMapper.AttrsMap[IdKey]; ok {
			if _, ok := b.selectMapper[value]; ok {
				panic(fmt.Errorf("gobatis: select mapper with id: %s redeclared", value))
			}
			b.selectMapper[value] = mappers.Select[i]
		}
	}
}

func (b *DB) mapInsert(mappers *Mapper) {
	for i, insertMapper := range mappers.Insert {
		if mapperTypeValue, ok := mappers.AttrMap[TypeKey]; ok {
			if tv, ok := insertMapper.AttrsMap[TypeKey]; !ok || strings.TrimSpace(tv) == `` {
				insertMapper.AttrsMap[TypeKey] = mapperTypeValue
			}
		}
		if value, ok := insertMapper.AttrsMap[IdKey]; ok {
			if _, ok := b.insertMapper[value]; ok {
				panic(fmt.Errorf("gobatis: insert mapper with id: %s redeclared", value))
			}
			b.insertMapper[value] = mappers.Insert[i]
		}
	}
}

func (b *DB) mapUpdate(mappers *Mapper) {
	for i, updateMapper := range mappers.Update {
		if mapperTypeValue, ok := mappers.AttrMap[TypeKey]; ok {
			if tv, ok := updateMapper.AttrsMap[TypeKey]; !ok || strings.TrimSpace(tv) == `` {
				updateMapper.AttrsMap[TypeKey] = mapperTypeValue
			}
		}
		if value, ok := updateMapper.AttrsMap[IdKey]; ok {
			if _, ok := b.updateMapper[value]; ok {
				panic(fmt.Errorf("gobatis: update mapper with id: %s redeclared", value))
			}
			b.updateMapper[value] = mappers.Update[i]
		}
	}
}

func (b *DB) mapDelete(mappers *Mapper) {
	for i, deleteMapper := range mappers.Delete {
		if mapperTypeValue, ok := mappers.AttrMap[TypeKey]; ok {
			if tv, ok := deleteMapper.AttrsMap[TypeKey]; !ok || strings.TrimSpace(tv) == `` {
				deleteMapper.AttrsMap[TypeKey] = mapperTypeValue
			}
		}
		if value, ok := deleteMapper.AttrsMap[IdKey]; ok {
			if _, ok := b.deleteMapper[value]; ok {
				panic(fmt.Errorf("gobatis: delete mapper with id: %s redeclared", value))
			}
			b.deleteMapper[value] = mappers.Delete[i]
		}
	}
}

// WithContext set context to db
func (b *DB) WithContext(ctx context.Context) *DB {
	db := b.Clone()
	db.ctx = ctx
	db.Error = nil
	return db
}

// Transaction start transaction to database
// when transaction completed successfully, the tx will be committed and can't use again
func (b *DB) Transaction(fn func(tx *DB) error) (err error) {
	var tx *sql.Tx

	if b.Error != nil {
		return b.Error
	}

	db := b.Clone()

	tx, err = db.db.BeginTx(db.ctx, &sql.TxOptions{
		Isolation: 0,
		ReadOnly:  false,
	})
	if err != nil {
		return err
	}

	defer func() {
		if rerr := recover(); rerr != nil {
			switch x := rerr.(type) {
			case error:
				err = x
			case string:
				err = errors.New(x)
			default:
				err = fmt.Errorf("transaction panic: %v", rerr)
			}
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf(`rollback error: %#v, raw error: %#v`, rollbackErr, err)
			}
		}

		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf(`rollback error: %#v, raw error: %#v`, rollbackErr, err)
			}
		} else if err = tx.Commit(); err == nil {
			db.tx = nil
		}
	}()

	db.tx = tx

	return fn(db)
}

// RawQuery database
// then call Find to get result
func (b *DB) RawQuery(query string, args ...any) *DB {
	db := b.Clone()

	if db.logger != nil {
		db.logger.Log(db.ctx, LogLevelDebug, query, args...)
	}

	if db.tx != nil {
		db.rows, db.Error = db.tx.QueryContext(db.ctx, query, args...)
	} else {
		db.rows, db.Error = db.db.QueryContext(db.ctx, query, args...)
	}

	return db
}

// RawExec do an insert, update or delete operation
func (b *DB) RawExec(query string, args ...any) *DB {
	var err error
	var result sql.Result
	if b.Error != nil {
		return b
	}

	db := b.Clone()
	if db.logger != nil {
		db.logger.Log(db.ctx, LogLevelDebug, query, args...)
	}

	if db.tx != nil {
		result, err = db.tx.ExecContext(db.ctx, query, args...)
	} else {
		result, err = db.db.ExecContext(db.ctx, query, args...)
	}
	if err != nil {
		db.Error = err
		return db
	}

	db.RowsAffected, db.Error = result.RowsAffected()
	if db.Error != nil {
		return db
	}

	db.LastInserId, _ = result.LastInsertId()
	return db
}

// Find  result from previous Query call
func (b *DB) Find(dest any) *DB {
	if b.Error != nil {
		return b
	}

	var t = b
	switch b.mapperType {
	// to support postgres-like sql: insert/update/delete xxx returning xxx
	case mapperInsert, mapperUpdate, mapperDelete:
		statements, args, err := b.bindVars.Vars()
		if err != nil {
			b.Error = err
			return b
		}
		t = b.RawQuery(statements, args...)
	default:
		// omit
	}

	db := t.Clone()

	if db.rows == nil {
		db.Error = ErrorNowRowsFound
		return db
	}
	defer func() {
		db.rows.Close()
		db.rows = nil
	}()

	db.Error = db.scan(db.rows, dest)

	return db
}

// Mapper fetch mapper from all xml with the id identifier
//
// forexample:
//
// fetch data: db.Mapper('id').Args(variables).Find(dest).Error
//
// otherwise: db.Mapper('id').Args(variables).Execute().Error
func (b *DB) Mapper(mapperId string) *DB {
	if b.Error != nil {
		return b
	}

	db := b.Clone()

	if mapper, ok := db.selectMapper[mapperId]; ok {
		db.mapper = mapper
		db.mapperType = mapperSelect
		return db
	}

	if mapper, ok := db.insertMapper[mapperId]; ok {
		db.mapper = mapper
		db.mapperType = mapperInsert
		return db
	}

	if mapper, ok := db.updateMapper[mapperId]; ok {
		db.mapper = mapper
		db.mapperType = mapperUpdate
		return db
	}

	if mapper, ok := db.deleteMapper[mapperId]; ok {
		db.mapper = mapper
		db.mapperType = mapperDelete
		return db
	}

	db.Error = fmt.Errorf("gobatis: mapper with id: %s not found", mapperId)
	return db
}

// Args alias to Bind operation
func (b *DB) Args(variables interface{}) *DB {
	return b.Bind(variables)
}

// Bind variables to mapper
// generate stmt prepared handler
// next call will use the stmt.
// caller should have known if the variables input was map, he must make sure the input variables
// [ thread-safe ].
func (b *DB) Bind(variables interface{}) *DB {
	if b.Error != nil {
		return b
	}

	db := b.Clone()
	if db.mapper == nil {
		db.Error = ErrorMapperCallFirst
		return db
	}

	t := reflect.TypeOf(variables)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// make sure the input value's type is map
	// if tag contains foreach, the input' type must be `map`
	// caller should have known if the variables input was map, he must make sure the input variables
	// thread-safe.
	if t.Kind() == reflect.Struct {
		varBuf, err := json.Marshal(variables)
		if err != nil {
			db.Error = err
			return db
		}
		var variablesMap map[string]interface{}
		err = json.Unmarshal(varBuf, &variablesMap)
		if err != nil {
			db.Error = err
			return db
		}
		variables = variablesMap
	}

	db.bindVars = db.mapper.Bind(db.ctx, &HandlerPayload{
		Input: variables, SqlMapper: db.sqlMapper, fromChoose: false, uuidMap: sync.Map{},
	})
	statements, args, err := db.bindVars.Vars()
	if err != nil {
		db.Error = err
		return db
	}

	switch db.mapperType {
	case mapperSelect:
		return db.RawQuery(statements, args...)
	default:
		return db
	}
}

// Execute execute database's insert, update and delete
// if database running statements with returning, use method Find instead.
func (b *DB) Execute() *DB {
	if b.Error != nil {
		return b
	}

	db := b.Clone()
	if db.mapper == nil {
		db.Error = ErrorMapperCallFirst
		return db
	}

	statements, args, err := db.bindVars.Vars()
	if err != nil {
		db.Error = err
		return db
	}

	switch db.mapperType {
	case mapperInsert, mapperUpdate, mapperDelete:
		return db.RawExec(statements, args...)
	default:
		db.Error = ErrorExecuteFailedWithType
		return db
	}
}

func (b *DB) columnName(f reflect.StructField) string {
	tags := []string{`expr`, `leopard`, `db`, `gorm`, `sql`, `json`}
	for _, tag := range tags {
		if n, ok := f.Tag.Lookup(tag); ok {
			for _, piece := range strings.Split(n, `;`) {
				if !strings.HasPrefix(piece, `column`) {
					continue
				}
				return piece[strings.Index(piece, `:`)+1:]
			}

			if strings.Contains(n, `,`) {
				return n[0:strings.Index(n, `,`)]
			}

			return n
		}
	}
	return strings.ToLower(f.Name)
}

func (b *DB) parseEmbed(v map[string][]int, typ reflect.Type, idxs []int, index int) map[string][]int {
	idx := make([]int, index+1, index+2)
	copy(idx, idxs)

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		if f.PkgPath != `` {
			continue
		}

		idx[index] = i

		if typ := f.Type; f.Anonymous {
			switch {
			case typ.Kind() == reflect.Struct:
				v = b.parseEmbed(v, typ, idx, index+1)
			}
			continue
		}

		newIdx := append([]int{}, idx...)
		v[f.Name] = newIdx
		v[b.columnName(f)] = newIdx
	}
	return v
}

type rowScan struct {
	types []reflect.Type
	ctype []*sql.ColumnType
	value func(v ...any) (reflect.Value, error)
}

func (rs *rowScan) values() []any {
	vs := make([]any, 0, len(rs.types))
	for _, typ := range rs.types {
		vs = append(vs, reflect.New(typ).Interface())
	}
	return vs
}

func (b *DB) scanStruct(typ reflect.Type, columns []string, ctypes []*sql.ColumnType) (*rowScan, error) {
	names := make(map[string][]int, typ.NumField())
	rs := &rowScan{types: make([]reflect.Type, 0, typ.NumField())}

	names = b.parseEmbed(names, typ, []int{}, 0)
	for i, column := range columns {
		var idx []int
		switch name := strings.Split(column, "(")[0]; {
		case names[name] != nil:
			idx = names[name]
		case names[strings.ToLower(name)] != nil:
			idx = names[strings.ToLower(name)]
		default:
			switch ctypes[i].ScanType() {
			case timeType, timePtrType:
				rs.types = append(rs.types, nullTimeType)
			default:
				rs.types = append(rs.types, ctypes[i].ScanType())
			}
			continue
		}
		rtype := typ.Field(idx[0]).Type
		for _, vi := range idx[1:] {
			rtype = rtype.Field(vi).Type
		}

		rs.types = append(rs.types, rtype)
	}

	rs.value = func(vs ...any) (reflect.Value, error) {
		dest := reflect.New(typ).Elem()
		for i, v := range vs {

			name := columns[i]
			if reflect.ValueOf(v).IsNil() {
				continue
			}
			name = name[strings.Index(name, `.`)+1:]

			rv := reflect.Indirect(reflect.ValueOf(v))

			idx, ok := names[name]
			if !ok {
				continue
			}

			dv := dest.Field(idx[0])
			for _, vi := range idx[1:] {
				dv = dv.Field(vi)
			}

			dv.Set(rv)
		}

		return dest, nil
	}

	return rs, nil
}

func (b *DB) scanPointer(typ reflect.Type, columns []string, ctypes []*sql.ColumnType) (*rowScan, error) {
	typ = typ.Elem()
	rs, err := b.scanType(typ, columns, ctypes)
	if err != nil {
		return nil, err
	}
	w := rs.value
	rs.value = func(vs ...any) (reflect.Value, error) {
		v, err := w(vs...)
		if err != nil {
			return reflect.Value{}, err
		}
		rv := reflect.Indirect(v)
		pv := reflect.New(rv.Type())
		pv.Elem().Set(rv)
		return pv, nil
	}
	return rs, nil
}

func (b *DB) scanMap(typ reflect.Type, columns []string, ctypes []*sql.ColumnType) (*rowScan, error) {
	rs := &rowScan{types: make([]reflect.Type, 0, len(ctypes))}

	for _, ty := range ctypes {
		switch ty.ScanType() {
		case timeType, timePtrType:
			rs.types = append(rs.types, nullTimeType)
		default:
			rs.types = append(rs.types, ty.ScanType())
		}
	}

	rs.value = func(vs ...any) (reflect.Value, error) {
		mv := reflect.MakeMap(typ)

		for i, v := range vs {
			rv := reflect.Indirect(reflect.ValueOf(v))
			switch {
			case typ.Elem().Kind() == reflect.Interface || rv.Kind() == typ.Elem().Kind():
			default:
				continue
			}
			mv.SetMapIndex(reflect.ValueOf(columns[i]), rv)
		}

		return mv, nil
	}

	return rs, nil
}

func (b *DB) scanType(typ reflect.Type, columns []string, ctypes []*sql.ColumnType) (*rowScan, error) {
	switch k := typ.Kind(); {
	case k == reflect.Map:
		return b.scanMap(typ, columns, ctypes)
	case k == reflect.Interface:
		return b.scanMap(reflect.TypeOf((*map[string]any)(nil)).Elem(), columns, ctypes)
	case k == reflect.Struct:
		return b.scanStruct(typ, columns, ctypes)
	case k == reflect.Pointer:
		return b.scanPointer(typ, columns, ctypes)
	default:
		return nil, fmt.Errorf(`scanType: unsupported type ([]%s)`, k)
	}
}

func (b *DB) scanSlice(rows *sql.Rows, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer {
		return ErrorInvalidScanSliceType
	}

	v = reflect.Indirect(v)
	if k := v.Kind(); k != reflect.Slice {
		return fmt.Errorf("ScanSlice: invalid type: %s. expect slice as artument", v)
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	types, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	scan, err := b.scanType(v.Type().Elem(), columns, types)
	if err != nil {
		return err
	}

	for rows.Next() {
		vs := scan.values()

		if err = rows.Scan(vs...); err != nil {
			return err
		}

		rv, err := scan.value(vs...)
		if err != nil {
			return err
		}

		v.Set(reflect.Append(v, rv))
	}

	return rows.Err()
}

func (b *DB) scan(rows *sql.Rows, dest any) error {
	t := reflect.TypeOf(dest)
	if t.Kind() != reflect.Pointer {
		return ErrorInvalidScanRowType
	}

	t = t.Elem()
	if t.Kind() == reflect.Slice {
		return b.scanSlice(rows, dest)
	}

	v := reflect.MakeSlice(reflect.SliceOf(t), 0, 5)
	vt := reflect.NewAt(v.Type(), v.UnsafePointer())

	err := b.scanSlice(rows, vt.Interface())
	if err != nil {
		return err
	}

	if vt.Elem().Len() == 0 {
		return ErrorNotFound
	}

	reflect.ValueOf(dest).Elem().Set(vt.Elem().Index(0))
	return nil
}
