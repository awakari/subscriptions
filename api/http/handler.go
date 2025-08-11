package http

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/awakari/subscriptions/model"
	"github.com/awakari/subscriptions/service"
	"github.com/awakari/subscriptions/storage"
	"github.com/gin-gonic/gin"
	"io"
	"math/rand"
	"net/http"
	"strconv"
)

type Handler interface {
	Update(ctx *gin.Context)
	Get(ctx *gin.Context)
}

type handler struct {
	svc        service.Service
	clientHttp *http.Client
	userAgent  string
}

const pageLimitDefault = 10
const pageLimitMax = 1000

var errCallback = errors.New("callback failure")

func NewHandler(svc service.Service, clientHttp *http.Client, userAgent string) Handler {
	return handler{
		svc:        svc,
		clientHttp: clientHttp,
		userAgent:  userAgent,
	}
}

// Update godoc
// @Summary Subscribe/unsubscribe to an interest (WebSub)
// @Schemes
// @Description Create or delete a callback for a given interest and receiver URL using the WebSub protocol
// @Tags Subscriptions
// @Accept mpfd
// @Param interestId query string true "Interest ID"
// @Param format query string false "Delivery format, must be one of 'rss' or 'json'. Default is 'rss'."
// @Param payload formData WebSubRequest true "WebSub subscribe/unsubscribe request data"
// @Param interval query int false "Minimum interval between results, e.g. '5m' or '1ms'" default(0) Format(time.Duration)
// @Param X-Awakari-Group-Id header string true "default"
// @Param X-Awakari-User-Id header string true "foo"
// @Param Authorization	header string true "Bearer XXX..."
// @Success 201 {string} string "subscribed"
// @Success 204 {string} string "unsubscribed"
// @Failure 400 {string} string "invalid request"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "unsubscribe: not found"
// @Failure 409 {string} string "subscribe: a callback for the specified interest and URL already exists"
// @Failure 500 {string} string "internal failure"
// @Router /v2 [post]
func (h handler) Update(ctx *gin.Context) {
	groupId := ctx.GetString(model.KeyGroupId)
	userId := ctx.GetString(model.KeyUserId)
	req, err := parseRequest(ctx)
	switch err {
	case nil:
		h.update(ctx, req, groupId, userId)
	default:
		ctx.String(http.StatusBadRequest, err.Error())
	}
}

// Get godoc
// @Summary Get subscription list or details
// @Schemes
// @Description Read an existing subscription by its interest id and BASE64 encoded URL. Responds list when interest id or URL parameter is not specified. Responds list of all own subscriptions when none of both is specified.
// @Tags Subscriptions
// @Produce json
// @Param interestId query string false "Interest ID"
// @Param url query string false "Subscription URL (BASE64 encoded)"
// @Param limit query int false "Results page limit, used only when interest id is not specified."
// @Param X-Awakari-Group-Id header string true "default"
// @Param X-Awakari-User-Id header string true "foo"
// @Param Authorization	header string true "Bearer XXX..."
// @Success 200 {object} Subscription
// @Failure 400 {string} string "failed to decode the specified callback URL"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "not found"
// @Failure 500 {string} string "internal failure"
// @Router /v2 [get]
func (h handler) Get(ctx *gin.Context) {
	interestId := ctx.Query("interestId")
	urlEnc := ctx.Query("url")
	switch urlEnc {
	case "":
		switch interestId {
		case "":
			h.listByUser(ctx)
		default:
			h.count(ctx, interestId)
		}
	default:
		url, err := base64.URLEncoding.DecodeString(urlEnc)
		if err != nil {
			ctx.String(http.StatusBadRequest, "failed to base64 decode callback url %s", url)
			return
		}
		switch interestId {
		case "":
			h.listByUrl(ctx, string(url))
		default:
			h.read(ctx, interestId, string(url))
		}
	}
}

