package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestStructureQuery(t *testing.T) {

	// Prepare test data
	req := types.StructureRequest{
		ClientId:     "test-client-123",
		CodebasePath: "tmp\\projects\\go\\kubernetes",
		FilePath:     "pkg/auth/authorizer/abac/abac.go",
	}

	// Send request to local service
	reqUrl := fmt.Sprintf("%s/codebase-indexer/api/v1/files/structure?clientId=%s&codebasePath=%s&filePath=%s",
		baseURL,
		req.ClientId,
		url.QueryEscape(req.CodebasePath),
		url.QueryEscape(req.FilePath),
	)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(reqUrl)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	// Parse response
	var result response.Response[types.StructureResponseData]
	err = json.Unmarshal(body, &result)
	assert.NoError(t, err)

	//t.Logf("resp:%+v", string(body))

	// Verify response structure
	assert.Equal(t, 0, result.Code) // 0 indicates success
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		// Verify the structure of the first result
		firstItem := result.Data.List[0]
		assert.NotEmpty(t, firstItem.Name)
		assert.NotEmpty(t, firstItem.ItemType)
		assert.NotEmpty(t, firstItem.Content)
		assert.NotNil(t, firstItem.Position)
	}
}
