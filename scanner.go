package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ScanConfig 结构体存储了所有用于执行扫描操作的配置参数。
type ScanConfig struct {
	StartDir              string
	FilenameMode          string
	OutputFormat          string
	MaxDepth              int
	StopKeywords          map[string]struct{}
	StopCaseSensitive     bool
	ExcludeTypes          map[string]struct{}
	ExcludeCaseSensitive  bool
	FilenameKeywords      map[string]struct{}
	FilenameCaseSensitive bool
}

// renamedFileInfo 是一个自定义的fs.FileInfo实现。
type renamedFileInfo struct {
	fs.FileInfo
	name string
}

// Name 返回完整的路径。
func (r renamedFileInfo) Name() string { return r.name }

// runScan 在内存中生成扫描结果数据，而不是直接写入文件。
func runScan(config ScanConfig) (interface{}, string, error) {
	var walkErr error

	switch config.OutputFormat {
	case "Excel":
		f := excelize.NewFile()
		sheetName := "Scan Results"
		index, _ := f.NewSheet(sheetName)
		f.DeleteSheet("Sheet1")
		f.SetActiveSheet(index)

		headers := []string{"名称", "类型", "相对路径", "修改时间", "创建时间"}
		style, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
		for i, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, header)
			f.SetCellStyle(sheetName, cell, cell, style)
		}
		f.SetColWidth(sheetName, "A", "A", 35)
		f.SetColWidth(sheetName, "B", "B", 10)
		f.SetColWidth(sheetName, "C", "C", 60)
		f.SetColWidth(sheetName, "D", "E", 22)

		rowNum := 2
		walkErr = performWalk(config, func(d fs.DirEntry, info fs.FileInfo) {
			relPath, _ := filepath.Rel(config.StartDir, info.Name())
			itemType := "文件"
			if d.IsDir() {
				itemType = "文件夹"
			}
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", rowNum), d.Name())
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowNum), itemType)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowNum), filepath.ToSlash(relPath))
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowNum), info.ModTime().Format("2006-01-02 15:04:05"))
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowNum), getCreateTime(info).Format("2006-01-02 15:04:05"))
			rowNum++
		})

		if walkErr != nil {
			return nil, "", walkErr
		}
		return f, "scan_results.xlsx", nil

	case "TXT":
		var buf bytes.Buffer
		writer := bufio.NewWriter(&buf)

		_, _ = writer.WriteString("名称@类型@相对路径@修改时间@创建时间\n")
		walkErr = performWalk(config, func(d fs.DirEntry, info fs.FileInfo) {
			relPath, _ := filepath.Rel(config.StartDir, info.Name())
			itemType := "文件"
			if d.IsDir() {
				itemType = "文件夹"
			}
			line := fmt.Sprintf("%s@%s@%s@%s@%s\n",
				d.Name(),
				itemType,
				filepath.ToSlash(relPath),
				info.ModTime().Format("2006-01-02 15:04:05"),
				getCreateTime(info).Format("2006-01-02 15:04:05"))
			_, _ = writer.WriteString(line)
		})
		writer.Flush()

		if walkErr != nil {
			return nil, "", walkErr
		}
		return buf.Bytes(), "scan_results.txt", nil

	default:
		return nil, "", fmt.Errorf("未知的输出格式: %s", config.OutputFormat)
	}
}

