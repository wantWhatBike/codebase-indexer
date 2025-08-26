package codegraph

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"github.com/zgsm-ai/codebase-indexer/pkg/utils"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zgsm-ai/codebase-indexer/internal/store/codegraph/codegraphpb"
	"google.golang.org/protobuf/proto"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/sourcegraph/scip/bindings/go/scip"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const dbName = "badger.db"

const openMaxRetries = 5
const openRetryInterval = time.Second
const eachSymbolKeepResult = 2

// BadgerDBGraph implements GraphStore using BadgerDB
type BadgerDBGraph struct {
	basePath string
	db       *badger.DB
}

func NewBadgerDBGraph(opts ...GraphOption) (GraphStore, error) {
	b := &BadgerDBGraph{}
	for _, opt := range opts {
		opt(b)
	}
	dbPath := filepath.Join(b.basePath, dbName)
	// 自定义 BadgerDB 配置
	badgerOpts := badger.DefaultOptions(dbPath)

	// 值日志配置
	badgerOpts.ValueLogFileSize = 1 << 30 // 512 << 20 512MB // 1 << 30 1GB
	badgerOpts.ValueThreshold = 128       // 小于128字节的值直接存储在LSM树中

	// 内存表配置
	badgerOpts.NumMemtables = 2              // 增加内存表数量
	badgerOpts.NumLevelZeroTables = 50       // 增加L0层表数量
	badgerOpts.NumLevelZeroTablesStall = 100 // 增加L0层表数量阈值

	// 其他优化
	badgerOpts.BlockSize = 4 * 1024        // 4KB
	badgerOpts.BloomFalsePositive = 0.01   // 降低误判率
	badgerOpts.SyncWrites = false          // 异步写入
	badgerOpts.DetectConflicts = false     // 禁用冲突检测
	badgerOpts.Compression = options.ZSTD  // 使用ZSTD压缩
	badgerOpts.ZSTDCompressionLevel = 3    // 压缩级别
	badgerOpts.CompactL0OnClose = true     // 关闭时压缩L0层
	badgerOpts.VerifyValueChecksum = false // 禁用校验和验证

	// 设置索引缓存大小
	badgerOpts.IndexCacheSize = 256 << 20 // 256MB

	badgerOpts = badgerOpts.WithLoggingLevel(badger.WARNING)

	badgerDB, err := openWithRetry(badgerOpts, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open graph database:%w", err)
	}

	b.db = badgerDB
	return b, nil
}

func openWithRetry(badgerOpts badger.Options, dbPath string) (*badger.DB, error) {
	var badgerDB *badger.DB
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("badger db open failed with panic, err: %v", r)
		}
	}()

	for i := 1; i <= openMaxRetries; i++ {

		badgerDB, err = badger.Open(badgerOpts)
		if err == nil {
			break
		}
		logx.Errorf("badger db  open failed, retrying %d/%d", i, openMaxRetries)
		// （1）获取锁失败；（2）Manifest文件损坏；（3）table不存在；（4）打开内存表失败；
		//  file does not exist for table (recreate)
		if strings.Contains(err.Error(), "file does not exist for table") ||
			strings.Contains(err.Error(), "Manifest file might be corrupted") {
			logx.Errorf("start to delete badgerdb path %s for recreate.", dbPath)
			if rErr := os.RemoveAll(dbPath); rErr != nil {
				logx.Errorf("delete badgerdb path %s for recreate, err:%v", dbPath, rErr)
			}
		}
		time.Sleep(openRetryInterval)
	}
	return badgerDB, err
}

type GraphOption func(*BadgerDBGraph)

func WithPath(basePath string) GraphOption {
	return func(b *BadgerDBGraph) {
		b.basePath = basePath
	}
}

