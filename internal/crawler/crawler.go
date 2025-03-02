package crawler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/yugo-ibuki/docrawl/internal/parser"
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
	baseURL    string
	maxDepth   int
	timeout    int
	delay      time.Duration // クローリング間の遅延
	visited    map[string]bool
	visitedMux sync.Mutex
	pages      []Page
	pagesMux   sync.Mutex
	semaphore  chan struct{}
}

// New は新しいCrawlerインスタンスを作成する
func New(baseURL string, maxDepth, timeout int, delaySeconds float64) *Crawler {
	return &Crawler{
		baseURL:   baseURL,
		maxDepth:  maxDepth,
		timeout:   timeout,
		delay:     time.Duration(delaySeconds * float64(time.Second)),
		visited:   make(map[string]bool),
		pages:     []Page{},
		semaphore: make(chan struct{}, 5), // 同時に5つまでのリクエストを許可
	}
}

// Crawl はベースURLからクローリングを開始し、見つかったページをすべて返す
func (c *Crawler) Crawl() ([]Page, error) {
	baseURLParsed, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	// ベースURLのドメインを保存
	baseDomain := baseURLParsed.Hostname()

	// クローリング開始
	err = c.crawlPage(c.baseURL, 0, baseDomain)
	if err != nil {
		return nil, err
	}

	return c.pages, nil
}

// crawlPage は指定されたURLとその子URLをクロールする
func (c *Crawler) crawlPage(pageURL string, depth int, baseDomain string) error {
	// 最大深度をチェック
	if depth > c.maxDepth {
		return nil
	}

	// URLの正規化
	normalizedURL := normalizeURL(pageURL)

	// すでに訪れたページか確認
	c.visitedMux.Lock()
	if c.visited[normalizedURL] {
		c.visitedMux.Unlock()
		return nil
	}
	c.visited[normalizedURL] = true
	c.visitedMux.Unlock()

	// 同時実行数を制限
	c.semaphore <- struct{}{}
	defer func() { <-c.semaphore }()

	// HTTPクライアントの作成
	client := &http.Client{
		Timeout: time.Duration(c.timeout) * time.Second,
	}

	// HTTPリクエスト
	req, err := http.NewRequest("GET", normalizedURL, nil)
	if err != nil {
		return err
	}

	// 自然なブラウザリクエストのヘッダーを設定
	setRequestHeaders(req)

	// HTTPレスポンス
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ステータスコード %d: %s", resp.StatusCode, normalizedURL)
	}

	// goqueryでドキュメントを解析
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	// タイトルとコンテンツを抽出
	title := doc.Find("title").Text()
	content := parser.ExtractMainContent(doc)

	// ページを保存
	page := Page{
		URL:     normalizedURL,
		Title:   title,
		Content: content,
		Depth:   depth,
	}

	c.pagesMux.Lock()
	c.pages = append(c.pages, page)
	c.pagesMux.Unlock()

	fmt.Printf("クロール完了: %s (深度: %d)\n", normalizedURL, depth)

	// リンクを抽出して再帰的にクロール
	var wg sync.WaitGroup
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// 相対URLを絶対URLに変換
		absoluteURL, err := resolveURL(normalizedURL, href)
		if err != nil {
			return
		}

		// 同じドメイン内のリンクのみをクロール
		parsedURL, err := url.Parse(absoluteURL)
		if err != nil || parsedURL.Hostname() != baseDomain {
			return
		}

		// 外部リソースのリンクをスキップ
		if strings.HasSuffix(absoluteURL, ".pdf") || strings.HasSuffix(absoluteURL, ".zip") {
			return
		}

		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			_ = c.crawlPage(url, depth+1, baseDomain)
		}(absoluteURL)
	})

	wg.Wait()
	return nil
}

// normalizeURL はURLを正規化する
func normalizeURL(rawURL string) string {
	// フラグメントを削除
	if i := strings.Index(rawURL, "#"); i > 0 {
		rawURL = rawURL[:i]
	}
	return rawURL
}

// resolveURL は相対URLを絶対URLに変換する
func resolveURL(base, href string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	refURL, err := url.Parse(href)
	if err != nil {
		return "", err
	}

	resolvedURL := baseURL.ResolveReference(refURL)
	return resolvedURL.String(), nil
}

// setRequestHeaders はリクエストに一般的なブラウザのヘッダーを設定する
func setRequestHeaders(req *http.Request) {
	// 一般的なブラウザのユーザーエージェントを使用
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36")

	// 追加のヘッダーを設定して自然なブラウザリクエストに見せる
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")
}
