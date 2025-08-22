package logic

import (
	"context"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/errs"
	"github.com/zgsm-ai/codebase-indexer/internal/store/vector"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"gorm.io/gorm"

	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	minPositive = 1
	defaultTopK = 5
	paramQuery  = "query"
)

type SemanticLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSemanticSearchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SemanticLogic {
	return &SemanticLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SemanticLogic) SemanticSearch(req *types.SemanticSearchGetRequest) (resp *types.SemanticSearchResponseData, err error) {
	topK := req.TopK
	if topK < minPositive {
		topK = defaultTopK
	}
	if utils.IsBlank(req.Query) {
		return nil, errs.NewInvalidParamErr(paramQuery, req.Query)
	}
	clientId := req.ClientId
	clientPath := req.CodebasePath

	codebase, err := l.svcCtx.Querier.Codebase.FindByClientIdAndPath(l.ctx, clientId, clientPath)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NewRecordNotFoundErr(types.NameCodeBase, fmt.Sprintf("client_id: %s, clientPath: %s", clientId, clientPath))
	}
	ctx := context.WithValue(l.ctx, tracer.Key, tracer.RequestTraceId(int(codebase.ID)))

	documents, err := l.svcCtx.VectorStore.Query(ctx, req.Query, topK,
		vector.Options{CodebaseId: codebase.ID,
			CodebasePath: codebase.Path, CodebaseName: codebase.Name})
	if err != nil {
		return nil, err
	}
	return &types.SemanticSearchResponseData{
		List: documents,
	}, nil
}
