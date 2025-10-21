package gobatis

type Sql struct {
	Id   string `xml:"id,attr"`
	Text string `xml:",chardata"`
}

func NewSql() *Sql {
	return &Sql{}
}
