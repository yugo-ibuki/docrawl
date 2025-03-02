package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/yugo-ibuki/docrawl/internal/crawler"
	"github.com/yugo-ibuki/docrawl/internal/pdf"
)

var (
	baseURL      string
	outputPath   string
	maxDepth     int
	timeout      int
	delaySeconds float64 // クローリング間の遅延（秒）
	outputFormat string  // 出力形式（txtまたはpdf）
)

var rootCmd = &cobra.Command{
	Use:   "docrawl",
	Short: "ドキュメントサイトをクローリングしてテキストまたはPDFに変換するツール",
	Long: `docrawlはドキュメントサイト全体をクローリングし、
内容をテキストファイルまたはPDFとして保存するCLIツールです。技術のライブラリのような
ドキュメントサイトを対象としています。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if baseURL == "" {
			return fmt.Errorf("ベースURLを指定してください")
		}

		// クローラーを初期化
		crawler := crawler.New(baseURL, maxDepth, timeout, delaySeconds)
		pages, err := crawler.Crawl()
		if err != nil {
			return err
		}

		// 出力パスの調整
		if outputFormat == "txt" && !containsExtension(outputPath, ".txt") {
			if containsExtension(outputPath, ".pdf") {
				// PDFの拡張子をTXTに変更
				outputPath = outputPath[:len(outputPath)-4] + ".txt"
			} else {
				// 拡張子がない場合はTXTを追加
				outputPath = outputPath + ".txt"
			}
		}

		// テキスト形式で出力する場合
		if outputFormat == "txt" {
			// テキストファイルを直接生成
			err = crawler.GenerateTXT(pages, outputPath)
			if err != nil {
				return err
			}
			fmt.Printf("成功: %s にテキストファイルが生成されました\n", outputPath)
			return nil
		}

		// PDF（またはテキスト）ジェネレーターを使用
		generator := pdf.NewGenerator(outputPath)
		if err := generator.GeneratePDF(pages); err != nil {
			return err
		}

		return nil
	},
}

// containsExtension はパスに特定の拡張子が含まれているかを確認
func containsExtension(path, ext string) bool {
	return len(path) >= len(ext) && path[len(path)-len(ext):] == ext
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVarP(&baseURL, "url", "u", "", "クローリング開始URLを指定 (必須)")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "output.pdf", "出力ファイルパス")
	rootCmd.Flags().IntVarP(&maxDepth, "depth", "d", 3, "クローリングの最大深度")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "リクエストタイムアウト（秒）")
	rootCmd.Flags().Float64VarP(&delaySeconds, "delay", "w", 2.0, "リクエスト間の待機時間（秒）")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "txt", "出力形式 (txt または pdf)")

	rootCmd.MarkFlagRequired("url")
}
