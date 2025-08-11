package model

import (
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
	ErrorCount   uint32        `json:"errorCount,omitempty"`
	InternalId   string        `json:"-"`
}
