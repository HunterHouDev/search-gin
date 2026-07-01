package utils

import (
	"fmt"
	"hash/fnv"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ValidatePath 验证路径是否在允许的目录内，防止路径遍历攻击
// allowedDirs: 允许的目录列表
// userPath: 用户提供的路径
// 返回：清理后的绝对路径和错误
func ValidatePath(userPath string, allowedDirs []string) (string, error) {
	// 清理路径（移除 .. 等）
	cleanPath := filepath.Clean(userPath)

	// 获取绝对路径
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("无效的路径: %v", err)
	}

	// 检查路径是否以允许的目录开头
	for _, allowedDir := range allowedDirs {
		// 清理允许的目录路径
		cleanAllowed := filepath.Clean(allowedDir)
		absAllowed, err := filepath.Abs(cleanAllowed)
		if err != nil {
			continue
		}

		// 检查路径是否在允许的目录内
		if absPath == absAllowed || strings.HasPrefix(absPath, absAllowed+string(filepath.Separator)) {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("路径访问被拒绝: 不在允许的目录范围内")
}

// DirpathForId 根据文件路径生成唯一 ID（FNV-1a 哈希，确定性、零分配）
func DirpathForId(path string) string {
	h := fnv.New64a()
	h.Write([]byte(path))
	id := fmt.Sprintf("%x", h.Sum64())
	return id
}

func ConcatSuffix(path string, suffix string) string {
	oldSuffix := GetSuffix(path)
	idx := strings.LastIndex(path, "."+oldSuffix)
	if idx < 0 {
		return path + "." + suffix
	}
	return path[:idx] + "." + suffix
}

func ExistsFiles(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		// 只有明确确认文件存在才返回 true，权限错误等其他情况一律视为不存在
		return false
	}
	return true
}

func GetSuffix(filename string) string {

	var suffix string
	if filename == "" {
		return suffix
	}
	suffix = filepath.Ext(filename)
	suffix = strings.ToLower(suffix)
	if strings.Contains(suffix, ".") {
		suffix = strings.TrimPrefix(suffix, ".")
	}
	return suffix

}
func GetMovieType(fileName string) string {

	code := ""
	rights := strings.Split(fileName, "{{")
	if len(rights) <= 1 {
		return "无"
	}
	for index, value := range rights {
		if index == 0 {
			continue
		}
		right := value
		lefts := strings.Split(right, "}}")
		for _, left := range lefts {
			return left
		}
	}
	return code

}

// GetTitle 获取文件名
func GetTitle(filename string) string {
	result := ""
	if filename == "" {
		return result
	}
	lastSuffix := path.Ext(filename)
	filename = strings.TrimSuffix(filename, lastSuffix)
	return filename

}

// GetAuthor 根据 文件名称  分析番号 [] 中包含 '-'符号...
func GetAuthor(fileName string) string {
	code := ""
	rights := strings.Split(fileName, "[")
	if len(rights) <= 1 {
		title := GetTitle(fileName)
		if len(title) > 20 {
			return title[0:20]
		}
		return title
	}
	for index, value := range rights {
		if index == 0 {
			continue
		}
		right := value
		lefts := strings.Split(right, "]")
		for _, left := range lefts {
			if !strings.Contains(left, "-") {
				return left
			}
		}
	}
	return code
}

func GetTags(fileName string, movieType string) []string {
	var res []string
	if movieType != "" {
		res = append(res, movieType)
	}
	rights := strings.Split(fileName, "《")
	if len(rights) <= 1 {
		return nil
	}
	for index, value := range rights {
		if index == 0 {
			continue
		}
		right := value
		lefts := strings.Split(right, "》")
		arr := strings.Split(lefts[0], ",")
		for i := 0; i < len(arr); i++ {
			if arr[i] != "" && IndexOf(res, arr[i]) == -1 {
				res = append(res, arr[i])
			}
		}
	}

	return res
}
func GetTagStr(fileName string) string {

	rights := strings.Split(fileName, "《")
	if len(rights) <= 1 {
		return ""
	}
	for index, value := range rights {
		if index == 0 {
			continue
		}
		right := value
		lefts := strings.Split(right, "》")
		return lefts[0]
	}
	return ""
}

// GetCode 根据 文件名称  分析番号 [] 中包含 '-'符号...
func GetCode(fileName string) string {
	code := ""
	rights := strings.Split(fileName, "[")
	if len(rights) <= 1 {
		code = GetTitle(fileName)
	} else {
		for index, value := range rights {
			if index == 0 {
				continue
			}
			right := value
			lefts := strings.Split(right, "]")
			for _, left := range lefts {
				if strings.Contains(left, "-") || strings.Contains(left, "_") {
					return left
				} else {
					code = left
				}
			}
		}
		if strings.Contains(code, ".mp4") {
			code = strings.ReplaceAll(code, ".mp4", "")
		}
	}
	return strings.ToUpper(code)
}

func GetSeriesByCode(code string) string {
	rights := strings.Split(code, "-")
	if len(rights) > 1 {
		return rights[0]
	}
	return ""
}

func GetSizeStr(fSize int64) string {

	fileSize := float64(fSize)
	result := ""
	if fileSize <= 1024 {
		result = fmt.Sprintf("%.f", fileSize)
	} else if fileSize <= 1024*1024 {
		size := fileSize / 1024
		result = fmt.Sprintf("%.f", size) + " k"
	} else if fileSize <= 1024*1024*1024 {
		size := fileSize / (1024 * 1024)
		result = fmt.Sprintf("%.2f", size) + " M"
	} else if fileSize <= 1024*1024*1024*1024 {
		size := fileSize / (1024 * 1024 * 1024)
		result = fmt.Sprintf("%.2f", size) + " G"
	} else if fileSize <= 1024*1024*1024*1024*1024 {
		size := fileSize / (1024 * 1024 * 1024 * 1024)
		result = fmt.Sprintf("%.2f", size) + " T"
	} else {
		size := fileSize / (1024 * 1024 * 1024 * 1024)
		result = fmt.Sprintf("%.2f", size) + " T"
	}
	return result
}
