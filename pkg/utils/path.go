package utils

import (
	"fmt"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// ToUnixPath 将相对路径转换为 Unix 风格（使用 / 分隔符，去除冗余路径元素）
func ToUnixPath(rawPath string) string {
	// path.Clean 会自动处理为 Unix 风格路径，去除多余的 /、. 和 ..
	filePath := path.Clean(rawPath)
	filePath = filepath.ToSlash(filePath)
	return strings.ReplaceAll(filePath, "\\", "/")
}

// PathEqual 比较路径是否相等，/ \ 转为 /
func PathEqual(a, b string) bool {
	return filepath.ToSlash(a) == filepath.ToSlash(b)
}

func IsChild(parent, path string) bool {
	// 确保路径规范化（处理斜杠、相对路径等）
	parent = ToUnixPath(filepath.Clean(parent))
	path = ToUnixPath(filepath.Clean(path))

	// 计算相对路径
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false // 无法计算相对路径（如跨磁盘）
	}

	// 相对路径不能以 ".." 开头，且不能等于 "."（即相同路径）
	return !strings.HasPrefix(rel, "..") && rel != "."
}

// IsHiddenFile 判断文件或目录是否为隐藏项
func IsHiddenFile(path string) bool {
	// 标准化路径，处理相对路径、符号链接等
	cleanPath := filepath.Clean(path)

	// 处理特殊路径
	if cleanPath == "." || cleanPath == ".." {
		return false
	}

	// 分割路径组件（兼容不同操作系统的路径分隔符）
	components := strings.Split(cleanPath, string(filepath.Separator))

	// 检查每个组件是否以"."开头（且不为空字符串）
	for _, comp := range components {
		if len(comp) > 0 && comp[0] == '.' {
			return true
		}
	}

	return false
}

// IsAbs 判断路径是否为绝对路径（自动适配 Windows、Linux、macOS）
// 只要符合任一系统的绝对路径规则，就返回 true
func IsAbs(path string) bool {
	if path == "" {
		return false
	}

	// 1. 判断是否为 Unix 风格绝对路径（Linux/macOS，以 / 开头）
	if path[0] == '/' {
		return true
	}

	// 2. 判断是否为 Windows 风格绝对路径
	// 2.1 UNC 路径（以 \\ 开头，如 \\server\share）
	if len(path) >= 2 && path[0] == '\\' && path[1] == '\\' {
		return true
	}

	// 2.2 盘符路径（如 C:\ 或 C:/）
	if len(path) >= 3 && path[1] == ':' {
		if (path[2] == '\\' || path[2] == '/') &&
			(path[0] >= 'A' && path[0] <= 'Z' || path[0] >= 'a' && path[0] <= 'z') {
			return true
		}
	}

	// 不符合任何系统的绝对路径规则
	return false
}

// AbsToUnixRel 将绝对路径转换为相对于基准目录的相对路径
// 支持Windows、Linux、macOS等操作系统
func AbsToUnixRel(baseDir, absPath string) (string, error) {
	baseDir = ToUnixPath(baseDir)
	absPath = ToUnixPath(absPath)
	// 如果本身是相对路径，直接返回
	if !IsAbs(absPath) {
		return absPath, nil
	}
	absBase := filepath.Clean(baseDir)
	// 2. 标准化目标绝对路径
	absPath = filepath.Clean(absPath)

	// 3. 特殊处理：Windows系统下检查是否同盘符
	if runtime.GOOS == "windows" {
		baseVol := filepath.VolumeName(absBase)
		pathVol := filepath.VolumeName(absPath)

		// 不同盘符无法转换为相对路径
		if baseVol != pathVol {
			return "", fmt.Errorf("基准目录(%s)与目标路径(%s)位于不同盘符，无法转换", baseVol, pathVol)
		}
	}

	// 4. 计算相对路径
	return strings.TrimPrefix(strings.ReplaceAll(absPath, baseDir, types.EmptyString), "/"), nil

}
