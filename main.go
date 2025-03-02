package main

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
	delaySeconds float64
)

var rootCmd = &cobra.Command{
	Use:   "document-crawler",
	Short: "ドキュメントサイトをクローリングしてPDFに変換するツール",
	Long: `document-crawlerはドキュメントサイト全体をクローリングし、
内容をPDFとして保存するCLIツールです。技術のライブラリのような
ドキュメントサイトを対象としています。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if baseURL == "" {
			return fmt.Errorf("ベースURLを指定してください")
		}

		crawler := crawler.New(baseURL, maxDepth, timeout, delaySeconds)
		pages, err := crawler.Crawl()
		if err != nil {
			return err
		}

		generator := pdf.NewGenerator(outputPath)
		if err := generator.GeneratePDF(pages); err != nil {
			return err
		}

		fmt.Printf("成功: %s にPDFが生成されました\n", outputPath)
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVarP(&baseURL, "url", "u", "", "クローリング開始URLを指定 (必須)")
	rootCmd.Flags().StringVarP(&outputPath, "output", "o", "output.pdf", "出力PDFファイルパス")
	rootCmd.Flags().IntVarP(&maxDepth, "depth", "d", 3, "クローリングの最大深度")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "リクエストタイムアウト（秒）")

	rootCmd.MarkFlagRequired("url")
}
