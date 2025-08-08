package subject

import (
	"errors"
	"fmt"
	"github.com/awakari/subscriptions/model/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Decode(src Subject) (dst usage.Subject, err error) {
	switch src {
	case Subject_Interests:
		dst = usage.SubjectInterests
	case Subject_PublishHourly:
		dst = usage.SubjectPublishHourly
	case Subject_PublishDaily:
		dst = usage.SubjectPublishDaily
	case Subject_InterestsPublic:
		dst = usage.SubjectInterestsPublic
	case Subject_Subscriptions:
		dst = usage.SubjectSubscriptions
	default:
		err = status.Error(codes.InvalidArgument, fmt.Sprintf("invalid subject: %s", src))
	}
	return
}

func Encode(src usage.Subject) (dst Subject, err error) {
	switch src {
	case usage.SubjectInterests:
		dst = Subject_Interests
	case usage.SubjectPublishHourly:
		dst = Subject_PublishHourly
	case usage.SubjectPublishDaily:
		dst = Subject_PublishDaily
	case usage.SubjectInterestsPublic:
		dst = Subject_InterestsPublic
	case usage.SubjectSubscriptions:
		dst = Subject_Subscriptions
	default:
		err = errors.New(fmt.Sprintf("invalid subject: %s", src))
	}
	return
}
