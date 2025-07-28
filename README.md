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

## 使い方

### 基本的な使い方

```bash
# メール・パスワードでのログイン（推奨）
go run . -property "物件名"

# ヘッドレスモードで実行
go run . -property "物件名" -headless

# 電話認証システム対応版（電話認証が必要）
go run . -updated

# メール・パスワードログインを明示的に使用
go run . -email-login

# HTML構造の分析モード
go run . -analyze

# ログインページ検索モード
go run . -find-login
```

### コマンドラインオプション

- `-property`: 検索する物件名（省略時は「サンプル物件」）
- `-headless`: ヘッドレスモードで実行（ブラウザを表示しない）
- `-email-login`: メール・パスワードログインを使用（デフォルトと同じ）
- `-updated`: 電話認証システム対応版スクレーパーを使用
- `-analyze`: HTML構造分析モードで実行（開発・デバッグ用）
- `-find-login`: ログインページ検索モード（開発・デバッグ用）

## 実行例

```bash
# 「クレール立川」という物件を検索（メール・パスワードログイン）
go run . -property "クレール立川"

# ヘッドレスモードで実行
go run . -property "クレール立川" -headless

# 電話認証が必要な場合
go run . -updated
```

## 出力

プログラムは以下の情報を出力します：

1. **コンソール出力**: 各ステップの進捗状況と取得した物件情報
2. **スクリーンショット**:
   - `step1_login_page.png`: ログインページ
   - `step2_after_login.png`: ログイン後の画面
   - `step3_search_results.png`: 検索結果画面
   - `step4_property_details.png`: 物件詳細画面
3. **JSON形式の物件情報**: 取得できた物件詳細情報

## 注意事項

- **重要**: ITANDI BBは電話認証が必要なシステムです。完全な自動化には限界があります
- ログイン情報はソースコードに含まれています（実際の運用では環境変数等を使用してください）
- ITANDI BBのUIが変更された場合、セレクタの調整が必要になる可能性があります
- 初回実行時はChromiumのダウンロードに時間がかかる場合があります
- `-updated` フラグを使用すると、実際のITANDI BB構造に対応したスクレーパーが実行されます

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