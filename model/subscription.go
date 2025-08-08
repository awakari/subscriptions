package model

import (
	"errors"
	"time"
)

type Subscription struct {
	InterestId   string        `json:"interestId"`
	GroupId      string        `json:"groupId,omitempty"`
	UserId       string        `json:"userId,omitempty"`
	Url          string        `json:"url"`
	Secret       []byte        `json:"secret,omitempty"`
	Format       Format        `json:"format,omitempty"`
	IntervalMin  time.Duration `json:"intervalMin,omitempty"`
	LastResultAt time.Time     `json:"lastResultAt,omitempty"`
	InternalId   string        `json:"-"`
}

const MaxPageLen = 100

// ErrDelivery indicates a failed attempt to send a message to the client.
var ErrDelivery = errors.New("failed to deliver")

var ErrNotFound = errors.New("not found")
