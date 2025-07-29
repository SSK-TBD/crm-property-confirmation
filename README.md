# ITANDI BB スクレーパー

Go言語で実装されたITANDI BBの物件情報を取得するスクレーパーです。

## 機能

- ITANDI BBへの自動ログイン
- 物件名による検索
- 物件詳細情報の取得
- 各ステップでのスクリーンショット保存
- エラーハンドリングとリトライ機能

## 必要な環境

- Go 1.24.5以上
- Chromium（自動的にインストールされます）

## インストール

```bash
# 依存関係のインストール
go mod download

# Chromiumのインストール（macOS）
brew install chromium
```

## セットアップ

### 認証情報の設定

認証情報を環境変数で設定してください：

```bash
export ITANDI_EMAIL="your-email@example.com"
export ITANDI_PASSWORD="your-password"
```

または `.env` ファイルを作成（`.env.example` を参考）：

```bash
cp .env.example .env
# .env ファイルを編集して認証情報を設定
```

## 使い方

### 基本的な使い方

```bash
# 物件名を指定して検索
go run . -property "クレールメゾン遠里小野"

# ヘッドレスモードで実行
go run . -property "物件名" -headless
```

### コマンドラインオプション

- `-property`: 検索する物件名（省略時は「サンプル物件」）
- `-headless`: ヘッドレスモードで実行（ブラウザを表示しない）

## 実行例

```bash
# 環境変数を設定して実行
export ITANDI_EMAIL="info@clair-tachikawa.com"
export ITANDI_PASSWORD="clair123"
go run . -property "クレールメゾン遠里小野"

# .envファイルを使用
source .env && go run . -property "クレールメゾン遠里小野"
```

## 出力

プログラムは以下のファイルを生成します：

1. **JSON出力**: `property_details_YYYYMMDD_HHMMSS.json` - 物件詳細情報
2. **DOM出力**: `property_card_dom_YYYYMMDD_HHMMSS.html` - 物件カードのDOM（モバイル表示）
3. **スクリーンショット**:
   - `step1_login_page.png`: ログインページ
   - `step2_after_login.png`: ログイン後の画面
   - `step3_search_results.png`: 検索結果画面
   - `step4_property_details.png`: 物件詳細画面

## 注意事項

- **重要**: 認証情報は絶対にコミットしないでください
- `.env` ファイルは `.gitignore` に追加されています
- ITANDI BBのUIが変更された場合、セレクタの調整が必要になる可能性があります
- 初回実行時はChromiumのダウンロードに時間がかかる場合があります

## トラブルシューティング

### Chromiumが起動しない場合

```bash
# 権限の問題を解決
xattr -d com.apple.quarantine /Applications/Chromium.app
```

### ログインに失敗する場合

- ネットワーク接続を確認してください
- ITANDI BBのサイトがアクセス可能か確認してください
- ログイン情報が正しいか確認してください

## 開発

### プロジェクト構造

```
.
├── main.go                    # メインプログラム
├── itandi_scraper.go          # 従来版スクレーパー
├── itandi_scraper_updated.go  # 実際の構造対応版スクレーパー
├── analyze.go                 # HTML構造分析ツール
├── main_updated.go            # 更新版実行ロジック
├── go.mod                     # Go モジュール定義
└── go.sum                     # 依存関係のチェックサム
```

### セレクタのカスタマイズ

物件情報の取得に使用するセレクタは `itandi_scraper.go` の `GetPropertyDetails` メソッドで定義されています。ITANDI BBのHTML構造に合わせて調整してください。

```go
selectors := map[string]string{
    "property_name": `h1, h2, .property-name, .building-name`,
    "address":       `.address, .location, .property-address`,
    "price":         `.price, .rent, .property-price`,
    "area":          `.area, .floor-area, .property-area`,
    "layout":        `.layout, .floor-plan, .property-layout`,
}
```