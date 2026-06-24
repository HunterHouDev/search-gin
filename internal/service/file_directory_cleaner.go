package service

import (
	"net"
	"os"
	"path/filepath"
	"search-gin/internal/model"
	"search-gin/pkg/utils"
	"strings"
)

// WalkOptions WalkInner 的控制参数
type WalkOptions struct {
	// 是否递归子目录
	Recursive bool
	// 文件类型白名单
	Types    []string
	// 根扫描目录列表（小目录检测 / 空目录清理时跳过这些目录自身）
	RootDirs []string
	// 是否清理空目录
	IsCleanEmpty bool
}

// WalkInner 栈式后序遍历目录树，默认收集匹配文件，opts 控制是否递归和清理空目录。
func WalkInner(baseDir string, opts WalkOptions) (allFiles []model.FileItem, dirSize int64) {
	typeSet := utils.ToSet(opts.Types)
	sizeMap := make(map[string]int64)

	type dirState struct {
		path      string
		visited   bool
		fileCount int
	}
	stack := []dirState{{path: baseDir, visited: false}}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !cur.visited {
			files, err := os.ReadDir(cur.path)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", cur.path, err)
				continue
			}

			var fileCount int
			var subDirs []string
			for _, f := range files {
				p := filepath.Join(cur.path, f.Name())
				if f.IsDir() {
					subDirs = append(subDirs, p)
				} else {
					fileCount++
					sizeMap[cur.path] = 0
					name := f.Name()
					suffix := utils.GetSuffix(name)
					info, err := f.Info()
					if err != nil {
						utils.InfoFormat("获取文件信息失败: %s, 错误: %v", p, err)
						continue
					}
					if utils.HasItemSet(typeSet, suffix) {
						movie := model.EasyFile(cur.path, p, name, suffix,
							info.Size(), info.ModTime(), baseDir)
						SetMovieNode(&movie)
						allFiles = append(allFiles, movie)
					}
					sizeMap[cur.path] += info.Size()
				}
			}

			stack = append(stack, dirState{path: cur.path, visited: true, fileCount: fileCount})

			if opts.Recursive && len(subDirs) > 0 {
				for i := len(subDirs) - 1; i >= 0; i-- {
					stack = append(stack, dirState{path: subDirs[i], visited: false})
				}
			}
		} else {
			if opts.IsCleanEmpty && cur.fileCount == 0 && utils.IndexOf(opts.RootDirs, cur.path) < 0 {
				if err := os.Remove(cur.path); err != nil {
					utils.InfoFormat("删除空目录失败: %s, 错误: %v", cur.path, err)
				}
			}
			currentSize := sizeMap[cur.path]
			if currentSize <= 20000000 && utils.IndexOf(opts.RootDirs, cur.path) < 0 {
				AppendSmallDir(model.NewFileInfoFold(cur.path, currentSize, true))
			}
			if cur.path != baseDir {
				parentPath := filepath.Dir(cur.path)
				sizeMap[parentPath] += currentSize
			}
		}
	}

	return allFiles, sizeMap[baseDir]
}

// stackItem 目录遍历栈项
type stackItem struct {
	path       string
	queryChild bool
	visited    bool
}

// DeleteOne 删除指定文件夹下的指定文件名的文件
func (s *searchService) DeleteOne(dirName string, fileName string) {
	if len(fileName) == 0 {
		return
	}

	files, err := os.ReadDir(dirName)
	if err != nil {
		utils.InfoFormat("读取目录失败: %s, 错误: %v", dirName, err)
		return
	}

	deleted := false
	for _, f := range files {
		if strings.HasPrefix(f.Name(), fileName) {
			path := filepath.Join(dirName, f.Name())
			if err := os.Remove(path); err != nil {
				utils.InfoFormat("删除文件失败: %s, 错误: %v", path, err)
			} else {
				deleted = true
			}
		}
	}

	if deleted {
		filesThen, err := os.ReadDir(dirName)
		if err != nil {
			utils.InfoFormat("读取目录失败: %s, 错误: %v", dirName, err)
			return
		}
		if len(filesThen) == 0 {
			s.UpDirClear(dirName)
		}
	}
}

// DownDeleteDir 迭代方式删除文件夹及其内容
func (s *searchService) DownDeleteDir(dirname string) {
	postOrderStack := []stackItem{{path: dirname, visited: false}}

	for len(postOrderStack) > 0 {
		current := postOrderStack[len(postOrderStack)-1]
		postOrderStack = postOrderStack[:len(postOrderStack)-1]
		currentPath := current.path
		visited := current.visited

		if !visited {
			files, err := os.ReadDir(currentPath)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentPath, err)
				continue
			}

			postOrderStack = append(postOrderStack, stackItem{path: currentPath, visited: true})

			for i := len(files) - 1; i >= 0; i-- {
				ff := files[i]
				path := filepath.Join(currentPath, ff.Name())
				if ff.IsDir() {
					postOrderStack = append(postOrderStack, stackItem{path: path, visited: false})
				} else {
					if err := os.Remove(path); err != nil {
						utils.InfoFormat("删除文件失败: %s, 错误: %v", path, err)
					}
				}
			}
		} else {
			if err := os.Remove(currentPath); err != nil {
				utils.InfoFormat("删除目录失败: %s, 错误: %v", currentPath, err)
			}
		}
	}

	parentDir := filepath.Dir(dirname)
	s.UpDirClear(parentDir)
}

// UpDirClear 迭代方式向上删除空文件夹
func (s *searchService) UpDirClear(dirname string) {
	currentDir := dirname

	for {
		if filepath.Clean(currentDir) == "/" || filepath.Dir(currentDir) == currentDir {
			break
		}

		files, err := os.ReadDir(currentDir)
		if err != nil {
			utils.InfoFormat("读取目录失败: %s, 错误: %v", currentDir, err)
			break
		}

		if len(files) == 0 {
			if err := os.Remove(currentDir); err != nil {
				utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
				break
			}
			currentDir = filepath.Dir(currentDir)
		} else {
			break
		}
	}
}

// GetIpAddr 获取本机 IP
func GetIpAddr() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		utils.InfoFormat("GetIpAddrError:%v \n\n", err)
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "127.0.0.1"
	}
	ip := strings.Split(localAddr.String(), ":")[0]
	return ip
}
