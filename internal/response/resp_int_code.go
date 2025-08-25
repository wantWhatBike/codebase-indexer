package response

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
)

const (
	CodeIntOK = 0

	CodeIntError = -1
)

type ResponseWithIntCode[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
}

func OkWithIntCode(w http.ResponseWriter) {
	httpx.OkJson(w, wrapResponseWithIntCode(nil))
}

func ErrorWithIntCode(w http.ResponseWriter, e error) {
	logx.WithCallerSkip(2).Errorf("response error: %v", e)
	httpx.WriteJson(w, http.StatusBadRequest, wrapResponseWithIntCode(e)) // TODO 500会触发go-zero breaker
}

func BytesWithIntCode(w http.ResponseWriter, v []byte) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(v)
	w.Header().Set("Content-Type", "application/octet-stream")
}

func JsonCtxWithIntCode(ctx context.Context, w http.ResponseWriter, v any) {
	httpx.OkJsonCtx(ctx, w, wrapResponseWithIntCode(v))
}

func JsonWithIntCode(w http.ResponseWriter, v any) {
	httpx.OkJson(w, wrapResponseWithIntCode(v))
}

func wrapResponseWithIntCode(v any) ResponseWithIntCode[any] {
	var resp ResponseWithIntCode[any]
	switch data := v.(type) {
	case error:
		resp.Code = CodeIntError
		resp.Message = data.Error()
	default:
		resp.Code = CodeIntOK
		resp.Message = MessageOk
		resp.Success = true
		resp.Data = v
	}

	return resp
}
