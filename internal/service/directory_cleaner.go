package service

import (
	"net"
	"os"
	"path/filepath"
	"search-gin/pkg/utils"
	"strings"
)

// stackItem 目录遍历栈项
type stackItem struct {
	path       string
	queryChild bool
	visited    bool
}

// removeWalk 迭代方式删除空目录
func removeWalk(baseDir string, deep bool) {
	dirStack := []stackItem{{path: baseDir, queryChild: deep, visited: false}}

	for len(dirStack) > 0 {
		current := dirStack[len(dirStack)-1]
		dirStack = dirStack[:len(dirStack)-1]
		currentDir := current.path
		visited := current.visited

		if !visited {
			files, err := os.ReadDir(currentDir)
			if err != nil {
				utils.InfoFormat("读取目录失败: %s, 错误: %v", currentDir, err)
				continue
			}

			if len(files) > 0 && current.queryChild {
				dirStack = append(dirStack, stackItem{path: currentDir, queryChild: current.queryChild, visited: true})

				for _, fi := range files {
					pathAbs := filepath.Join(currentDir, fi.Name())
					if fi.IsDir() {
						dirStack = append(dirStack, stackItem{path: pathAbs, queryChild: current.queryChild, visited: false})
					}
				}
			} else if len(files) == 0 {
				if err := os.Remove(currentDir); err != nil {
					utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
				}
			}
		} else {
			if files, err := os.ReadDir(currentDir); err == nil && len(files) == 0 {
				if err := os.Remove(currentDir); err != nil {
					utils.InfoFormat("删除空目录失败: %s, 错误: %v", currentDir, err)
				}
			}
		}
	}
}

// DeleteOne 删除指定文件夹下的指定文件名的文件
func (d *searchService) DeleteOne(dirName string, fileName string) {
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
			d.UpDirClear(dirName)
		}
	}
}

// DownDeleteDir 迭代方式删除文件夹及其内容
func (d *searchService) DownDeleteDir(dirname string) {
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
	d.UpDirClear(parentDir)
}

// UpDirClear 迭代方式向上删除空文件夹
func (d *searchService) UpDirClear(dirname string) {
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
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip := strings.Split(localAddr.String(), ":")[0]
	return ip
}
