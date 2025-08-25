package handler

import (
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zgsm-ai/codebase-indexer/internal/logic"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func definitionParseHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileDefinitionParseRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		relFilePath, err := utils.AbsToUnixRel(req.CodebasePath, req.FilePath)
		if err == nil {
			req.FilePath = relFilePath
		}

		l := logic.NewFileDefinitionLogic(r.Context(), svcCtx)
		resp, err := l.ParseFileDefinitions(&req)
		if err != nil {
			response.Error(w, err)
		} else {
			response.Json(w, resp)
		}
	}
}
