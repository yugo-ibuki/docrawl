package crawler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// Page はクロールされたページの情報を格納する構造体
type Page struct {
	URL     string
	Title   string
	Content string
	Depth   int
}

// Crawler はウェブサイトをクロールする構造体
type Crawler struct {
	baseURL  string
	maxDepth int
	timeout  int
	delay    time.Duration
}

// New は新しいCrawlerインスタンスを作成する
func New(baseURL string, maxDepth, timeout int, delaySeconds float64) *Crawler {
	return &Crawler{
		baseURL:  baseURL,
		maxDepth: maxDepth,
		timeout:  timeout,
		delay:    time.Duration(delaySeconds * float64(time.Second)),
	}
}

// Crawl はベースURLからクローリングを開始し、見つかったページをすべて返す
func (c *Crawler) Crawl() ([]Page, error) {
	fmt.Printf("ページをクロール中: %s\n", c.baseURL)

	// シンプルなHTTPクライアントを作成
	client := &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}

	// リクエストの設定
	req, err := http.NewRequest("GET", c.baseURL, nil)
	if err != nil {
		return nil, err
	}

	// ブラウザのUser-Agentを設定
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// リクエストを送信
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込む
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// HTMLを解析
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	// タイトルを取得
	title := doc.Find("title").Text()
	fmt.Printf("タイトル: %s\n", title)

	// HTMLをプレーンテキストに変換
	textContent := extractText(doc)

	// 結果を表示
	fmt.Printf("テキストコンテンツサイズ: %d bytes\n", len(textContent))

	// 1ページのみを返す
	return []Page{
		{
			URL:     c.baseURL,
			Title:   title,
			Content: textContent,
			Depth:   0,
		},
	}, nil
}

// extractText はHTMLドキュメントからプレーンテキストを抽出する
func extractText(doc *goquery.Document) string {
	var sb strings.Builder

	// タイトルを抽出
	title := doc.Find("title").Text()
	sb.WriteString("# " + title + "\n\n")

	// ヘッダーを抽出して見出しとして追加
	doc.Find("h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			// ヘッダーレベルを取得
			nodeName := goquery.NodeName(s)
			level := int(nodeName[1] - '0')
			prefix := strings.Repeat("#", level) + " "

			sb.WriteString("\n" + prefix + text + "\n")
		}
	})

	// 段落を抽出
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			sb.WriteString("\n" + text + "\n")
		}
	})

	// リストを抽出
	doc.Find("ul, ol").Each(func(i int, s *goquery.Selection) {
		sb.WriteString("\n")
		s.Find("li").Each(func(j int, li *goquery.Selection) {
			text := strings.TrimSpace(li.Text())
			if text != "" {
				sb.WriteString("* " + text + "\n")
			}
		})
	})

	// テーブルを抽出
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		sb.WriteString("\n[テーブル]\n")
		s.Find("tr").Each(func(j int, tr *goquery.Selection) {
			var cells []string
			tr.Find("th, td").Each(func(k int, cell *goquery.Selection) {
				text := strings.TrimSpace(cell.Text())
				cells = append(cells, text)
			})
			sb.WriteString(strings.Join(cells, " | ") + "\n")
		})
	})

	// コードブロックを抽出
	doc.Find("pre, code").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			sb.WriteString("\n```\n" + text + "\n```\n")
		}
	})

	return sb.String()
}

// GenerateTXT はクロールしたページからTXTファイルを生成する
func (c *Crawler) GenerateTXT(pages []Page, outputPath string) error {
	if len(pages) == 0 {
		return fmt.Errorf("生成するページがありません")
	}

	// 出力ディレクトリを作成
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("出力ディレクトリの作成に失敗しました: %w", err)
	}

	// 拡張子が.txtでない場合は変更
	if !strings.HasSuffix(strings.ToLower(outputPath), ".txt") {
		outputPath = outputPath + ".txt"
	}

	// テキストファイルを作成
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("テキストファイルの作成に失敗しました: %w", err)
	}
	defer file.Close()

	// ヘッダー情報を書き込み
	fmt.Fprintf(file, "# ドキュメント: %s\n", pages[0].Title)
	fmt.Fprintf(file, "# URL: %s\n", pages[0].URL)
	fmt.Fprintf(file, "# 取得日時: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// コンテンツを書き込み
	fmt.Fprintln(file, pages[0].Content)

	// 成功メッセージを表示
	fmt.Printf("テキストファイルが生成されました: %s\n", outputPath)
	return nil
}

// nodeToText はHTMLノードをテキストに変換するヘルパー関数
func nodeToText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var buf strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		buf.WriteString(nodeToText(c))
	}

	return buf.String()
}
