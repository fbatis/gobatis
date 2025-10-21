package gobatis

import "testing"

func TestIfStatementTag(t *testing.T) {
	var out []*Employees
	if err := db.WithContext(ctx).
		Mapper(`findEmployeeById`).
		Args(&Gobatis{`id`: 500}).Find(&out).Error; err != nil {
		panic(err)
	}
	t.Log(out)
}

func TestIfElseStatementTag(t *testing.T) {
	var out []*Employees
	if err := db.WithContext(ctx).
		Mapper(`findEmployeeByIfElseId`).
		Args(&Gobatis{`id`: 501}).Find(&out).Error; err != nil {
		panic(err)
	}
	t.Log(out)
}

func TestIfElifElseStatementTag(t *testing.T) {
	var out []*Employees
	if err := db.WithContext(ctx).
		Mapper(`findEmployeeByIfElifElseId`).
		Args(&Gobatis{`department`: 2}).Find(&out).Error; err != nil {
		panic(err)
	}
	t.Log(out)
}
