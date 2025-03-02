package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
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
	baseURL     string
	maxDepth    int
	timeout     int
	delay       time.Duration
	totalTime   time.Duration // 総実行時間
	visitedURLs map[string]bool
	mu          sync.Mutex // 並行アクセスのための排他制御
}

// New は新しいCrawlerインスタンスを作成する
func New(baseURL string, maxDepth, timeout int, delaySeconds float64, totalTimeSeconds int) *Crawler {
	return &Crawler{
		baseURL:     baseURL,
		maxDepth:    maxDepth,
		timeout:     timeout,
		delay:       time.Duration(delaySeconds * float64(time.Second)),
		totalTime:   time.Duration(totalTimeSeconds) * time.Second,
		visitedURLs: make(map[string]bool),
	}
}

// Crawl はベースURLからクローリングを開始し、見つかったページをすべて返す
func (c *Crawler) Crawl() ([]Page, error) {
	var pages []Page
	var mu sync.Mutex // pagesの保護用ミューテックス
	
	// コンテキストを作成（総時間制限付き）
	ctx, cancel := context.WithTimeout(context.Background(), c.totalTime)
	defer cancel()

	// エラーチャネルを作成
	errChan := make(chan error, 1)
	done := make(chan bool, 1)

	// クローリングを別のゴルーチンで実行
	go func() {
		err := c.crawlRecursive(ctx, c.baseURL, 0, &pages, &mu)
		if err != nil && err != context.DeadlineExceeded {
			errChan <- err
		}
		done <- true
	}()

	// タイムアウトまたはクローリング完了を待つ
	select {
	case err := <-errChan:
		return pages, fmt.Errorf("クローリング中にエラーが発生: %w", err)
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("\n指定された時間が経過したため、クローリングを終了します")
		}
		<-done // クローリングの完了を待つ
		return pages, nil
	case <-done:
		return pages, nil
	}
}

// crawlRecursive は再帰的にページをクロールする
func (c *Crawler) crawlRecursive(ctx context.Context, url string, depth int, pages *[]Page, mu *sync.Mutex) error {
	// コンテキストのキャンセルをチェック
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 最大深度チェック
	if depth > c.maxDepth {
		return nil
	}

	// 既に訪問済みのURLはスキップ（スレッドセーフに）
	c.mu.Lock()
	if c.visitedURLs[url] {
		c.mu.Unlock()
		return nil
	}
	c.visitedURLs[url] = true
	c.mu.Unlock()

	fmt.Printf("ページをクロール中 (深度 %d): %s\n", depth, url)

	// 遅延を入れる
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.delay):
	}

	// シンプルなHTTPクライアントを作成
	client := &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}

	// リクエストの設定
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	// ブラウザのUser-Agentを設定
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// リクエストを送信
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込む
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// HTMLを解析
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return err
	}

	// タイトルを取得
	title := doc.Find("title").Text()
	fmt.Printf("タイトル: %s\n", title)

	// HTMLをプレーンテキストに変換
	textContent := extractText(doc)

	// 結果を表示
	fmt.Printf("テキストコンテンツサイズ: %d bytes\n", len(textContent))

	// ページを追加（スレッドセーフに）
	mu.Lock()
	*pages = append(*pages, Page{
		URL:     url,
		Title:   title,
		Content: textContent,
		Depth:   depth,
	})
	mu.Unlock()

	// 同じドメイン内のリンクを収集して再帰的にクロール
	baseURL, err := parseBaseURL(url)
	if err != nil {
		return err
	}

	var links []string
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			nextURL, err := resolveURL(baseURL, href)
			if err != nil {
				return
			}

			// 同じドメインのURLのみを処理
			if strings.HasPrefix(nextURL, baseURL) {
				links = append(links, nextURL)
			}
		}
	})

	// 各リンクを順番にクロール
	for _, link := range links {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := c.crawlRecursive(ctx, link, depth+1, pages, mu); err != nil {
				if err == context.DeadlineExceeded {
					return err
				}
				fmt.Printf("警告: %sのクロール中にエラーが発生: %v\n", link, err)
			}
		}
	}

	return nil
}

// parseBaseURL はURLからベースURLを抽出する
func parseBaseURL(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s://%s", u.Scheme, u.Host), nil
}

// resolveURL は相対URLを絶対URLに解決する
func resolveURL(base, ref string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return "", err
	}

	resolvedURL := baseURL.ResolveReference(refURL)
	return resolvedURL.String(), nil
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
	fmt.Fprintf(file, "# クロール結果\n")
	fmt.Fprintf(file, "# 開始URL: %s\n", c.baseURL)
	fmt.Fprintf(file, "# 取得日時: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "# 取得ページ数: %d\n\n", len(pages))

	// 各ページの内容を書き込み
	for i, page := range pages {
		fmt.Fprintf(file, "\n%s\n", strings.Repeat("=", 80))
		fmt.Fprintf(file, "# ページ %d/%d\n", i+1, len(pages))
		fmt.Fprintf(file, "# URL: %s\n", page.URL)
		fmt.Fprintf(file, "# 深度: %d\n", page.Depth)
		fmt.Fprintf(file, "%s\n", strings.Repeat("=", 80))
		fmt.Fprintln(file)

		// コンテンツをクリーンアップして書き込み
		cleanedContent := cleanupTextContent(page.Content)
		fmt.Fprintln(file, cleanedContent)
		fmt.Fprintln(file)
	}

	// 成功メッセージを表示
	fmt.Printf("テキストファイルが生成されました: %s\n", outputPath)
	return nil
}

// cleanupTextContent はテキストコンテンツを整形する
func cleanupTextContent(content string) string {
	// 改行を統一（Windowsの CRLF を LF に変換）
	content = strings.ReplaceAll(content, "\r\n", "\n")
	
	// 連続する空白行を削除
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	
	// 先頭と末尾の空白を削除
	content = strings.TrimSpace(content)
	
	// 行末の空白を削除
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	
	// 連続する空行を最大1行に制限
	var result []string
	prevEmpty := false
	
	for _, line := range lines {
		isEmpty := len(strings.TrimSpace(line)) == 0
		
		if isEmpty && prevEmpty {
			// 連続する空行をスキップ
			continue
		}
		
		result = append(result, line)
		prevEmpty = isEmpty
	}
	
	return strings.Join(result, "\n")
}
