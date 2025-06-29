package codebase

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/parser"
	"github.com/zgsm-ai/codebase-indexer/internal/tracer"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/zgsm-ai/codebase-indexer/pkg/utils"

	"github.com/zgsm-ai/codebase-indexer/internal/config"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

const (
	defaultLocalFileMode = 0644
	defaultLocalDirMode  = 0755
)

var _ Store = &localCodebase{}

type localCodebase struct {
	cfg config.CodeBaseStoreConf
	mu  sync.RWMutex // 保护并发访问
}

func NewLocalCodebase(cfg config.CodeBaseStoreConf) (Store, error) {
	if cfg.Local.BasePath == types.EmptyString {
		return nil, errors.New("local base path cannot be empty")
	}
	return &localCodebase{
		cfg: cfg,
	}, nil
}

func (l *localCodebase) GetSyncFileListCollapse(ctx context.Context, codebasePath string) (*types.CollapseSyncMetaFile, error) {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCodebasePathNotExists
	}
	// filepath -> mode(add delete modify)
	// 根据元数据获取代码文件列表
	// 递归目录，进行处理，并发
	// 获取代码文件列表
	fileModeMap := make(map[string]string)
	list, err := l.List(ctx, codebasePath, types.SyncMedataDir, types.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("embeddingProcessor metadata dir is empty")
	}
	//TODO collapse list to fileList
	// 对目录下的文件按名字升序排序
	treeSet := utils.NewTimestampTreeSet()
	// sort
	for _, f := range list {
		treeSet.Add(f.Name)
	}
	var metaFileList []string
	it := treeSet.Iterator()
	for it.Next() {
		metadataFile := it.Value().(string)
		metaFileList = append(metaFileList, metadataFile)
		syncMetaData, err := l.Read(ctx, codebasePath, filepath.Join(types.SyncMedataDir, metadataFile), types.ReadOptions{})
		if err != nil {
			tracer.WithTrace(ctx).Errorf("read metadata file %v failed: %v", metadataFile, err)
			continue
		}
		if syncMetaData == nil {
			tracer.WithTrace(ctx).Errorf("sync file %s metadata is empty", metadataFile)
			continue
		}
		var syncMetaObj *types.SyncMetadataFile

		err = json.Unmarshal(syncMetaData, &syncMetaObj)
		if err != nil {
			tracer.WithTrace(ctx).Errorf("failed to unmarshal metadata error: %v, raw: %s", err, syncMetaData)
		}
		files := syncMetaObj.FileList
		for k, v := range files {
			// add delete modify
			fileModeMap[k] = v
		}

	}
	return &types.CollapseSyncMetaFile{CodebasePath: codebasePath,
		FileModelMap: fileModeMap, MetaFilePaths: metaFileList}, nil

}

func (l *localCodebase) Open(ctx context.Context, codebasePath string, filePath string) (io.ReadSeekCloser, error) {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCodebasePathNotExists
	}
	return os.Open(filepath.Join(codebasePath, filePath))
}

func (l *localCodebase) DeleteAll(ctx context.Context, codebasePath string) error {
	if codebasePath == types.EmptyString {
		return errors.New("codebasePath cannot be empty")
	}

	if !strings.HasPrefix(codebasePath, l.cfg.Local.BasePath) || codebasePath == "*" {
		return fmt.Errorf("illegal codebasePath:%s", codebasePath)
	}
	tracer.WithTrace(ctx).Infof("start to delete codebasePath [%s]", codebasePath)
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCodebasePathNotExists
	}
	err = os.RemoveAll(codebasePath)
	tracer.WithTrace(ctx).Infof("delete codebasePath [%s] successfully", codebasePath)
	return err
}

// Init 初始化一个新的代码库
func (l *localCodebase) Init(ctx context.Context, clientId string, clientPath string) (*types.Codebase, error) {
	if clientId == types.EmptyString || clientPath == types.EmptyString {
		return nil, errors.New("clientId and clientPath cannot be empty")
	}

	if l.cfg.Local.BasePath == types.EmptyString {
		return nil, errors.New("basePath cannot be empty")
	}

	// 生成唯一的路径
	uniquePath, err := generateCodebasePath(l.cfg.Local.BasePath, clientId, clientPath)
	if err != nil {
		return nil, err
	}
	// 直接使用 uniquePath，因为它已经包含了 basePath
	codebasePath := uniquePath
	// 创建目录
	err = os.MkdirAll(codebasePath, defaultLocalDirMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create codebase directory: %w", err)
	}

	return &types.Codebase{BasePath: codebasePath}, nil
}

