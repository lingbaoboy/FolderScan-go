package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/skratchdot/open-golang/open"
	"github.com/xuri/excelize/v2"
)

// AppUI 结构体封装了应用程序的所有UI组件和状态。
type AppUI struct {
	win                   fyne.Window
	dirEntry              *widget.Entry
	depthEntry            *widget.Entry
	scanAllCheck          *widget.Check
	fnameKeywordEntry     *widget.Entry
	fnameKeywordCaseCheck *widget.Check
	fnameKeywordModeRadio *widget.RadioGroup
	stopKeywordEntry      *widget.Entry
	stopKeywordCaseCheck  *widget.Check
	excludeTypesEntry     *widget.Entry
	excludeTypesCaseCheck *widget.Check
	outputFormatRadio     *widget.RadioGroup
	startButton           *widget.Button
	statusBox             *fyne.Container
	resultContainer       *fyne.Container
	statusText            binding.String
	resultPath            binding.String
	resultVisible         binding.Bool
	isScanning            binding.Bool
	startDir              string
}

// createAndRunGUI 负责创建并运行应用的图形用户界面。
func createAndRunGUI() {
	a := app.NewWithID("com.lingbaoboy.folderscanner.v1.0")
	win := a.NewWindow("目录扫描与导出工具 v1.0")

	ui := &AppUI{win: win}

	ui.statusText = binding.NewString()
	ui.statusText.Set("请选择或输入起始目录，然后设置扫描参数")
	ui.resultPath = binding.NewString()
	ui.resultVisible = binding.NewBool()
	ui.resultVisible.Set(false)
	ui.isScanning = binding.NewBool()
	ui.isScanning.Set(false)

	ui.dirEntry = widget.NewEntry()
	ui.dirEntry.SetPlaceHolder("请选择或手动输入一个目录...")
	ui.dirEntry.OnSubmitted = func(text string) {
		ui.startDir = text
		ui.clearStatus()
	}
	dirSelectButton := widget.NewButton("选择...", ui.selectDirectory)
	dirBox := container.NewBorder(nil, nil, nil, dirSelectButton, ui.dirEntry)

	ui.depthEntry = widget.NewEntry()
	ui.depthEntry.SetText("3")
	ui.scanAllCheck = widget.NewCheck("遍历所有子文件夹", func(checked bool) {
		if checked {
			ui.depthEntry.Disable()
		} else {
			ui.depthEntry.Enable()
		}
	})
	ui.fnameKeywordEntry = widget.NewEntry()
	ui.fnameKeywordCaseCheck = widget.NewCheck("大小写敏感", nil)
	ui.fnameKeywordModeRadio = widget.NewRadioGroup([]string{"忽略含关键词项 (黑名单)", "仅保留含关键词项 (白名单)"}, nil)
	ui.fnameKeywordModeRadio.SetSelected("忽略含关键词项 (黑名单)")
	ui.fnameKeywordModeRadio.Horizontal = true

	ui.stopKeywordEntry = widget.NewEntry()
	ui.stopKeywordEntry.SetText("")
	ui.stopKeywordCaseCheck = widget.NewCheck("大小写敏感", nil)
	ui.excludeTypesEntry = widget.NewEntry()
	ui.excludeTypesEntry.SetText("")
	ui.excludeTypesCaseCheck = widget.NewCheck("大小写敏感", nil)
	ui.outputFormatRadio = widget.NewRadioGroup([]string{"Excel (.xlsx)", "Text (.txt)"}, nil)
	ui.outputFormatRadio.SetSelected("Excel (.xlsx)")
	ui.outputFormatRadio.Horizontal = true

	optionsForm := widget.NewForm(
		widget.NewFormItem("遍历层级:", container.NewBorder(nil, nil, nil, ui.scanAllCheck, ui.depthEntry)),
		widget.NewFormItem("名称关键词:", container.NewVBox(container.NewBorder(nil, nil, nil, ui.fnameKeywordCaseCheck, ui.fnameKeywordEntry), ui.fnameKeywordModeRadio)),
		widget.NewFormItem("目录停止关键字:", container.NewVBox(container.NewBorder(nil, nil, nil, ui.stopKeywordCaseCheck, ui.stopKeywordEntry), widget.NewLabel("多个关键字用空格分隔, 如: .D .M"))),
		widget.NewFormItem("排除文件类型:", container.NewVBox(container.NewBorder(nil, nil, nil, ui.excludeTypesCaseCheck, ui.excludeTypesEntry), widget.NewLabel("多个类型用空格分隔, 如: .pdf .docx"))),
		widget.NewFormItem("输出格式:", ui.outputFormatRadio),
	)

	ui.startButton = widget.NewButton("开始扫描", ui.startScan)
	ui.isScanning.AddListener(binding.NewDataListener(func() {
		scanning, _ := ui.isScanning.Get()
		if scanning {
			ui.startButton.SetText("正在扫描...")
			ui.startButton.Disable()
		} else {
			ui.startButton.SetText("开始扫描")
			ui.startButton.Enable()
		}
	}))
	statusLabel := widget.NewLabelWithData(ui.statusText)
	resultPathEntry := widget.NewEntryWithData(ui.resultPath)
	resultPathEntry.Disable()
	resultOpenBtn := widget.NewButton("打开结果", func() {
		path, _ := ui.resultPath.Get()
		if path != "" {
			_ = open.Run(path)
		}
	})
	ui.resultContainer = container.NewBorder(nil, nil, nil, resultOpenBtn, resultPathEntry)
	ui.resultVisible.AddListener(binding.NewDataListener(func() {
		visible, _ := ui.resultVisible.Get()
		if visible {
			ui.resultContainer.Show()
		} else {
			ui.resultContainer.Hide()
		}
	}))
	ui.statusBox = container.NewVBox(statusLabel, ui.resultContainer)
	ui.resultContainer.Hide()

	mailURL, _ := url.Parse("mailto:lingbaoboy@gmail.com")
	authorInfo := container.NewVBox(
		widget.NewLabel(""),
		widget.NewSeparator(),
		container.NewCenter(widget.NewLabel("作者：lingbaoboy")),
		container.NewCenter(widget.NewHyperlink("邮箱：lingbaoboy@gmail.com", mailURL)),
		container.NewCenter(widget.NewLabel("本软件使用 CC BY-NC 协议，仅限非商业用途。商业使用需取得作者授权。")),
	)

	content := container.NewVBox(
		dirBox,
		widget.NewSeparator(),
		container.NewVBox(widget.NewLabelWithStyle("扫描参数", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), optionsForm),
		widget.NewSeparator(),
		container.NewCenter(ui.startButton),
		widget.NewSeparator(),
		ui.statusBox,
		layout.NewSpacer(),
		authorInfo,
	)

	win.SetContent(container.New(layout.NewPaddedLayout(), content))
	win.Resize(fyne.NewSize(620, 680))
	win.ShowAndRun()
}

