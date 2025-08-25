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

func definitionQueryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DefinitionRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		// 转换为相对路径
		relFilePath, err := utils.AbsToUnixRel(req.CodebasePath, req.FilePath)
		if err == nil {
			req.FilePath = relFilePath
		}

		l := logic.NewDefinitionQueryLogic(r.Context(), svcCtx)
		resp, err := l.QueryDefinition(&req)
		if err != nil {
			response.Error(w, err)
		} else {
			response.Json(w, resp)
		}
	}
}
