package example

import (
	"context"
	"embed"
	"testing"

	"github.com/fbatis/gobatis"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed statements/*.xml
var embedFs embed.FS

var db *gobatis.DB
var err error
var ctx = context.TODO()

func init() {
	db, err = gobatis.OpenWithEmbedFs(
		`pgx`,
		`postgres://postgres:3333@10.1.209.146:5432/test`,
		embedFs,
		`statements`,
	)
	if err != nil {
		panic(err)
	}
}

/*
	create table example_array (
	  id serial primary key,
	  array_of_integers int[],
	  array_of_text text[],
	  array_of_char char[],
	  array_of_bool boolean[],
	  array_of_float float[],
	  array_of_numeric numeric[],
	  array_of_date date[],
	  array_of_timestamp timestamptz[]
	);
*/
type ExampleArray struct {
	Id               int                   `json:"id"`
	ArrayOfIntegers  gobatis.PgArrayInt    `json:"array_of_integers"`
	ArrayOfText      gobatis.PgArrayString `json:"array_of_text"`
	ArrayOfChar      gobatis.PgArrayString `json:"array_of_char"`
	ArrayOfBool      gobatis.PgArrayBool   `json:"array_of_bool"`
	ArrayOfFloat     gobatis.PgArrayFloat  `json:"array_of_float"`
	ArrayOfNumeric   gobatis.PgArrayString `json:"array_of_numeric"`
	ArrayOfDate      gobatis.PgArrayString `json:"array_of_date"`
	ArrayOfTimestamp gobatis.PgArrayString `json:"array_of_timestamp"`
}

func TestPgArrayNull(t *testing.T) {
	var out []*ExampleArray
	if err = db.WithContext(ctx).
		Mapper(`findExampleArrayById`).
		Args(&gobatis.Args{`id`: 2}).
		Find(&out).Error; err != nil {
		panic(err)
	}
	t.Logf("%#v", out)
}

func TestInsertPgArrayNull(t *testing.T) {
	if err = db.WithContext(ctx).Mapper(`insertExampleArrayWithNull`).
		Args(&gobatis.Args{
			`array_of_integers`: &gobatis.PgArrayInt{111, 222, 333},
			`array_of_text`:     nil,
			`array_of_char`:     nil,
		}).
		Execute().Error; err != nil {
		panic(err)
	}
}
