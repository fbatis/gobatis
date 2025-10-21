package gobatis

import (
	"context"
	"embed"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed statements/*.xml
var embedFs embed.FS

type Employees struct {
	EmployeeId       int    `json:"employee_id" expr:"employee_id"`
	Name             string `json:"name" expr:"name"`
	Department       int    `json:"department" expr:"department"`
	PerformanceScore string `json:"performance_score" expr:"performance_score"`
	Salary           string `json:"salary" expr:"salary"`
}

type Log struct{}

func (l *Log) Log(ctx context.Context, level int, format string, args ...any) {
	fmt.Printf("sql: %s  %#v\n", format, args)
}

var (
	//dsn        = `root:3333@tcp(10.1.209.146:3306)/test?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms`
	//driverName = `mysql`
	driverName = `pgx`
	dsn        = `postgres://postgres:3333@10.1.209.146:5432/test`

	db  *DB
	ctx = context.TODO()
)

func init() {
	var err error
	db, err = OpenWithEmbedFs(driverName, dsn, embedFs, `statements`, WithLogger(&Log{}))
	if err != nil {
		panic(err)
	}
}

func TestMapper(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`findEmployeeWithForeachId`).
		Bind(BatisInput{`ids`: []int{1, 2, 400}}).
		Find(&out); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}

func TestMapperChoose(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`findEmployeeByNameOrDepartmentWithWhere`).
		Bind(BatisInput{
			`id`:         1,
			`department`: 2,
			`page`:       2,
		}).
		Find(&out); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}

func TestMapperCTE(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`findByCTE`).
		Bind(BatisInput{
			`page`: 1,
		}).
		Find(&out); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}

func TestMapperDelete(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`deleteById`).
		Bind(BatisInput{
			`id`: 1,
		}).
		Execute(); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}

func TestMapperIfElifElse(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`findEmployeeByIfId`).
		Bind(BatisInput{
			`id`: 2,
			//`department`: 2,
		}).
		Find(&out); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}

func TestForeach(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`findEmployeesWithComplex`).
		Bind(BatisInput{
			`list`: []map[string]interface{}{
				{
					`employee_id`: 401,
					`department`:  1,
				},
				{
					`employee_id`: 402,
					`department`:  2,
				},
				{
					`employee_id`: 405,
					`department`:  3,
				},
			},
		}).
		Find(&out); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}

func TestInsert(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`insertMulti`).
		Bind(BatisInput{
			`list`: []*Employees{
				{
					EmployeeId:       500,
					Name:             "500",
					Department:       50,
					PerformanceScore: "5000",
					Salary:           "8000",
				},
				{
					EmployeeId:       501,
					Name:             "500",
					Department:       50,
					PerformanceScore: "5000",
					Salary:           "8000",
				},
			},
		}).
		Execute(); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}

func TestTrim(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`findEmployeeByIfIdTrim`).
		Bind(BatisInput{
			`id`:         500,
			`department`: 3,
		}).
		Find(&out); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}

func TestFindById(t *testing.T) {
	var out []Employees
	if db = db.WithContext(ctx).
		Mapper(`findById`).
		Bind(BatisInput{
			`id`: 500,
		}).
		Find(&out); db.Error != nil {
		panic(db.Error)
	} else {
		fmt.Printf("%#v\n", out)
	}
}