// BatchWrite 批量写入文档和符号
func (b BadgerDBGraph) BatchWrite(ctx context.Context, docs []*codegraphpb.Document) error {
	wb := b.db.NewWriteBatch()
	// 写入文档
	for _, doc := range docs {
		docBytes, err := SerializeDocument(doc)
		if err != nil {
			return err
		}
		if err := wb.Set(DocKey(doc.Path), docBytes); err != nil {
			return err
		}
	}
	return wb.Flush()
}

func (b BadgerDBGraph) BatchWriteCodeStructures(ctx context.Context, docs []*codegraphpb.CodeDefinition) error {
	wb := b.db.NewWriteBatch()
	// 写入文档
	for _, doc := range docs {
		docBytes, err := SerializeDocument(doc)
		if err != nil {
			return err
		}
		if err := wb.Set(StructKey(doc.Path), docBytes); err != nil {
			return err
		}
	}
	return wb.Flush()
}

// QueryRelations 实现查询接口
func (b BadgerDBGraph) QueryRelations(ctx context.Context, opts *types.RelationRequest) ([]*types.GraphNode, error) {
	startTime := time.Now()
	defer func() {
		tracer.WithTrace(ctx).Infof("QueryRelations execution time: %d ms", time.Since(startTime).Milliseconds())
	}()

	// 1. 获取文档
	var doc codegraphpb.Document
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(DocKey(opts.FilePath))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("document not found: %s", opts.FilePath)
			}
			return fmt.Errorf("failed to get document: %w", err)
		}
		return item.Value(func(val []byte) error {
			return DeserializeDocument(val, &doc)
		})
	})
	if err != nil {
		tracer.WithTrace(ctx).Errorf("Failed to get document: %v", err)
		return nil, err
	}

	var res []*types.GraphNode
	var foundSymbols []*codegraphpb.Symbol

	// Find root symbols based on query options
	if opts.SymbolName != "" {
		foundSymbols = b.querySymbolsByNameAndLine(&doc, opts)
		tracer.WithTrace(ctx).Debugf("Found %d symbols by name and line", len(foundSymbols))
	} else {
		foundSymbols = b.querySymbolsByPosition(ctx, &doc, opts)
		tracer.WithTrace(ctx).Debugf("Found %d symbols by position", len(foundSymbols))
	}

	// Check if any root symbols were found
	if len(foundSymbols) == 0 {
		err := fmt.Errorf("symbol not found: name %s startLine %d in document %s", opts.SymbolName, opts.StartLine, opts.FilePath)
		tracer.WithTrace(ctx).Errorf("%v", err)
		return nil, err
	}

	// root
	// 找定义节点，以定义节点为根节点进行深度遍历
	for _, s := range foundSymbols {
		// 如果当前Symbol 就是定义，加入
		if s.Role == codegraphpb.RelationType_RELATION_DEFINITION {
			res = append(res, &types.GraphNode{
				FilePath:   doc.Path,
				SymbolName: s.Name,
				Identifier: s.Identifier,
				Position:   types.ToPosition(s.Range),
				NodeType:   string(types.NodeTypeDefinition),
			})
			continue
		}
		// 不是定义节点，找它的relation中的定义节点
		relations := s.Relations
		if len(relations) == 0 {
			continue
		}
		for _, r := range relations {
			if r.RelationType == codegraphpb.RelationType_RELATION_DEFINITION {
				// 定义节点，加入root
				res = append(res, &types.GraphNode{
					FilePath:   r.FilePath,
					SymbolName: r.Name,
					Identifier: r.Identifier,
					Position:   types.ToPosition(r.Range),
					NodeType:   string(types.NodeTypeDefinition),
				})
			}
		}
	}

	tracer.WithTrace(ctx).Debugf("Found %d root nodes", len(res))

	// Build the rest of the tree recursively
	// We need to build children for the root nodes found
	for _, rootNode := range res {
		// Pass the corresponding original symbol proto to the recursive function
		b.buildChildrenRecursive(ctx, rootNode, opts.MaxLayer)
	}
	return res, nil
}