// performWalk 是核心的目录遍历函数。
// 此版本旨在实现高性能和高正确性的统一，解决了之前版本的所有已知问题。
func performWalk(config ScanConfig, onMatch func(d fs.DirEntry, info fs.FileInfo)) error {
	// --- 关键修正 ---
	// 1. 在遍历开始前，仅执行一次路径规范化和计算。
	// 2. 使用 filepath.Abs 将起始路径转换为绝对路径，以消除相对路径（如 "."）带来的歧义。
	absStartDir, err := filepath.Abs(config.StartDir)
	if err != nil {
		// 如果起始路径无效，则无法继续，直接返回错误。
		return fmt.Errorf("无法解析起始路径 '%s': %w", config.StartDir, err)
	}

	// 3. 计算“基础路径”的组件数量。这是整个深度计算的核心。
	//    - 使用 ToSlash 统一路径分隔符为 "/"。
	//    - 使用 TrimRight 清理结尾的斜杠，解决最原始的 bug。
	//    - 这样得到的组件数量是一个稳定、可靠的基准值。
	basePathPartCount := len(strings.Split(strings.TrimRight(filepath.ToSlash(absStartDir), "/"), "/"))

	return filepath.WalkDir(config.StartDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("访问 %q 时出错: %v", path, err)
			return nil
		}

		// 跳过根目录本身（因为它深度为0，不应被计入第1层）
		if path == config.StartDir {
			return nil
		}

		// --- 核心逻辑 ---
		// 1. 首先，无条件对每个遍历到的项进行“是否记录”的判断
		if shouldLogItem(d, config) {
			info, err := d.Info()
			if err != nil {
				log.Printf("获取 %q 的信息时出错: %v", path, err)
				return nil
			}
			infoWithName := renamedFileInfo{FileInfo: info, name: path}
			onMatch(d, infoWithName)
		}

		// 2. 然后，如果这个项是目录，再判断是否要“深入”
		if d.IsDir() {
			// 将当前路径也转换为绝对路径，以匹配基准值的计算方式
			absPath, err := filepath.Abs(path)
			if err != nil {
				log.Printf("无法获取绝对路径 for %q: %v", path, err)
				return nil
			}

			// 使用与基准值完全相同的方法来计算当前路径的组件数
			currentPathPartCount := len(strings.Split(filepath.ToSlash(absPath), "/"))

			// 深度就是二者的差值
			currentDepth := currentPathPartCount - basePathPartCount

			// 如果当前目录的深度已经达到或超过了限制，则使用 SkipDir 跳过，不再深入。
			if config.MaxDepth != -1 && currentDepth >= config.MaxDepth {
				return fs.SkipDir
			}

			// 原始的停止关键字逻辑保持不变
			dirName := d.Name()
			if !config.StopCaseSensitive {
				dirName = strings.ToLower(dirName)
			}
			for keyword := range config.StopKeywords {
				if strings.Contains(dirName, keyword) {
					log.Printf("命中停止关键字，记录目录 '%s' 但跳过其内部。", path)
					return fs.SkipDir
				}
			}
		}

		return nil
	})
}

// shouldLogItem 根据配置决定一个给定的目录项是否应该被包含在结果中。
func shouldLogItem(item fs.DirEntry, config ScanConfig) bool {
	itemName := item.Name()
	if !item.IsDir() {
		ext := filepath.Ext(itemName)
		extToCheck := ext
		if !config.ExcludeCaseSensitive {
			extToCheck = strings.ToLower(ext)
		}
		if _, found := config.ExcludeTypes[extToCheck]; found {
			return false
		}
	}
	if len(config.FilenameKeywords) > 0 {
		itemNameToCheck := itemName
		if !config.FilenameCaseSensitive {
			itemNameToCheck = strings.ToLower(itemName)
		}
		isMatched := false
		for keyword := range config.FilenameKeywords {
			if strings.Contains(itemNameToCheck, keyword) {
				isMatched = true
				break
			}
		}
		if config.FilenameMode == "blacklist" && isMatched {
			return false
		}
		if config.FilenameMode == "whitelist" && !isMatched {
			return false
		}
	}
	return true
}

// parseKeywords 将一个以空格分隔的字符串解析为一个map，用于快速查找。
func parseKeywords(text string, caseSensitive bool) map[string]struct{} {
	keywords := make(map[string]struct{})
	parts := strings.Fields(text)
	for _, p := range parts {
		keyword := p
		if !caseSensitive {
			keyword = strings.ToLower(keyword)
		}
		if keyword != "" {
			keywords[keyword] = struct{}{}
		}
	}
	return keywords
}

// getCreateTime 是一个跨平台的辅助函数。
func getCreateTime(fi fs.FileInfo) time.Time {
	return getCreateTimePlatform(fi)
}
