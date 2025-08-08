package model

type Format int

const (
	FormatRss Format = iota
	FormatCeJs
	FormatHtml
)

func (f Format) String() string {
	return [...]string{
		"rss",
		"json",
		"html",
	}[f]
}
