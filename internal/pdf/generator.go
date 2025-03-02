package pdf

import (
	"fmt"
	"os"
	"regexp"
	"strings"

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

// GeneratePDF はクロールしたページからPDFを生成する（今回はテキストファイルに変更）
func (g *Generator) GeneratePDF(pages []crawler.Page) error {
	if len(pages) == 0 {
		return fmt.Errorf("生成するページがありません")
	}

	// テキストファイルの出力パスを設定
	txtOutputPath := getTextPath(g.outputPath)

	// テキストファイルを生成
	err := generateTextFile(pages, txtOutputPath)
	if err != nil {
		return err
	}

	fmt.Printf("成功: %s が生成されました\n", txtOutputPath)
	return nil
}

// getTextPath はPDFのパスからテキストファイルのパスを生成する
func getTextPath(pdfPath string) string {
	// PDFの拡張子をTXTに変更
	if strings.HasSuffix(strings.ToLower(pdfPath), ".pdf") {
		return pdfPath[:len(pdfPath)-4] + ".txt"
	}
	// 拡張子がない場合や他の拡張子の場合はTXTを追加
	return pdfPath + ".txt"
}

// generateTextFile はページの内容からテキストファイルを生成する
func generateTextFile(pages []crawler.Page, outputPath string) error {
	// テキストファイルを作成
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("テキストファイルの作成に失敗しました: %w", err)
	}
	defer file.Close()

	// ヘッダー情報を書き込み
	fmt.Fprintf(file, "# ドキュメント収集結果\n")
	fmt.Fprintf(file, "# 取得ページ数: %d\n\n", len(pages))

	// 各ページの内容を書き込み
	for i, page := range pages {
		fmt.Fprintf(file, "=== ページ %d/%d ===\n", i+1, len(pages))
		fmt.Fprintf(file, "URL: %s\n", page.URL)
		fmt.Fprintf(file, "タイトル: %s\n\n", page.Title)

		// コンテンツを整形して書き込み
		content := cleanupContent(page.Content)
		fmt.Fprintln(file, content)

		// ページの区切り
		if i < len(pages)-1 {
			fmt.Fprintln(file, "\n"+strings.Repeat("-", 80)+"\n")
		}
	}

	return nil
}

// cleanupContent はコンテンツを整形する
func cleanupContent(content string) string {
	// 改行を統一（Windowsの CRLF を LF に変換）
	content = strings.ReplaceAll(content, "\r\n", "\n")
	
	// 連続する空白行を削除（3行以上の空行を2行に）
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	
	// 先頭と末尾の空白を削除
	content = strings.TrimSpace(content)
	
	// 行末の空白を削除
	regexTrailingSpaces := regexp.MustCompile(`[ \t]+\n`)
	content = regexTrailingSpaces.ReplaceAllString(content, "\n")
	
	// 連続するスペースを1つに
	regexMultipleSpaces := regexp.MustCompile(`[ \t]{2,}`)
	content = regexMultipleSpaces.ReplaceAllString(content, " ")
	
	// 見出し周りの空行を整理（見出し前に空行を挿入し、見出し後の空行を削除）
	// 文字列を一括で置換するのではなく、必要な調整を行う
	lines := strings.Split(content, "\n")
	for i := 0; i < len(lines); i++ {
		if i > 0 && strings.HasPrefix(strings.TrimSpace(lines[i]), "#") {
			// 見出しの前に空行がなければ追加（すでに前が空行の場合は追加しない）
			if lines[i-1] != "" {
				lines = append(lines[:i], append([]string{""}, lines[i:]...)...)
				i++ // 挿入された行をスキップ
			}
		}
	}
	
	return strings.Join(lines, "\n")
}