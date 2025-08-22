package utils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestToUnixPath(t *testing.T) {
	// 测试用例
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "a//b/c", "a/b/c"},
		{"with dot", "a/./b/c", "a/b/c"},
		{"with parent", "a/b/../c", "a/c"},
		{"mixed separators", "a\\b\\c", "a/b/c"}, // 自动转换为 /
		{"root absolute", "/a/b/c", "/a/b/c"},    // 转为相对路径
		{"current dir", ".", "."},
		{"parent dir", "..", ".."},
		{"complex", "../../a/./b//c/..", "../../a/b"},
	}

	for _, tc := range testCases {
		result := ToUnixPath(tc.input)
		assert.Equal(t, tc.expected, result, fmt.Sprintf("Test case: %s", tc.name))
	}
}

func TestIsChild(t *testing.T) {
	tests := []struct {
		parent string
		path   string
		want   bool
	}{
		// 基本直接子文件
		{"a", "a/b.txt", true},
		{"a/", "a/b.txt", true},  // 修复：处理末尾斜杠
		{"a", "a/b/c.txt", true}, // 修复：处理多级子目录

		// 边缘情况
		{"a", "a", false},            // 修复：相同路径返回 false
		{"a", "b.txt", false},        // 正确
		{"a/b/", "b/a/b.txt", false}, // 正确
		{"a/b", "a/b/c/d.txt", true}, // 修复：深层子目录

		// 路径规范化
		{"a/b", "a/b/c/../d.txt", true}, // 修复：处理 ".."
		{"/a", "/a/b", true},            // 处理绝对路径
		{"a", "a/./b", true},            // 处理 "."
	}

	for _, tt := range tests {
		t.Run(tt.parent+"_"+tt.path, func(t *testing.T) {
			if got := IsChild(tt.parent, tt.path); got != tt.want {
				t.Errorf("IsChild() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHidden(t *testing.T) {
	testCases := []struct {
		path     string
		expected bool
	}{
		{".gitignore", true},         // 隐藏文件
		{"./.test", true},            // 隐藏文件（带前缀）
		{"/home/user/.config", true}, // 隐藏目录
		{"README.md", false},         // 普通文件
		{"./dir/file.txt", false},    // 普通文件路径
		{"/../.cache/file", true},    // 隐藏文件（带上级目录）
		{"//dir//.hidden//", true},   // 隐藏目录（带多余斜杠）
		{"/", false},                 // 根目录
		{"..", false},                // 上级目录
		{"./", false},                // 当前目录
		{"...hidden", true},          // 非隐藏文件（.在中间）
	}

	for _, tc := range testCases {
		tc := tc // 捕获循环变量
		t.Run(tc.path, func(t *testing.T) {
			result := IsHiddenFile(tc.path)
			require.Equal(t, tc.expected, result,
				"expected %v for path %q, got %v",
				tc.expected, tc.path, result)
		})
	}
}

// 测试用例结构体定义
type absToRelTest struct {
	name      string // 测试用例名称
	baseDir   string // 基准目录
	absPath   string // 绝对路径
	wantRel   string // 预期相对路径
	expectErr bool   // 是否预期产生错误
}

// 表格驱动测试
func TestAbsToRel(t *testing.T) {
	// 构建测试用例集合
	tests := []absToRelTest{
		{
			name:    "同一目录下的文件",
			baseDir: "/home/user/project",
			absPath: "/home/user/project/readme.md",
			wantRel: "readme.md",
		},
		{
			name:    "子目录下的文件",
			baseDir: "/home/user/project",
			absPath: "/home/user/project/docs/api.md",
			wantRel: "docs/api.md",
		},
		{
			name:    "父目录文件（跨层级）",
			baseDir: "/home/user/project/src",
			absPath: "/home/user/config.ini",
			wantRel: "../../config.ini",
		},
		{
			name:      "不同磁盘（Windows，预期错误）",
			baseDir:   "C:\\user\\docs",
			absPath:   "D:\\data\\file.txt",
			expectErr: true,
		},
		{
			name:    "同级目录",
			baseDir: "/a/b/c",
			absPath: "/a/b/d/file.txt",
			wantRel: "../d/file.txt",
		},
		{
			name:    "根目录下的文件",
			baseDir: "/",
			absPath: "/etc/hosts",
			wantRel: "etc/hosts",
		},
	}

	// 遍历执行测试用例
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRel, err := AbsToRel(tt.baseDir, tt.absPath)

			// 验证错误是否符合预期
			if (err != nil) != tt.expectErr {
				t.Errorf("AbsToRel() 错误 = %v, 预期错误 %v", err, tt.expectErr)
				return
			}

			// 如果预期有错误则不需要继续验证结果
			if tt.expectErr {
				return
			}

			// 统一路径分隔符后再比较（确保跨平台兼容性）
			gotRel = filepath.Clean(gotRel)
			wantRel := filepath.Clean(tt.wantRel)

			// 验证转换结果
			if gotRel != wantRel {
				t.Errorf("AbsToRel() = %v, 预期 %v", gotRel, wantRel)
			}
		})
	}
}
