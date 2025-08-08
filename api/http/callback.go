package http

// Callback model info
// @Description callback where results are being pushed to
type Callback struct {

	// Url callback URL that is used to push new results with POST HTTP method, should respond 200/OK
	Url string `json:"url" example:"https://example.com/callback/to/receive/results"`

	// Format the delivery format, maybe one of "rss" or "json"
	Format string `json:"fmt" example:"rss"`
}

type CallbackList struct {
	Count int64 `json:"count"`
}

// InterestPage model info
// @Description list of interest ids
type InterestPage struct {
	Page []string `json:"page"`
}
