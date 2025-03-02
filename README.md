# docrawl

技術ドキュメントサイトをクローリングしてPDF化するCLIツール

## 概要

docrawlは技術ライブラリなどのドキュメントサイト全体をクローリングし、その内容をPDFとして保存するCLIツールです。指定したURLから始めて、同一ドメイン内のページを再帰的にクロールし、整形されたPDFドキュメントを生成します。

## インストール

```bash
go install github.com/yugo-ibuki/docrawl@latest
```

## 使い方

### 基本的な使い方

```bash
docrawl --url https://example.com/docs
```

### オプション

| オプション | 短縮形 | デフォルト値 | 説明 |
|------------|--------|--------------|------|
| `--url`    | `-u`   | (必須)       | クローリング開始URLを指定 |
| `--output` | `-o`   | `output.pdf` | 出力PDFファイルパス |
| `--depth`  | `-d`   | `3`          | クローリングの最大深度 |
| `--timeout`| `-t`   | `30`         | リクエストタイムアウト（秒） |

### 使用例

```bash
# 基本的な使用法
docrawl -u https://example.com/docs

# 出力ファイル名を指定
docrawl -u https://example.com/docs -o example-docs.pdf

# 最大深度を変更
docrawl -u https://example.com/docs -d 5

# タイムアウト時間を変更
docrawl -u https://example.com/docs -t 60

# すべてのオプションを指定
docrawl -u https://example.com/docs -o example-docs.pdf -d 5 -t 60
```

## 機能

- 指定されたURLからドキュメントサイト全体をクローリング
- 同一ドメイン内のリンクのみを追跡
- 最大クローリング深度の設定
- PDFドキュメントへの変換
- リクエストタイムアウトの設定
- 並行クローリングによる高速な処理

## 注意事項

- 対象サイトのロボット排除規約を尊重してください
- サイトへの過度な負荷を避けるため、適切なクローリング深度とタイムアウト設定を行ってください
- 生成されたPDFは個人的な使用のみを目的としてください
- 一部のウェブサイトではJavaScriptによるコンテンツレンダリングが行われるため、そのようなサイトでは適切にコンテンツが取得できない場合があります

## ライセンス

[MIT](LICENSE)