package http

// Subscription model info
// @Description Subscription defines where to push results, format and current delivery failure count
type Subscription struct {

	// Url callback URL that is used to push new results with POST HTTP method, should respond 200/OK
	Url string `json:"url" example:"https://example.com/callback/to/receive/results"`

	// Format the delivery format, maybe one of "rss" or "json"
	Format string `json:"fmt" example:"rss"`

	// ErrorCount represents the current delivery failure count since last success
	ErrorCount uint32 `json:"errorCount" example:"42"`
}

type SubscriptionList struct {
	Count int64 `json:"count"`
}

// InterestPage model info
// @Description list of interest ids
type InterestPage struct {
	Page []string `json:"page"`
}
