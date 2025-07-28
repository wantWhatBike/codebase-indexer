package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestDeleteCodebase(t *testing.T) {
	var result response.Response[types.DeleteCodebaseResponseData]
	err := doRequest(http.MethodDelete, "/codebase-indexer/api/v1/codebase", map[string]string{
		"clientId":     "test-client-123",
		"codebasePath": "\\tmp\\test\\test-project",
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
}

func TestDeleteIndex(t *testing.T) {
	var result response.Response[types.DeleteIndexResponseData]
	err := doRequest(http.MethodDelete, "/codebase-indexer/api/v1/index", map[string]string{
		"clientId":     "test-client-123",
		"codebasePath": "\\tmp\\projects\\go\\kubernetes",
		"taskType":     "codegraph",
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
}

func TestIndexSummary(t *testing.T) {
	var result response.Response[types.IndexSummaryResonseData]
	err := doRequest(http.MethodGet, "/codebase-indexer/api/v1/index/summary", map[string]string{
		"clientId":     "test-client-123",
		"codebasePath": "\\tmp\\projects\\go\\kubernetes",
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
	assert.NotNil(t, result.Data.Embedding)
	assert.NotNil(t, result.Data.CodeGraph)
}

func TestIndexCodegraphTask(t *testing.T) {
	var result response.Response[types.IndexTaskResponseData]
	reqBody := types.IndexTaskRequest{
		ClientId:     "test-client-123",
		CodebasePath: "\\tmp\\projects\\go\\kubernetes",
		IndexType:    "codegraph",
	}

	err := doRequest(http.MethodPost, "/codebase-indexer/api/v1/index/task", nil, reqBody, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
}

func TestIndexEmbeddingTask(t *testing.T) {
	var result response.Response[types.IndexTaskResponseData]
	reqBody := types.IndexTaskRequest{
		ClientId:     "test-client-123",
		CodebasePath: "\\tmp\\projects\\go\\kubernetes",
		IndexType:    "embedding",
	}

	err := doRequest(http.MethodPost, "/codebase-indexer/api/v1/index/task", nil, reqBody, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
}

func TestIndexAllTask(t *testing.T) {
	var result response.Response[types.IndexTaskResponseData]
	reqBody := types.IndexTaskRequest{
		ClientId:     "test-client-123",
		CodebasePath: "\\tmp\\projects\\go\\kubernetes",
		IndexType:    "all",
	}

	err := doRequest(http.MethodPost, "/codebase-indexer/api/v1/index/task", nil, reqBody, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
}
