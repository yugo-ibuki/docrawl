package parser

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ExtractMainContent はHTMLドキュメントからメインコンテンツを抽出する
func ExtractMainContent(doc *goquery.Document) string {
	// メインコンテンツの可能性が高い要素のセレクタ
	// これはサイトによって調整が必要
	selectors := []string{
		"main", "article", ".content", ".documentation", ".docs-content",
		"#content", "#main-content", ".main-content", ".article-content",
	}

	for _, selector := range selectors {
		selection := doc.Find(selector)
		if selection.Length() > 0 {
			// 見つかった最初の要素を使用
			html, err := selection.First().Html()
			if err == nil && len(html) > 0 {
				return cleanHTML(html)
			}
		}
	}

	// 特定のセレクタが見つからない場合は、bodyコンテンツを使用
	body := doc.Find("body")
	if body.Length() > 0 {
		html, err := body.Html()
		if err == nil {
			return cleanHTML(html)
		}
	}

	// 最終手段としてHTMLドキュメント全体を返す
	html, err := doc.Html()
	if err == nil {
		return cleanHTML(html)
	}

	return ""
}

// cleanHTML はHTMLをクリーンアップする
func cleanHTML(html string) string {
	// JavaScript部分を削除
	html = removeElement(html, "<script", "</script>")

	// CSSスタイルを削除
	html = removeElement(html, "<style", "</style>")

	// インラインスタイルを削除
	html = removeAttribute(html, "style=\"", "\"")

	// クラス属性を削除
	html = removeAttribute(html, "class=\"", "\"")

	// IDを削除
	html = removeAttribute(html, "id=\"", "\"")

	// データ属性を削除
	html = removeDataAttributes(html)

	// 連続する空白を単一の空白に置換
	html = strings.Join(strings.Fields(html), " ")

	return html
}

// removeElement はHTMLから特定の要素を削除する
func removeElement(html, startTag, endTag string) string {
	result := html
	for {
		startIdx := strings.Index(strings.ToLower(result), strings.ToLower(startTag))
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(strings.ToLower(result[startIdx:]), strings.ToLower(endTag))
		if endIdx == -1 {
			break
		}

		endIdx += startIdx + len(endTag)
		if endIdx <= len(result) {
			result = result[:startIdx] + result[endIdx:]
		} else {
			break
		}
	}
	return result
}

// removeAttribute はHTML要素から特定の属性を削除する
func removeAttribute(html, attrStart, attrEnd string) string {
	result := html
	for {
		startIdx := strings.Index(strings.ToLower(result), strings.ToLower(attrStart))
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(result[startIdx:], attrEnd)
		if endIdx == -1 {
			break
		}

		endIdx += startIdx + len(attrEnd)
		if endIdx <= len(result) {
			result = result[:startIdx] + result[endIdx:]
		} else {
			break
		}
	}
	return result
}

// removeDataAttributes はHTML要素からdata-*属性を削除する
func removeDataAttributes(html string) string {
	result := html
	for {
		startIdx := strings.Index(strings.ToLower(result), "data-")
		if startIdx == -1 {
			break
		}

		// data-属性の前にスペースがあるか確認
		if startIdx > 0 && result[startIdx-1] != ' ' {
			// 実際のdata-属性でない場合は次の候補を探す
			result = result[startIdx+5:]
			continue
		}

		// 属性値の終わりを見つける
		endIdx := strings.Index(result[startIdx:], "\"")
		if endIdx == -1 {
			break
		}

		// 属性値を含む引用符の後の位置
		valueEndIdx := strings.Index(result[startIdx+endIdx+1:], "\"")
		if valueEndIdx == -1 {
			break
		}

		endIdx = startIdx + endIdx + valueEndIdx + 2
		if endIdx <= len(result) {
			result = result[:startIdx] + result[endIdx:]
		} else {
			break
		}
	}
	return result
}
