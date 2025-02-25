package pdf

import (
	"fmt"

	"github.com/jung-kurt/gofpdf"
	"github.com/yugo-ibuki/docrawl/internal/crawler"
)

// Generator はPDFを生成する構造体
type Generator struct {
	outputPath string
}

// NewGenerator は新しいGeneratorインスタンスを作成する
func NewGenerator(outputPath string) *Generator {
	return &Generator{
		outputPath: outputPath,
	}
}

// GeneratePDF はクロールしたページからPDFを生成する
func (g *Generator) GeneratePDF(pages []crawler.Page) error {
	if len(pages) == 0 {
		return fmt.Errorf("PDFを生成するページがありません")
	}

	// PDFを作成
	pdf := gofpdf.New("P", "mm", "A4", "")

	// フォントを設定
	pdf.AddUTF8Font("NotoSans", "", "")
	pdf.SetFont("NotoSans", "", 11)

	// 各ページについて
	for _, page := range pages {
		// 新しいPDFページを追加
		pdf.AddPage()

		// ヘッダー: ページタイトルとURL
		pdf.SetFont("NotoSans", "B", 16)
		pdf.CellFormat(190, 10, page.Title, "0", 1, "C", false, 0, "")

		pdf.SetFont("NotoSans", "I", 8)
		pdf.CellFormat(190, 5, page.URL, "0", 1, "C", false, 0, "")

		pdf.Ln(10)

		// 本文
		pdf.SetFont("NotoSans", "", 11)
		cleanText := cleanTextForPDF(page.Content)

		// テキストの分割と改行処理
		lines := splitTextIntoLines(cleanText, 80)
		for _, line := range lines {
			if line == "" {
				pdf.Ln(5) // 空行の場合は行間を空ける
			} else {
				pdf.MultiCell(190, 5, line, "0", "L", false)
			}
		}

		// ページ番号
		pdf.SetY(-15)
		pdf.SetFont("NotoSans", "I", 8)
		pageNum := fmt.Sprintf("%d / %d", pdf.PageNo(), len(pages))
		pdf.CellFormat(0, 10, pageNum, "0", 0, "C", false, 0, "")
	}

	// PDFを保存
	return pdf.OutputFileAndClose(g.outputPath)
}

// cleanTextForPDF はPDFに表示するテキストをクリーンアップする
func cleanTextForPDF(text string) string {
	// HTMLタグの簡易的な除去（より高度な処理が必要な場合はHTMLパーサーを使用）
	inTag := false
	var result []rune

	for _, r := range text {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			// タグの後に空白を挿入
			result = append(result, ' ')
			continue
		}
		if !inTag {
			result = append(result, r)
		}
	}

	return string(result)
}

// splitTextIntoLines はテキストを行に分割する
func splitTextIntoLines(text string, maxChars int) []string {
	var lines []string
	var currentLine string

	for _, r := range text {
		currentLine += string(r)

		// 改行文字の処理
		if r == '\n' {
			lines = append(lines, currentLine[:len(currentLine)-1])
			currentLine = ""
			continue
		}

		// 行の長さが最大文字数に達した場合
		if len(currentLine) >= maxChars {
			// 空白を探して分割
			lastSpace := -1
			for i := len(currentLine) - 1; i >= 0; i-- {
				if currentLine[i] == ' ' {
					lastSpace = i
					break
				}
			}

			if lastSpace > 0 {
				lines = append(lines, currentLine[:lastSpace])
				currentLine = currentLine[lastSpace+1:]
			} else {
				// 空白が見つからない場合は強制的に分割
				lines = append(lines, currentLine)
				currentLine = ""
			}
		}
	}

	// 残りのテキストを追加
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
