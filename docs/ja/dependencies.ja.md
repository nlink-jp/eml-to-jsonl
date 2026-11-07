# 依存関係

## 実行時

| パッケージ | バージョン | 目的 | 内製しない理由 |
|---------|---------|---------|-----------------|
| `github.com/nlink-jp/mailparse` | v0.2.0 | メールを読む: RFC 5322 のヘッダーとアドレス、MIME、転送エンコーディング、文字コード、添付されたメール | 内製である。組織のメールパーサーで、msg-to-jsonl・mail-analyzer・mail-analyzer-local と共有し、同じ読み方をする。 |
| `golang.org/x/text` | （mailparse 経由） | 文字コードの変換（ISO-2022-JP、Shift_JIS、EUC-JP、Windows のコードページ） | Go チームが保守する、Go の定番の文字コードライブラリ。 |
| `golang.org/x/net` | （mailparse 経由） | `<meta charset>` からの文字コードの判定（`html`） | Go チームが保守する。 |

コミュニティ製のパッケージは使わない。

## 使用している標準ライブラリパッケージ

| パッケージ | 目的 |
|---------|---------|
| `encoding/json` | JSONL の出力 |
| `path/filepath` | ディレクトリのグロブ |

## 開発

| ツール | 目的 |
|------|---------|
| `golangci-lint` | 静的解析 |
| `govulncheck` | 依存関係の脆弱性スキャン |