// QueryDefinitions 查询定义
func (b BadgerDBGraph) QueryDefinitions(ctx context.Context, opts *types.DefinitionRequest, pc *parser.ProjectConfig) ([]*types.DefinitionNode, error) {
	startTime := time.Now()
	defer func() {
		tracer.WithTrace(ctx).Infof("query definitions cost %d ms", time.Since(startTime).Milliseconds())
	}()

	var res []*types.DefinitionNode

	var foundSymbols []*codegraphpb.Symbol
	queryStartLine := int32(opts.StartLine - 1)
	queryEndLine := int32(opts.EndLine - 1)

	snippet := opts.CodeSnippet

	// 根据代码片段中的标识符名模糊搜索
	if snippet != types.EmptyString {
		// 调用tree_sitter 解析，获取所有的标识符及位置
		baseParser := parser.NewBaseParser()
		parsedData, err := baseParser.Parse(ctx, &types.SourceFile{
			Path:    opts.FilePath,
			Content: []byte(snippet),
		}, parser.ParseOptions{
			IncludeContent: false,
			ProjectConfig:  pc,
		})
		if err != nil {
			return nil, fmt.Errorf("faled to parse code snippet for definition query: %w", err)
		}
		imports := parsedData.Imports
		elements := parsedData.Elements
		// TODO 找到所有的外部依赖, 当前只处理call
		var callNames []string
		for _, e := range elements {
			if c, ok := e.(*parser.Call); ok {
				callNames = append(callNames, c.Name)
			}
		}
		if len(callNames) == 0 {
			return nil, fmt.Errorf("no function/method call found in code snippet")
		}

		// 根据所找到的call 的name + imports， 去模糊匹配symbol
		foundSymbolKeys, err := b.searchSymbolNames(ctx, callNames, imports)
		if err != nil {
			return nil, fmt.Errorf("failed to search function/method call names: %w", err)
		}
		if len(foundSymbolKeys) == 0 {
			return nil, fmt.Errorf("failed to search symbol by function/method call names")
		}
		for _, key := range foundSymbolKeys {
			var document codegraphpb.Document
			if err = b.loadValue(ctx, key.DocKey, &document); err != nil {
				tracer.WithTrace(ctx).Errorf("Failed to get document by key: %s, err:%v", string(key.DocKey), err)
				continue
			}
			if symbol := b.findSymbolInDocByRange(ctx, &document, key.Range); symbol != nil {
				foundSymbols = append(foundSymbols, symbol)
			} else {
				tracer.WithTrace(ctx).Errorf("Failed to find symbol in document %s by range: %v",
					string(key.DocKey), key.Range)
			}
		}

	} else {
		// 1. 获取文档
		var doc codegraphpb.Document
		err := b.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get(DocKey(opts.FilePath))
			if err != nil {
				if errors.Is(err, badger.ErrKeyNotFound) {
					return fmt.Errorf("document not found: %s", opts.FilePath)
				}
				return fmt.Errorf("failed to get document: %w", err)
			}
			return item.Value(func(val []byte) error {
				return DeserializeDocument(val, &doc)
			})
		})
		if err != nil {
			tracer.WithTrace(ctx).Errorf("Failed to get document: %v", err)
			return nil, err
		}

		foundSymbols = b.findSymbolInDocByLineRange(ctx, &doc, queryStartLine, queryEndLine)

	}
	// 去重, 如果range 出现过，则剔除
	existedDefs := make(map[string]bool)
	// 遍历，封装结果，取定义
	for _, s := range foundSymbols {
		// 本身是定义
		if s.Role == codegraphpb.RelationType_RELATION_DEFINITION {
			// 去掉本范围内的定义，仅过滤范围查询，不过滤全文检索
			if len(s.Range) > 0 && snippet != types.EmptyString && isInLinesRange(s.Range[0], queryStartLine, queryEndLine) {
				continue
			}
			res = append(res, &types.DefinitionNode{
				FilePath: s.Path,
				Name:     s.Name,
				Position: types.ToPosition(s.Range),
			})

			continue
		} else if s.Role == codegraphpb.RelationType_RELATION_REFERENCE { // 引用
			for _, r := range s.Relations {
				// 引用的 relation 是定义
				if r.RelationType == codegraphpb.RelationType_RELATION_DEFINITION {
					// 如果已经访问过，就跳过
					if isSymbolExists(r.FilePath, r.Range, existedDefs) {
						continue
					}
					existedDefs[symbolMapKey(r.FilePath, r.Range)] = true
					// 引用节点，加入node的children
					res = append(res, &types.DefinitionNode{
						FilePath: r.FilePath,
						Name:     r.Name, //TODO r.Name 有时候是identifier，未处理
						Position: types.ToPosition(r.Range),
					})
				}
			}
		} else {
			tracer.WithTrace(ctx).Errorf("QueryDefinitions: unsupported symbol %s type %s ", s.Identifier, string(s.Role))
		}
	}
	// 根据struct_doc 重新封装它的 position，变为范围。doc的range是单行。
	if err := b.refillDefinitionRange(ctx, res); err != nil {
		tracer.WithTrace(ctx).Errorf("QueryDefinitions:  refill definition range err:%v", utils.TruncateError(err))
	}

	return res, nil
}

