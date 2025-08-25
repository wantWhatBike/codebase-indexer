package handler

import (
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zgsm-ai/codebase-indexer/internal/logic"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func getCodebaseHashHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CodebaseHashRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.ErrorWithIntCode(w, err)
			return
		}

		l := logic.NewCompareCodebaseLogic(r.Context(), svcCtx)
		resp, err := l.GetCodebaseHash(&req)
		if err != nil {
			response.ErrorWithIntCode(w, err)
		} else {
			response.JsonWithIntCode(w, resp)
		}
	}
}