// selectDirectory 打开一个文件夹选择对话框，并更新UI。
func (ui *AppUI) selectDirectory() {
	dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil {
			dialog.ShowError(err, ui.win)
			return
		}
		if uri != nil {
			ui.startDir = uri.Path()
			ui.dirEntry.SetText(ui.startDir)
			ui.clearStatus()
		}
	}, ui.win)
}

// startScan 是“开始扫描”按钮的回调函数。
func (ui *AppUI) startScan() {
	ui.startDir = ui.dirEntry.Text
	if ui.startDir == "" {
		dialog.ShowError(fmt.Errorf("请选择或输入一个起始目录！"), ui.win)
		return
	}
	if _, err := os.Stat(ui.startDir); os.IsNotExist(err) {
		dialog.ShowError(fmt.Errorf("指定的目录不存在: %s", ui.startDir), ui.win)
		return
	}

	config := ScanConfig{StartDir: ui.startDir}
	var err error
	if ui.scanAllCheck.Checked {
		config.MaxDepth = -1
	} else {
		config.MaxDepth, err = strconv.Atoi(ui.depthEntry.Text)
		if err != nil || config.MaxDepth <= 0 {
			dialog.ShowError(fmt.Errorf("遍历层级必须是一个正整数！"), ui.win)
			return
		}
	}
	config.StopCaseSensitive = ui.stopKeywordCaseCheck.Checked
	config.StopKeywords = parseKeywords(ui.stopKeywordEntry.Text, config.StopCaseSensitive)
	config.ExcludeCaseSensitive = ui.excludeTypesCaseCheck.Checked
	config.ExcludeTypes = parseKeywords(ui.excludeTypesEntry.Text, config.ExcludeCaseSensitive)
	config.FilenameCaseSensitive = ui.fnameKeywordCaseCheck.Checked
	config.FilenameKeywords = parseKeywords(ui.fnameKeywordEntry.Text, config.FilenameCaseSensitive)
	if strings.HasPrefix(ui.fnameKeywordModeRadio.Selected, "忽略") {
		config.FilenameMode = "blacklist"
	} else {
		config.FilenameMode = "whitelist"
	}
	if strings.HasPrefix(ui.outputFormatRadio.Selected, "Excel") {
		config.OutputFormat = "Excel"
	} else {
		config.OutputFormat = "TXT"
	}

	ui.clearStatus()
	ui.isScanning.Set(true)

	go func() {
		defer ui.isScanning.Set(false)
		data, defaultFilename, err := runScan(config)
		if err != nil {
			ui.onScanError(err)
			return
		}

		fyne.Do(func() {
			ui.saveResultsWithPermissionsDialog(data, defaultFilename)
		})
	}()
}