func (l *localCodebase) Add(ctx context.Context, codebasePath string, source io.Reader, target string) error {
	if codebasePath == types.EmptyString {
		return errors.New("codebasePath cannot be empty")
	}
	if target == types.EmptyString {
		return errors.New("target cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCodebasePathNotExists
	}

	// 构建完整路径
	fullPath := filepath.Join(codebasePath, target)

	// 确保目标目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, defaultLocalDirMode); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// 创建目标文件
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// 复制内容
	_, err = io.Copy(file, source)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (l *localCodebase) Unzip(ctx context.Context, codebasePath string, source io.Reader) error {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCodebasePathNotExists
	}

	// Create a temporary file to store the zip content
	tmpFile, err := os.CreateTemp(types.EmptyString, "codebase-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, source); err != nil {
		return fmt.Errorf("failed to copy zip content: %w", err)
	}

	// Open the zip file
	zipReader, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer zipReader.Close()

	// Extract each file
	for _, file := range zipReader.File {
		filePath := filepath.Join(codebasePath, file.Name)

		// Check for zip slip vulnerability
		if !strings.HasPrefix(filePath, codebasePath) {
			return fmt.Errorf("illegal file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, defaultLocalDirMode); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), defaultLocalDirMode); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip file entry: %w", err)
		}

		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultLocalFileMode)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file: %w", err)
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			rc.Close()
			outFile.Close()
			return fmt.Errorf("failed to extract file: %w", err)
		}

		rc.Close()
		outFile.Close()
	}

	return nil
}

