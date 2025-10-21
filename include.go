package gobatis

type Include struct {
	RefId string `xml:"refid,attr"`
	Alias string `xml:"alias,attr"`
	Value string `xml:"value,attr"`
}

func NewInclude() *Include {
	return &Include{}
}
