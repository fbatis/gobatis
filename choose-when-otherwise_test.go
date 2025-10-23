package gobatis

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"
)

func TestChooseWhen(t *testing.T) {
	var out []*Employees
	if err := db.WithContext(ctx).
		Mapper(`findEmployeeByChooseWhenId`).
		Args(&Gobatis{
			//`id`:         400,
			//`department`: 3,
		}).Find(&out).Error; err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}

type Strategy struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func (s *Strategy) Scan(value any) error {
	switch v := value.(type) {
	case []uint8:
		return json.Unmarshal(v, &s)
	default:
		return nil
	}
}

func (s *Strategy) Value() (driver.Value, error) {
	return json.Marshal(s)
}

type MOrder struct {
	Id      int    `json:"id"`
	OrderId string `json:"order_id"`
	Name    string `json:"name"`

	Price     float64   `json:"price"`
	Strategy  *Strategy `json:"strategy"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	//DeletedAt *time.Time `json:"deleted_at"`
}

type Roads struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	StartGeom string `json:"start_geom"`
	EndGeom   string `json:"end_geom"`
}

func TestRoads(t *testing.T) {
	var out []*Roads
	if err := db.WithContext(ctx).
		Mapper(`findRoadById`).
		Args(&Gobatis{
			`id`: 13,
		}).Find(&out).Error; err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}

func TestFindMOrderById(t *testing.T) {
	var out []*MOrder
	if err := db.WithContext(ctx).
		Mapper(`findMOrderById`).
		Args(&Gobatis{
			`ids`: []*MOrder{
				{
					Id: 10006404,
				},
				{
					Id: 10006405,
				},
			},
		}).Find(&out).Error; err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}

func TestComplexSQL(t *testing.T) {
	var out []*MOrder
	if err := db.WithContext(ctx).
		Mapper(`findMOrderComplexMore`).
		Args(&Gobatis{
			`page`: `limit 5`,
			`conditions`: []*Gobatis{
				{
					`alias`: `a`,
					`detail`: []*Gobatis{
						{`id`: 10007527},
						{`price`: `9254.4`, `order_id`: `m0000006303`},
						{`order_id`: `m0000006302`},
					},
				},
				{
					`alias`: `b`,
					`detail`: []Gobatis{
						{`id`: 10007526},
					},
				},
				{
					`alias`: `c`,
					`detail`: []*Gobatis{
						{`id`: 10007524},
					},
				},
			},
		}).Find(&out).Error; err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}

func TestInsertMOrderMulti(t *testing.T) {

	var out []*MOrder
	if err := db.WithContext(ctx).
		Mapper(`insertMOrderMultiAndReturning`).
		Args(&Gobatis{
			`list`: []*MOrder{
				{
					OrderId: `hello02888`,
					Price:   108.0,
					Strategy: &Strategy{
						Name:  `hello`,
						Price: 100.0,
					},
				},
				{
					OrderId: `hello02999`,
					Price:   108.5,
					Strategy: &Strategy{
						Name:  `Golang`,
						Price: 100.5,
					},
				},
			},
		}).Find(&out).Error; err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}

func TestInsertMysqlMOrderMulti(t *testing.T) {

	var out []*MOrder
	if err := db.WithContext(ctx).
		Mapper(`insertMOrderMySQLMulti`).
		Args(&Gobatis{
			`list`: []*MOrder{
				{
					Name:  `h3`,
					Price: 108.0,
					Strategy: &Strategy{
						Name:  `hello`,
						Price: 100.0,
					},
				},
				{
					Name:  `h4`,
					Price: 108.5,
					Strategy: &Strategy{
						Name:  `Golang`,
						Price: 100.5,
					},
				},
			},
		}).Find(&out).Error; err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}
