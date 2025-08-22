package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zgsm-ai/codebase-indexer/internal/response"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func TestQueryDefinition(t *testing.T) {
	var result response.Response[types.DefinitionResponseData]
	err := doRequest(http.MethodGet, "/codebase-indexer/api/v1/search/definition", map[string]string{
		"clientId":     clientId,
		"codebasePath": "G:\\tmp\\projects\\go\\kubernetes",
		"filePath":     "pkg/auth/authorizer/abac/abac.go",
		"startLine":    "59",
		"endLine":      "119",
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, 0, result.Code)
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		firstNode := result.Data.List[0]
		assert.NotEmpty(t, firstNode.FilePath)
		assert.NotNil(t, firstNode.Position)
		assert.NotNil(t, firstNode.Content)
		assert.NotEmpty(t, firstNode.Content)
	}
}

func TestQueryDefinitionByCodeSnippet(t *testing.T) {
	var result response.Response[types.DefinitionResponseData]
	err := doRequest(http.MethodGet, "/codebase-indexer/api/v1/search/definition", map[string]string{
		"clientId":     clientId,
		"codebasePath": "G:\\tmp\\projects\\go\\kubernetes",
		"filePath":     "pkg/auth/authorizer/abac/abac.go",
		"startLine":    "1",
		"endLine":      "2",
		"codeSnippet": `
package abac

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/kubernetes/pkg/apis/abac"

	// Import latest API for init/side-effects
	_ "k8s.io/kubernetes/pkg/apis/abac/latest"
	"k8s.io/kubernetes/pkg/apis/abac/v0"
)

type policyLoadError struct {
	path string
	line int
	data []byte
	err  error
}

func (p policyLoadError) Error() string {
	if p.line >= 0 {
		return fmt.Sprintf("error reading policy file %s, line %d: %s: %v", p.path, p.line, string(p.data), p.err)
	}
	return fmt.Sprintf("error reading policy file %s: %v", p.path, p.err)
}

// PolicyList is simply a slice of Policy structs.
type PolicyList []*abac.Policy

// NewFromFile attempts to create a policy list from the given file.
//
// TODO: Have policies be created via an API call and stored in REST storage.
func NewFromFile(path string) (PolicyList, error) {
	// File format is one map per line.  This allows easy concatenation of files,
	// comments in files, and identification of errors by line number.
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	pl := make(PolicyList, 0)

	decoder := abac.Codecs.UniversalDecoder()

	i := 0
	unversionedLines := 0
	for scanner.Scan() {
		i++
		p := &abac.Policy{}
		b := scanner.Bytes()

		// skip comment lines and blank lines
		trimmed := strings.TrimSpace(string(b))
		if len(trimmed) == 0 || strings.HasPrefix(trimmed, "#") {
			continue
		}

		decodedObj, _, err := decoder.Decode(b, nil, nil)
		if err != nil {
			if !(runtime.IsMissingVersion(err) || runtime.IsMissingKind(err) || runtime.IsNotRegisteredError(err)) {
				return nil, policyLoadError{path, i, b, err}
			}
			unversionedLines++
			// Migrate unversioned policy object
			oldPolicy := &v0.Policy{}
			if err := runtime.DecodeInto(decoder, b, oldPolicy); err != nil {
				return nil, policyLoadError{path, i, b, err}
			}
			if err := abac.Scheme.Convert(oldPolicy, p, nil); err != nil {
				return nil, policyLoadError{path, i, b, err}
			}
			pl = append(pl, p)
			continue
		}

		decodedPolicy, ok := decodedObj.(*abac.Policy)
		if !ok {
			return nil, policyLoadError{path, i, b, fmt.Errorf("unrecognized object: %#v", decodedObj)}
		}
		pl = append(pl, decodedPolicy)
	}

	if unversionedLines > 0 {
		klog.Warningf("Policy file %s contained unversioned rules. See docs/admin/authorization.md#abac-mode for ABAC file format details.", path)
	}

	if err := scanner.Err(); err != nil {
		return nil, policyLoadError{path, -1, nil, err}
	}
	return pl, nil
}
`,
	}, nil, &result)

	assert.NoError(t, err)
	assert.Equal(t, "0", result.Code)
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Data.List)

	// Verify that we got some results
	if len(result.Data.List) > 0 {
		firstNode := result.Data.List[0]
		assert.NotEmpty(t, firstNode.FilePath)
		assert.NotNil(t, firstNode.Position)
		assert.NotNil(t, firstNode.Content)
		assert.NotEmpty(t, firstNode.Content)
	}
}