func (h handler) update(ctx *gin.Context, srcReq WebSubRequest, groupId string, userId string) {

	reqChallenge := fmt.Sprintf("%x", rand.Uint64())
	urlCallBack := fmt.Sprintf(
		"%s?hub.mode=%s&hub.topic=%s&hub.challenge=%s&hub.lease_seconds=%d",
		srcReq.CallBack, srcReq.Mode, srcReq.Topic, reqChallenge, leaseSecondsDefault,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlCallBack, nil)
	var resp *http.Response
	if err == nil {
		req.Header.Add("User-Agent", h.userAgent)
		resp, err = h.clientHttp.Do(req)
	}
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			err = fmt.Errorf("%w, url: %s, response status: %d", errCallback, srcReq.CallBack, resp.StatusCode)
		}
	}
	var respChallenge []byte
	if err == nil {
		respChallenge, err = io.ReadAll(resp.Body)
	}
	if err == nil && reqChallenge != string(respChallenge) {
		err = fmt.Errorf(
			"%w: url %s response challenge mismatch: expected %s, got %s",
			errCallback, srcReq.CallBack, reqChallenge, string(respChallenge),
		)
	}

	if err == nil {
		switch srcReq.mode {
		case modeSubscribe:
			err = h.svc.Subscribe(ctx, model.Subscription{
				InterestId:  srcReq.interestId,
				GroupId:     groupId,
				UserId:      userId,
				Url:         srcReq.CallBack,
				Secret:      srcReq.secretBytes,
				Format:      srcReq.format,
				IntervalMin: srcReq.intervalMin,
			})
		case modeUnsubscribe:
			err = h.svc.Unsubscribe(ctx, srcReq.interestId, groupId, userId, srcReq.CallBack)
		}
	}

	switch {
	case err == nil:
		switch srcReq.mode {
		case modeSubscribe:
			ctx.Status(http.StatusAccepted)
		case modeUnsubscribe:
			ctx.Status(http.StatusNoContent)
		}
	case errors.Is(err, errCallback):
		ctx.String(http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound):
		ctx.String(http.StatusNotFound, fmt.Sprintf("a callback for the interest %s not found", srcReq.interestId))
	case errors.Is(err, service.ErrConflict):
		ctx.String(http.StatusConflict, fmt.Sprintf("a callback for the interest %s already exists", srcReq.interestId))
	case errors.Is(err, service.ErrPermitExhausted):
		ctx.String(http.StatusTooManyRequests, err.Error())
	default:
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("callback update failed: %s", err.Error()))
	}

	return
}

func (h handler) listByUser(ctx *gin.Context) {
	limitRaw := ctx.Query("limit")
	var limit int
	switch limitRaw {
	case "":
		limit = pageLimitDefault
	default:
		limit, _ = strconv.Atoi(limitRaw)
		if limit < 1 || limit > pageLimitMax {
			limit = pageLimitDefault
		}
	}
	groupId := ctx.GetString(model.KeyGroupId)
	userId := ctx.GetString(model.KeyUserId)
	page, err := h.svc.ListByUser(ctx, uint32(limit), groupId, userId)
	switch {
	case err == nil:
		if page == nil {
			page = []model.Subscription{}
		}
		ctx.JSON(http.StatusOK, page)
	case errors.Is(err, storage.ErrNotFound):
		ctx.Status(http.StatusNotFound)
	default:
		ctx.String(http.StatusInternalServerError, err.Error())
	}
}

func (h handler) count(ctx *gin.Context, interestId string) {
	count, err := h.svc.CountByInterest(ctx, interestId)
	switch {
	case err == nil:
		ctx.String(http.StatusOK, strconv.Itoa(int(count)))
	case errors.Is(err, storage.ErrNotFound):
		ctx.Status(http.StatusNotFound)
	default:
		ctx.String(http.StatusInternalServerError, err.Error())
	}
}

func (h handler) listByUrl(ctx *gin.Context, url string) {
	cursor := ctx.Query("cursor")
	limitRaw := ctx.Query("limit")
	var limit int
	switch limitRaw {
	case "":
		limit = pageLimitDefault
	default:
		limit, _ = strconv.Atoi(limitRaw)
		if limit < 1 || limit > pageLimitMax {
			limit = pageLimitDefault
		}
	}
	page, err := h.svc.ListByUrl(ctx, uint32(limit), url, cursor)
	switch {
	case err == nil:
		if page == nil {
			page = []string{}
		}
		ctx.JSON(http.StatusOK, InterestPage{
			Page: page,
		})
	case errors.Is(err, storage.ErrNotFound):
		ctx.Status(http.StatusNotFound)
	default:
		ctx.String(http.StatusInternalServerError, err.Error())
	}
}

func (h handler) read(ctx *gin.Context, interestId, url string) {
	groupId := ctx.GetString(model.KeyGroupId)
	userId := ctx.GetString(model.KeyUserId)
	sub, err := h.svc.Subscription(ctx, interestId, groupId, userId, url)
	switch {
	case err == nil:
		respCb := Subscription{
			Url:        sub.Url,
			Format:     sub.Format.String(),
			ErrorCount: sub.ErrorCount,
		}
		ctx.JSON(http.StatusOK, respCb)
	case errors.Is(err, storage.ErrNotFound):
		ctx.Status(http.StatusNotFound)
	default:
		ctx.String(http.StatusInternalServerError, err.Error())
	}
}