func isInLinesRange(current, start, end int32) bool {
	return current >= start-1 && current <= end-1
}

func isSymbolExists(filePath string, ranges []int32, state map[string]bool) bool {
	key := symbolMapKey(filePath, ranges)
	_, ok := state[key]
	return ok
}
func symbolMapKey(filePath string, ranges []int32) string {
	return filePath + "-" + utils.SliceToString(ranges)
}

// findSymbolInDocByIdentifier 在文档中查找指定名称的符号
func (b BadgerDBGraph) findSymbolInDocByIdentifier(doc *codegraphpb.Document, identifier string) *codegraphpb.Symbol {
	// TODO 使用Position 二分查找
	for _, sym := range doc.Symbols {
		if sym.Identifier == identifier {
			return sym
		}
	}
	return nil
}

// findSymbolInDocByIdentifier 在结构文件中符号在该范围的symbol
func (b BadgerDBGraph) findSymbolInStruct(ctx context.Context, doc *codegraphpb.CodeDefinition, position types.Position) *codegraphpb.Symbol {
	line := position.StartLine
	column := position.StartColumn
	if line == 0 && column == 0 {
		tracer.WithTrace(ctx).Errorf("findSymbolInStruct invalid position :%v, length less than 2", position)
		return nil
	}
	// TODO 二分查找
	var foundDef *codegraphpb.Definition
	for _, d := range doc.Definitions {
		parsedRange, err := scip.NewRange(d.Range)
		if err != nil {
			tracer.WithTrace(ctx).Errorf("findSymbolInStruct parse range error:%w", err)
			return nil
		}
		if parsedRange.Contains(scip.Position{Line: int32(line), Character: int32(column)}) {
			foundDef = d
			break
		}
	}

	if foundDef == nil {
		tracer.WithTrace(ctx).Debugf("findSymbolInStruct definition not found by position %v in doc: %s", position, doc.Path)
		return nil
	}
	// 找到了def, 下一步根据def 的path、 range，找 symbol
	var document codegraphpb.Document
	err := b.loadValue(ctx, DocKey(doc.Path), &document)
	if err != nil {
		tracer.WithTrace(ctx).Debugf("findSymbolInStruct document not found by path %v in doc: %s", position, doc.Path)
		return nil
	}
	return b.findSymbolInDocByRange(ctx, &document, foundDef.Range)

}