func (l *localCodebase) Delete(ctx context.Context, codebasePath string, path string) error {
	if codebasePath == types.EmptyString {
		return errors.New("codebasePath cannot be empty")
	}
	if path == types.EmptyString {
		return errors.New("path cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCodebasePathNotExists
	}

	fullPath := filepath.Join(codebasePath, path)
	return os.RemoveAll(fullPath)
}

func (l *localCodebase) MkDirs(ctx context.Context, codebasePath string, path string) error {
	if codebasePath == types.EmptyString || path == types.EmptyString {
		return errors.New("codebasePath and path cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCodebasePathNotExists
	}

	fullPath := filepath.Join(codebasePath, path)
	return os.MkdirAll(fullPath, defaultLocalDirMode)
}

func (l *localCodebase) Exists(ctx context.Context, codebasePath string, path string) (bool, error) {
	if codebasePath == types.EmptyString {
		return false, errors.New("codebasePath cannot be empty")
	}

	fullPath := filepath.Join(codebasePath, path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (l *localCodebase) Stat(ctx context.Context, codebasePath string, path string) (*types.FileInfo, error) {
	if codebasePath == types.EmptyString || path == types.EmptyString {
		return nil, errors.New("codebasePath and path cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCodebasePathNotExists
	}

	fullPath := filepath.Join(codebasePath, path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	return &types.FileInfo{
		Name:    info.Name(),
		Path:    fullPath,
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime(),
		Mode:    info.Mode(),
	}, nil
}

func (l *localCodebase) List(ctx context.Context, codebasePath string, dir string, option types.ListOptions) ([]*types.FileInfo, error) {
	if codebasePath == types.EmptyString {
		return nil, errors.New("codebasePath cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCodebasePathNotExists
	}

	fullPath := filepath.Join(codebasePath, dir)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []*types.FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 应用过滤规则
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(entry.Name()) {
			continue
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(entry.Name()) {
			continue
		}

		// 跳过隐藏文件和目录
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// 只返回文件，不返回目录
		if entry.IsDir() {
			continue
		}

		relPath := filepath.Join(dir, entry.Name())
		files = append(files, &types.FileInfo{
			Name:    entry.Name(),
			Path:    relPath,
			Size:    info.Size(),
			IsDir:   false,
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
		})
	}

	return files, nil
}

func (l *localCodebase) Tree(ctx context.Context, codebasePath string, subDir string, option types.TreeOptions) ([]*types.TreeNode, error) {
	if codebasePath == types.EmptyString {
		return nil, errors.New("codebasePath cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCodebasePathNotExists
	}

	// 使用 map 来构建目录树
	nodeMap := make(map[string]*types.TreeNode)
	walkBasePath := filepath.Join(codebasePath, subDir)

	err = filepath.Walk(walkBasePath, func(absFilePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏文件和目录
		if utils.IsHiddenFile(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 获取相对路径，相对codebasePath
		codeBaseRelativePath, err := filepath.Rel(codebasePath, absFilePath)
		if err != nil {
			return err
		}
		// 获取相对路径，相对codebasePath + subdir
		walkBaseRelativePath, err := filepath.Rel(walkBasePath, absFilePath)
		if err != nil {
			return err
		}

		// 应用过滤规则
		if option.ExcludePattern != nil && option.ExcludePattern.MatchString(walkBaseRelativePath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if option.IncludePattern != nil && !option.IncludePattern.MatchString(walkBaseRelativePath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 检查深度限制
		if option.MaxDepth > 0 {
			// 相对根+subdir 的depth
			depth := len(strings.Split(walkBaseRelativePath, string(filepath.Separator)))
			if depth > option.MaxDepth {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		var currentPath string
		var parts []string

		// 如果是根目录本身，跳过
		if walkBaseRelativePath == "." || utils.PathEqual(walkBaseRelativePath, subDir) {
			return nil
		}

		// 如果是根目录下的文件或目录
		if !strings.Contains(walkBaseRelativePath, string(filepath.Separator)) {
			currentPath = walkBaseRelativePath
			parts = []string{walkBaseRelativePath}
		} else {
			// 处理子目录中的文件和目录
			parts = strings.Split(walkBaseRelativePath, string(filepath.Separator))
			currentPath = parts[0]
		}

		// 处理路径中的每一级
		for i, part := range parts {
			if part == "" {
				continue
			}

			if i > 0 {
				currentPath = filepath.Join(currentPath, part)
			}

			// 如果节点已存在，跳过
			if _, exists := nodeMap[currentPath]; exists {
				continue
			}

			// 创建新节点
			isLast := i == len(parts)-1
			var size int64
			if isLast && !info.IsDir() {
				size = info.Size()
			}

			node := &types.TreeNode{
				FileInfo: types.FileInfo{
					Name:    part,
					Path:    codeBaseRelativePath,
					IsDir:   isLast && info.IsDir(),
					Size:    size,
					ModTime: info.ModTime(),
					Mode:    info.Mode(),
				},
				Children: make([]*types.TreeNode, 0),
			}

			// 将节点添加到 map
			nodeMap[currentPath] = node

			// 如果不是根级节点，添加到父节点的子节点列表
			if i > 0 {
				parentPath := filepath.Dir(currentPath)
				if parent, exists := nodeMap[parentPath]; exists {
					parent.Children = append(parent.Children, node)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// 构建根节点列表
	var rootNodes []*types.TreeNode
	for path, node := range nodeMap {
		if !strings.Contains(path, string(filepath.Separator)) {
			rootNodes = append(rootNodes, node)
		}
	}

	return rootNodes, nil
}

func (l *localCodebase) Read(ctx context.Context, codebasePath string, filePath string, option types.ReadOptions) ([]byte, error) {
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCodebasePathNotExists
	}

	fullPath := filepath.Join(codebasePath, filePath)

	// 如果StartLine <= 0，设置为1
	if option.StartLine <= 0 {
		option.StartLine = 1
	}

	// 打开文件
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 创建reader来读取文件
	reader := bufio.NewReader(file)
	var lines []string
	lineNum := 1

	// 读取行
	for {
		// 读取一行，允许超过默认缓冲区大小
		line, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		// 处理可能被截断的行
		var lineBuffer []byte
		lineBuffer = append(lineBuffer, line...)
		for isPrefix {
			line, isPrefix, err = reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			lineBuffer = append(lineBuffer, line...)
		}

		// 转换为字符串
		lineStr := string(lineBuffer)

		// 如果当前行号大于等于StartLine，则添加到结果中
		if lineNum >= option.StartLine {
			// 如果EndLine > 0 且当前行号大于EndLine，则退出
			if option.EndLine > 0 && lineNum > option.EndLine {
				break
			}
			lines = append(lines, lineStr)
		}
		lineNum++
	}

	// 将结果转换为字节数组
	return []byte(strings.Join(lines, "\n")), nil
}

func (l *localCodebase) Walk(ctx context.Context, codebasePath string, dir string, walkFn WalkFunc, walkOpts WalkOptions) error {
	if codebasePath == types.EmptyString {
		return errors.New("codebasePath cannot be empty")
	}

	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCodebasePathNotExists
	}

	fullDir := filepath.Join(codebasePath, dir)
	return filepath.Walk(fullDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil && !walkOpts.IgnoreError {
			return err
		}

		// 跳过隐藏文件和目录
		if utils.IsHiddenFile(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relativePath, err := filepath.Rel(codebasePath, filePath)
		if err != nil && !walkOpts.IgnoreError {
			return err
		}

		if relativePath == "." {
			return nil
		}
		fileExt := filepath.Ext(relativePath)
		if slices.Contains(walkOpts.ExcludeExts, fileExt) {
			return nil
		}

		if len(walkOpts.IncludeExts) > 0 && !slices.Contains(walkOpts.IncludeExts, fileExt) {
			return nil
		}

		for _, p := range walkOpts.ExcludePrefixes {
			if strings.HasPrefix(relativePath, p) {
				return nil
			}
		}

		for _, p := range walkOpts.IncludePrefixes {
			if !strings.HasPrefix(relativePath, p) {
				return nil
			}
		}

		// Convert Windows filePath separators to forward slashes
		relativePath = filepath.ToSlash(relativePath)

		// 只处理文件，不处理目录
		if info.IsDir() {
			return nil
		}

		// 构建 WalkContext
		walkCtx := &WalkContext{
			Path:         filePath,
			RelativePath: relativePath,
			Info: &types.FileInfo{
				Name:    info.Name(),
				Path:    relativePath,
				Size:    info.Size(),
				IsDir:   false,
				ModTime: info.ModTime(),
				Mode:    info.Mode(),
			},
			ParentPath: filepath.Dir(filePath),
		}
		file, err := os.Open(filePath)
		if err != nil && !walkOpts.IgnoreError {
			return err
		}
		if file == nil {
			return nil
		}
		defer file.Close()
		return walkFn(walkCtx, file)
	})
}

func (l *localCodebase) BatchDelete(ctx context.Context, codebasePath string, paths []string) error {
	if codebasePath == types.EmptyString {
		return errors.New("codebasePath cannot be empty")
	}
	exists, err := l.Exists(ctx, codebasePath, types.EmptyString)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCodebasePathNotExists
	}
	if len(paths) == 0 {
		return errors.New("batch delete paths cannot be empty")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(paths))

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			if err := l.Delete(ctx, codebasePath, p); err != nil {
				errChan <- fmt.Errorf("failed to delete %s: %w", p, err)
			}
		}(path)
	}

	wg.Wait()
	close(errChan)

	// 收集所有错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("batch delete errors: %v", errs)
	}
	return nil
}

func (l *localCodebase) ResolveSourceRoot(ctx context.Context, codebasePath string, language parser.Language) (string, error) {
	// SourceRootResolver 默认 SourceRoot 配置（按优先级排序）
	sourceRootResolver := getSourceRootResolver(ctx, codebasePath, l)
	resolver, ok := sourceRootResolver[language]
	if !ok {

		return types.EmptyString, ErrSourceRootResolverNotFound
	}
	return resolver()

}

func (l *localCodebase) InferLanguage(ctx context.Context, codebasePath string) (parser.Language, error) {
	start := time.Now()
	languageStats := make(map[parser.Language]int)
	analyzedFiles := 0

	err := l.Walk(ctx, codebasePath, types.EmptyString, func(walkCtx *WalkContext, reader io.ReadCloser) error {

		// 如果已经分析了足够多的文件，且某个语言占比超过60%，可以提前结束
		if analyzedFiles >= l.cfg.MinFilesToAnalyze {
			totalFiles := 0
			maxCount := 0
			for _, count := range languageStats {
				totalFiles += count
				if count > maxCount {
					maxCount = count
				}
			}
			if float64(maxCount)/float64(totalFiles) > 0.6 {
				return maxFileReached
			}
		}

		if analyzedFiles >= l.cfg.MaxFilesToAnalyze {
			return maxFileReached
		}
		// 跳过隐藏文件和目录
		if utils.IsHiddenFile(walkCtx.RelativePath) {
			return nil
		}

		ext := filepath.Ext(walkCtx.RelativePath)
		if ext == types.EmptyString {
			return nil
		}

		// 使用parser包中的语言配置
		langConfig, _ := parser.GetLangConfigByFilePath(walkCtx.RelativePath)
		if langConfig != nil {
			languageStats[langConfig.Language]++
		} else {
			return nil
		}

		analyzedFiles++
		return nil
	}, WalkOptions{
		IgnoreError: true,
	})

	tracer.WithTrace(ctx).Infof("scip_generator detect language analyzed %d files, cost %d ms",
		analyzedFiles, time.Since(start).Milliseconds())
	if err != nil && !errors.Is(err, maxFileReached) {
		return types.EmptyString, fmt.Errorf("failed to analyze codebase language: %v", err)
	}

	// 5. 选择出现频率最高的语言
	var dominantLanguage parser.Language
	maxCount := 0
	for lang, count := range languageStats {
		if count > maxCount {
			maxCount = count
			dominantLanguage = lang
		}
	}
	return dominantLanguage, nil
}
