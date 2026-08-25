package errcode_test

import (
	"errors"
	"testing"

	"github.com/hopeio/mix"
	"github.com/hopeio/scaffold/errcode"
)

func TestMap_Untyped(t *testing.T) {
	got := errcode.Map(errors.New("pq: boom"), errcode.DBError)
	if got != errcode.DBError {
		t.Fatalf("got=%v", got)
	}
}

func TestMap_TypedPassThrough(t *testing.T) {
	biz := errcode.InvalidArgument.Msg("auth.err.x", nil)
	if errcode.Map(biz, errcode.DBError) != biz {
		t.Fatal("typed ErrResp must pass through")
	}
	if errcode.Map(errcode.TimesTooMuch, errcode.RedisErr) != errcode.TimesTooMuch {
		t.Fatal("typed ErrCode must pass through")
	}
	if errcode.Map(nil, errcode.DBError) != nil {
		t.Fatal("nil")
	}
	_ = mix.Success
}
