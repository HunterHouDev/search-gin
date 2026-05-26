package utils

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
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

// SanitizeFilename 清理文件名，防止命令注入
// 移除可能导致命令注入的字符
func SanitizeFilename(filename string) string {
	// 移除危险的shell元字符
	dangerous := []string{";", "|", "`", "$", "(", ")", "<", ">", "&", "!", "\n", "\r"}
	cleaned := filename
	for _, char := range dangerous {
		cleaned = strings.ReplaceAll(cleaned, char, "")
	}
	return cleaned
}

func DirpathForId(path string) (string, string) {
	//res, _ := url.QueryUnescape(path)
	//res = strings.ReplaceAll(res, PathSeparator+PathSeparator, PathSeparator)
	//res = strings.ReplaceAll(res, PathSeparator, "~")
	//res = strings.ReplaceAll(res, PathSeparator, "~")
	//res = strings.ReplaceAll(res, ":", "1")
	//res = strings.ReplaceAll(res, ".", "2")
	//res = strings.ReplaceAll(res, ",", "3")
	//res = strings.ReplaceAll(res, "!", "4")
	//res = strings.ReplaceAll(res, "》", "5")
	//res = strings.ReplaceAll(res, "《", "6")
	//arr := strings.Split(res, "~")
	newpath, _ := Encrypt(path)
	//for i := 0; i < len(arr); i++ {
	//	curArr := arr[i]
	//	length := len(curArr)
	//	if i != 0 {
	//		newpath += "~"
	//	}
	//	if length > 30 {
	//		// newpath += curArr[0:100]
	//		// newpath += fmt.Sprintf("%d", (length))
	//		// newpath += curArr[length-100 : length]
	//		j := 0
	//		for _, value := range curArr {
	//			if j%4 == 0 {
	//				newpath += string(value)
	//			}
	//			j++
	//		}
	//	} else {
	//		newpath += curArr
	//	}
	//
	//}
	return newpath, newpath
}

func ConcatSuffix(path string, suffix string) string {
	path = strings.ReplaceAll(path, GetSuffix(path), suffix)
	return path
}

func ExistsFiles(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		// 修正：应该检查文件是否不存在，而不是存在
		return !os.IsNotExist(err)
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

// GetActress 根据 文件名称  分析番号 [] 中包含 '-'符号...
func GetActress(fileName string) string {
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

// Camel2Case 驼峰式写法转为下划线写法
func Camel2Case(name string) string {
	buffer := NewBuffer()
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i != 0 {
				buffer.Append('_')
			}
			buffer.Append(unicode.ToLower(r))
		} else {
			buffer.Append(r)
		}
	}
	return buffer.String()
}

// 下划线写法转为驼峰写法
//func Case2Camel(name string) string {
//	name = strings.Replace(name, "_", " ", -1)
//	name = strings.Title(name)
//	return strings.Replace(name, " ", "", -1)
//}

// Buffer 内嵌bytes.Buffer，支持连写
type Buffer struct {
	*bytes.Buffer
}

func NewBuffer() *Buffer {
	return &Buffer{Buffer: new(bytes.Buffer)}
}

func (b *Buffer) Append(i interface{}) *Buffer {
	switch val := i.(type) {
	case int:
		b.append(strconv.Itoa(val))
	case int64:
		b.append(strconv.FormatInt(val, 10))
	case uint:
		b.append(strconv.FormatUint(uint64(val), 10))
	case uint64:
		b.append(strconv.FormatUint(val, 10))
	case string:
		b.append(val)
	case []byte:
		b.Write(val)
	case rune:
		b.WriteRune(val)
	}
	return b
}

func (b *Buffer) append(s string) *Buffer {
	defer func() {
		if err := recover(); err != nil {
			InfoFormat("*****内存不够了！******")
		}
	}()
	b.WriteString(s)
	return b
}