// buildChildrenRecursive recursively builds the child nodes for a given GraphNode and its corresponding Symbol.
func (b BadgerDBGraph) buildChildrenRecursive(ctx context.Context, node *types.GraphNode, maxLayer int) {
	if maxLayer <= 0 || node == nil {
		tracer.WithTrace(ctx).Debugf("buildChildrenRecursive stopped: maxLayer=%d, node is nil=%v", maxLayer, node == nil)
		return
	}
	maxLayer-- // 防止死递归

	startTime := time.Now()
	defer func() {
		tracer.WithTrace(ctx).Debugf("buildChildrenRecursive for node %s took %v", node.Identifier, time.Since(startTime))
	}()

	symbolPath := node.FilePath
	identifier := node.Identifier
	position := node.Position

	// 根据path和position，定义到 symbol，从而找到它的relation
	var document codegraphpb.Document
	err := b.loadValue(ctx, DocKey(symbolPath), &document)
	if err != nil {
		tracer.WithTrace(ctx).Errorf("Failed to find document for path %s: %v", symbolPath, err)
		return
	}

	symbol := b.findSymbolInDocByIdentifier(&document, identifier)
	if symbol == nil {
		tracer.WithTrace(ctx).Debugf("Symbol not found in document: path=%s, identifier=%s", symbolPath, identifier)
		return
	}

	var children []*types.GraphNode

	// 找到symbol 的relation. 只有定义的symbol 有reference，引用节点的relation是定义节点
	if len(symbol.Relations) > 0 {
		for _, r := range symbol.Relations {
			if r.RelationType == codegraphpb.RelationType_RELATION_REFERENCE {
				// 引用节点，加入node的children
				children = append(children, &types.GraphNode{
					FilePath:   r.FilePath,
					SymbolName: r.Name,
					Identifier: r.Identifier,
					Position:   types.ToPosition(r.Range),
					NodeType:   string(types.NodeTypeReference),
				})
			}
		}
		tracer.WithTrace(ctx).Debugf("Found %d reference relations for symbol %s", len(children), identifier)
	}

	if len(children) == 0 {
		// 如果references 为空，说明当前 node 是引用节点， 找到它属于哪个函数/类/结构体，再找它的definition节点，再找引用
		var structFile codegraphpb.CodeDefinition
		err = b.loadValue(ctx, StructKey(symbolPath), &structFile)
		if err != nil {
			tracer.WithTrace(ctx).Errorf("Failed to find struct file for path %s: %v", symbolPath, err)
			return
		}
		if structFile.Path == types.EmptyString { // 没找到
			tracer.WithTrace(ctx).Debugf("Cannot find symbol %s struct file by symbol path %s", identifier, symbolPath)
		}
		// 定义symbol
		foundDefSymbol := b.findSymbolInStruct(ctx, &structFile, position)
		if foundDefSymbol == nil {
			tracer.WithTrace(ctx).Debugf("No definition found for symbol at position %+v", position)
			return
		}
		children = append(children, &types.GraphNode{
			FilePath:   foundDefSymbol.Path,
			SymbolName: foundDefSymbol.Name,
			Identifier: foundDefSymbol.Identifier,
			Position:   types.ToPosition(foundDefSymbol.Range),
			NodeType:   string(types.NodeTypeDefinition),
		})
		tracer.WithTrace(ctx).Debugf("Found definition for reference node: %s", foundDefSymbol.Identifier)
	}

	//当前节点的子
	node.Children = children

	// 继续递归
	for _, ch := range children {
		b.buildChildrenRecursive(ctx, ch, maxLayer)
	}
}

// loadValue 查找并反序列化文档
func (b BadgerDBGraph) loadValue(ctx context.Context, key []byte, data proto.Message) error {
	if len(key) == 0 {
		return fmt.Errorf("invalid input: key is empty")
	}
	if data == nil {
		return fmt.Errorf("invalid input: data is nil")
	}

	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil // Not finding the doc is not an error we should stop for
			}
			return fmt.Errorf("failed to get document: %w", err)
		}
		return item.Value(func(val []byte) error {
			if len(val) == 0 {
				return fmt.Errorf("empty document value")
			}
			if err := DeserializeDocument(val, data); err != nil {
				return fmt.Errorf("failed to deserialize document: %w", err)
			}
			return nil
		})
	})

	if err != nil {
		tracer.WithTrace(ctx).Errorf("loadValue error for key %s: %v", string(key), err)
	}
	return err
}

