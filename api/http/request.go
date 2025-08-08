package http

import (
	"fmt"
	"github.com/awakari/subscriptions/model"
	"github.com/gin-gonic/gin"
	"math"
	"net/url"
	"strings"
	"time"
)

// WebSubRequest model info
// @Description WebSub subscribe/unsubscribe request data
type WebSubRequest struct {

	// Callback URL to deliver results. To receive this URL should handle POST requests. On subscribe, it is tested with GET request and should return a 200/OK response with the same challenge as in the request.
	CallBack string `form:"hub.callback" binding:"required" example:"https://example.com/callback/to/receive/results"`

	// Mode must be one of "subscribe" or "unsubscribe", depending on the request purpose
	Mode string `form:"hub.mode" binding:"required" example:"subscribe"`

	// Topic is an interest feed URL
	Topic string `form:"hub.topic" binding:"required" example:"https://reader.awakari.com/v1/sub/rss/myInterest1"`

	// Secret is used to sign the pushed results when specified.
	Secret string `form:"hub.secret" example:"mySecret1"`

	// parsed and validated fields
	mode        mode
	secretBytes []byte
	interestId  string
	format      model.Format
	intervalMin time.Duration
}

type mode int

const (
	modeSubscribe mode = iota
	modeUnsubscribe
)

const leaseSecondsDefault = math.MaxInt32 // as long as possible, ~ 68 years
const secretMaxLen = 200

func parseRequest(ctx *gin.Context) (req WebSubRequest, err error) {
	format := ctx.Query("format")
	switch format {
	case model.FormatCeJs.String():
		req.format = model.FormatCeJs
	default:
		req.format = model.FormatRss
	}
	req.interestId = ctx.Query("interestId")
	err = ctx.Bind(&req)
	if err == nil {
		req.secretBytes = []byte(req.Secret)
		if len(req.secretBytes) > secretMaxLen {
			err = fmt.Errorf("secret is too long: %d bytes, limit is %d", len([]byte(req.Secret)), secretMaxLen)
		}
	}
	if err == nil {
		_, err = url.Parse(req.CallBack)
	}
	if err == nil {
		switch req.Mode {
		case "subscribe":
			req.mode = modeSubscribe
		case "unsubscribe":
			req.mode = modeUnsubscribe
		default:
			err = fmt.Errorf("hub.mode is invalid: %s", req.Mode)
		}
	}
	if err == nil && !strings.HasSuffix(req.Topic, req.interestId) {
		err = fmt.Errorf("hub.topic is invalid: %s", req.Topic)
	}
	if err == nil {
		intervalRaw := ctx.Query("interval")
		if intervalRaw != "" {
			req.intervalMin, err = time.ParseDuration(intervalRaw)
			if err != nil {
				err = fmt.Errorf("interval query parameter value is invalid: %s", intervalRaw)
			}
			if req.intervalMin < 0 {
				err = fmt.Errorf("interval query parameter value should not be negative: %s", intervalRaw)
			}
		}
	}
	return
}