// saveResultsWithPermissionsDialog 负责保存文件，并处理权限和文件占用错误。
func (ui *AppUI) saveResultsWithPermissionsDialog(data interface{}, defaultFilename string) {
	var attemptSave func(path string)

	saveToPath := func(path string) error {
		switch d := data.(type) {
		case *excelize.File:
			return d.SaveAs(path)
		case []byte:
			return os.WriteFile(path, d, 0666)
		default:
			return fmt.Errorf("不支持的数据类型，无法保存")
		}
	}

	attemptSave = func(path string) {
		err := saveToPath(path)

		if err == nil {
			ui.onScanComplete(path)
			return
		}

		if os.IsPermission(err) || strings.Contains(strings.ToLower(err.Error()), "access is denied") {
			saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, errDialog error) {
				if errDialog != nil {
					ui.onScanError(errDialog)
					return
				}
				if writer == nil {
					ui.statusText.Set("保存操作已取消。")
					return
				}
				defer writer.Close()

				var writeErr error
				switch d := data.(type) {
				case *excelize.File:
					writeErr = d.Write(writer)
				case []byte:
					_, writeErr = writer.Write(d)
				}

				if writeErr != nil {
					ui.onScanError(fmt.Errorf("保存到指定路径失败: %w", writeErr))
				} else {
					ui.onScanComplete(writer.URI().Path())
				}
			}, ui.win)

			saveDialog.SetFileName(defaultFilename)
			saveDialog.Show()
			return

		} else if strings.Contains(strings.ToLower(err.Error()), "used by another process") {
			dialog.NewConfirm("保存失败", "无法保存文件，因为它可能已被其他程序打开。\n请关闭文件后重试。", func(retry bool) {
				if retry {
					attemptSave(path)
				} else {
					ui.statusText.Set("保存操作已取消。")
				}
			}, ui.win).Show()
			return

		} else {
			ui.onScanError(fmt.Errorf("保存文件时出错: %w", err))
			return
		}
	}

	defaultPath := filepath.Join(ui.startDir, defaultFilename)
	attemptSave(defaultPath)
}

// onScanComplete 在扫描成功完成时被调用。
func (ui *AppUI) onScanComplete(path string) {
	ui.statusText.Set("扫描完成！结果已保存至:")
	ui.resultPath.Set(path)
	ui.resultVisible.Set(true)
}

// onScanError 在扫描过程中发生错误时被调用。
func (ui *AppUI) onScanError(err error) {
	errorMsg := fmt.Sprintf("扫描出错: %v", err)
	ui.statusText.Set(errorMsg)
	ui.resultVisible.Set(false)
	fyne.Do(func() {
		dialog.ShowError(fmt.Errorf(errorMsg), ui.win)
	})
}

// clearStatus 清除状态信息和上一次的扫描结果。
func (ui *AppUI) clearStatus() {
	if ui.startDir != "" {
		ui.statusText.Set("已选择目录: " + ui.startDir)
	} else {
		ui.statusText.Set("请选择或输入起始目录，然后设置扫描参数")
	}
	ui.resultVisible.Set(false)
}