// querySymbolsByNameAndLine 通过 symbolName + startLine
func (b BadgerDBGraph) querySymbolsByNameAndLine(doc *codegraphpb.Document, opts *types.RelationRequest) []*codegraphpb.Symbol {
	var nodes []*codegraphpb.Symbol
	queryName := opts.SymbolName
	// 根据名字和 行号， 找到symbol
	for _, s := range doc.Symbols {
		// symbol 名字 模糊匹配
		if strings.Contains(s.Identifier, queryName) {
			symbolRange := s.Range
			if symbolRange != nil && len(symbolRange) > 0 {
				if symbolRange[0] == int32(opts.StartLine-1) {
					nodes = append(nodes, s)
				}
			}
		}
	}
	return nodes
}

// querySymbolsByPosition 按位置查询 occurrence
func (b BadgerDBGraph) querySymbolsByPosition(ctx context.Context, doc *codegraphpb.Document, opts *types.RelationRequest) []*codegraphpb.Symbol {
	var nodes []*codegraphpb.Symbol
	scipPosition, err := toScipPosition([]int32{int32(opts.StartLine), int32(opts.StartColumn)})
	if err != nil {
		tracer.WithTrace(ctx).Errorf("toScipPosition error: %v", err)
		return nodes
	}
	for _, s := range doc.Symbols {
		// symbol 名字 模糊匹配
		if len(s.Range) == 0 {
			continue
		}
		sRange, err := scip.NewRange(s.Range)
		if err != nil {
			tracer.WithTrace(ctx).Errorf("parse doc range error: %v, range:%v", err, s.Range)
			continue
		}
		if sRange.Contains(scipPosition) {
			nodes = append(nodes, s)
		}

	}
	return nodes
}

// Close 关闭数据库连接
func (b BadgerDBGraph) Close() error {
	if b.db.IsClosed() {
		return nil
	}
	if err := b.db.RunValueLogGC(0.5); err != nil {
		_ = fmt.Errorf("failed to run value log GC, err:%v", err)
	}
	return b.db.Close()
}

// DeleteAll 删除所有数据并执行一次清理
func (b BadgerDBGraph) DeleteAll(ctx context.Context) error {
	// 删除所有数据
	if err := b.db.DropAll(); err != nil {
		return fmt.Errorf("codegraph delete all indexes error:%w", err)
	}
	if err := b.db.RunValueLogGC(0.1); err != nil {
		tracer.WithTrace(ctx).Errorf("codegraph failed to run value log GC, err:%v", err)
	}
	// 执行一次gc
	return nil
}

func (b BadgerDBGraph) DB() *badger.DB {
	return b.db
}

// TODO 这个symbol 得和scip统一，要找到 name 的position
func (b BadgerDBGraph) findSymbolInDocByRange(ctx context.Context, document *codegraphpb.Document, symbolRange []int32) *codegraphpb.Symbol {
	//TODO 二分查找
	for _, s := range document.Symbols {
		// s
		// 开始行
		if len(s.Range) < 2 {
			tracer.WithTrace(ctx).Debugf("findSymbolInDocByRange invalid range in doc:%s, less than 2: %v", s.Identifier, s.Range)
			continue
		}
		// 开始行、(TODO 列一致)   这里，当前tree-sitter 捕获的是 整个函数体，而scip则是name，暂时先只通过行号处理（要确保local被过滤）
		if s.Range[0] == symbolRange[0] {
			return s
		}
	}
	return nil
}

func (b BadgerDBGraph) Delete(ctx context.Context, files []string) error {
	tracer.WithTrace(ctx).Debugf("start to delete docs:%v", files)
	if len(files) == 0 {
		return nil
	}
	var docKeys [][]byte
	for _, v := range files {
		if v == types.EmptyString {
			tracer.WithTrace(ctx).Errorf("DeleteByCodebase docs, file path is empty")
			continue
		}
		docKeys = append(docKeys, DocKey(v))
		docKeys = append(docKeys, StructKey(v))
	}

	err := b.db.DropPrefix(docKeys...)
	tracer.WithTrace(ctx).Debugf("docs delete end:%v", docKeys)
	return err
}

func (b BadgerDBGraph) findSymbolInDocByLineRange(ctx context.Context, doc *codegraphpb.Document, startLine int32, endLine int32) []*codegraphpb.Symbol {
	var res []*codegraphpb.Symbol
	for _, s := range doc.Symbols {
		// s
		// 开始行
		if len(s.Range) < 2 {
			tracer.WithTrace(ctx).Debugf("findSymbolInDocByLineRange invalid range in doc:%s, less than 2: %v", s.Identifier, s.Range)
			continue
		}
		if s.Range[0] > endLine {
			break
		}
		// 开始行、(TODO 列一致)   这里，当前tree-sitter 捕获的是 整个函数体，而scip则是name，暂时先只通过行号处理（要确保local被过滤）
		if s.Range[0] >= startLine && s.Range[0] <= endLine {
			res = append(res, s)
		}
	}
	return res
}

func (b BadgerDBGraph) refillDefinitionRange(ctx context.Context, nodes []*types.DefinitionNode) error {
	var errs []error
	for _, n := range nodes {
		line := n.Position.StartLine - 1
		if line < 0 {
			errs = append(errs, fmt.Errorf("refillDefinitionRange node %s %s invalid line :%d, length less than 0", n.FilePath, n.Name, line))
			continue
		}
		var structFile codegraphpb.CodeDefinition
		err := b.loadValue(ctx, StructKey(n.FilePath), &structFile)
		if err != nil {
			errs = append(errs, fmt.Errorf("refillDefinitionRange node %s %s ,failed to find struct file by line %d: %v", n.FilePath, n.Name, line, err))
			continue
		}
		if structFile.Path == types.EmptyString { // 没找到
			errs = append(errs, fmt.Errorf("refillDefinitionRange node %s %s ,struct file not found", n.FilePath, n.Name))
			continue
		}

		// TODO 二分查找
		var foundDef *codegraphpb.Definition
		for _, d := range structFile.Definitions {
			if len(d.Range) == 0 {
				errs = append(errs, fmt.Errorf("refillDefinitionRange node %s %s , struct doc range length is 0", n.FilePath, n.Name))
				continue
			}
			if d.Range[0] == int32(line) {
				foundDef = d
				break
			}
		}
		if foundDef == nil { // 目前发现变量的定义会找不到
			errs = append(errs, fmt.Errorf("refillDefinitionRange definition not found by line %d in doc: %s", line, structFile.Path))
			continue
		}
		n.Position = types.ToPosition(foundDef.Range)
		n.NodeType = foundDef.Type
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (b BadgerDBGraph) DeleteByCodebase(ctx context.Context, codebaseId int32, codebasePath string) error {
	return b.DeleteAll(ctx)
}

func (b BadgerDBGraph) GetIndexSummary(ctx context.Context, codebaseId int32, codebasePath string) (*types.CodeGraphSummary, error) {
	start := time.Now()
	var relationFileCount int
	var definitionFileCount int
	err := b.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{})
		defer iter.Close()
		for iter.Rewind(); iter.Valid(); iter.Next() {
			key := iter.Item().Key()
			if isDocKey(key) {
				relationFileCount++
			} else if isStructKey(key) {
				definitionFileCount++
			} else {
				//tracer.WithTrace(ctx).Debugf("GetIndexSummary unknown key type:%s", string(key))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	tracer.WithTrace(ctx).Infof("codegraph getIndexSummary end, cost %d ms on total %d relation files, %d definition files",
		time.Since(start).Milliseconds(), relationFileCount, definitionFileCount)
	return &types.CodeGraphSummary{
		TotalFiles:           relationFileCount,
		TotalDefinitionFiles: definitionFileCount,
	}, nil
}

func (b BadgerDBGraph) BatchWriteDefSymbolKeysMap(ctx context.Context, defSymbolKeysMap map[string]*codegraphpb.KeySet) error {
	wb := b.db.NewWriteBatch()
	for key, docKeys := range defSymbolKeysMap {
		docKeysBytes, err := SerializeDocument(docKeys)
		if err != nil {
			return err
		}
		if err := wb.Set(SymbolIndexKey(key), docKeysBytes); err != nil {
			return err
		}
	}
	return wb.Flush()
}

func (b BadgerDBGraph) searchSymbolNames(ctx context.Context, names []string, imports []*parser.Import) ([]*codegraphpb.KeyRange, error) {
	start := time.Now()
	// 去重
	names = utils.DeDuplicate(names)
	foundKeys := make(map[string][]*codegraphpb.KeyRange)
	err := b.db.View(func(txn *badger.Txn) error {
		iter := txn.NewIterator(badger.IteratorOptions{})
		defer iter.Close()
		for iter.Rewind(); iter.Valid(); iter.Next() {
			key := iter.Item().Key()
			if !isSymbolIndexKey(key) {
				continue
			}
			symbolName := strings.TrimPrefix(string(key), symIndexPrefix)
			var matchName string
			for _, name := range names {
				if symbolName == name { //TODO 是否包含特殊符号？ 将 namespace 放入 KeyRange, 用于过滤（map->[keyRange]）
					matchName = name
					break
				}

			}
			if matchName == types.EmptyString {
				continue
			}

			item, err := txn.Get(key)
			if err != nil {
				tracer.WithTrace(ctx).Errorf("searchSymbolNames get key %s error:%v", string(key), err)
				continue
			}
			var keyset codegraphpb.KeySet
			err = item.Value(func(val []byte) error {
				if len(val) == 0 {
					return fmt.Errorf("empty symbol index value")
				}
				if err = DeserializeDocument(val, &keyset); err != nil {
					return fmt.Errorf("failed to deserialize document: %w", err)
				}
				return nil
			})
			if len(keyset.Keys) == 0 {
				continue
			}
			if _, ok := foundKeys[matchName]; !ok {
				foundKeys[matchName] = make([]*codegraphpb.KeyRange, 0)
			}
			foundKeys[matchName] = append(foundKeys[matchName], keyset.Keys...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// TODO 同一个name可能检索出多条数据，再根据import 过滤一遍。 namespace 要么用. 要么用/ 得判断
	total := 0
	for _, v := range foundKeys {
		total += len(v)
	}

	filteredResult := make([]*codegraphpb.KeyRange, 0, total)

	// TODO 根据 imports 过滤
	if len(imports) > 0 {
		for _, v := range foundKeys {
			for _, doc := range v {
				for _, imp := range imports {
					for _, fp := range imp.FilePaths {
						if strings.Contains(string(doc.DocKey), fp) || strings.Contains(fp, string(doc.DocKey)) { //TODO , go work ，多模块等特殊情况
							filteredResult = append(filteredResult, doc)
							break
						}
					}
				}
			}
		}
	}

	if len(filteredResult) == 0 {
		tracer.WithTrace(ctx).Debugf("result after filter is empty, use origin result ans keep 2 with each symbol")
		// 每个仅保留2个
		for _, v := range foundKeys {
			if len(v) > eachSymbolKeepResult {
				v = v[:eachSymbolKeepResult]
			}
			filteredResult = append(filteredResult, v...)
		}
	}
	tracer.WithTrace(ctx).Infof("codegraph symbol name search end, cost %d ms, names: %d, key found:%d, filtered result:%d",
		time.Since(start).Milliseconds(), len(names), total, len(filteredResult))
	return filteredResult, nil
}
