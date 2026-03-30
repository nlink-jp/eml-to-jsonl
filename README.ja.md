# eml-to-jsonl

シェルパイプライン向け EML パーサー。
`.eml` ファイルを読み込み、構造化された JSONL（1メール1行）を stdout に出力します。
[lite-llm](https://github.com/nlink-jp/lite-llm) などのツールとパイプで組み合わせ、
メール分析パイプラインを構築することを想定して設計されています。

## 機能

- ヘッダーの抽出: `from`, `to`, `cc`, `bcc`, `subject`, `date`, `message_id`, `in_reply_to`, `x_mailer`, `received`
- すべての本文を **UTF-8** にデコード。元のエンコードを `encoding` フィールドに記録
- マルチパート対応: `text/plain` 優先、`text/html` も含めて出力
- 転送エンコーディングの対応: `base64`、`quoted-printable`、`7bit`、`8bit`
- 日本語文字コードに対応: ISO-2022-JP、Shift_JIS、EUC-JP（その他 IANA 登録済み文字コードも対応）
- 添付ファイルのメタデータ（ファイル名、MIME タイプ、サイズ）をバイナリ埋め込みなしで出力
- 入力: stdin、ファイル引数、ディレクトリ（ディレクトリ指定時は `*.eml` を一括処理）

## インストール

```sh
git clone https://github.com/nlink-jp/eml-to-jsonl.git
cd eml-to-jsonl
make build
# dist/ を PATH に追加するか、dist/eml-to-jsonl を PATH 上のディレクトリにコピーしてください
```

## 使い方

```sh
# 単一ファイル
eml-to-jsonl message.eml

# 複数ファイル
eml-to-jsonl mail1.eml mail2.eml

# ディレクトリ（*.eml を一括処理）
eml-to-jsonl ~/exported-mail/

# 標準入力から
cat message.eml | eml-to-jsonl

# 整形出力（人間向け確認用）
eml-to-jsonl --pretty message.eml

# lite-llm へパイプしてメール分析
eml-to-jsonl inbox/ | lite-llm -p "各メールを1文で要約してください。"
```

## 出力フォーマット

各メッセージが1行の JSON として出力されます:

```json
{
  "source": "inbox/message.eml",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "from": "山田太郎 <yamada@example.co.jp>",
  "to": ["鈴木花子 <suzuki@example.co.jp>"],
  "cc": [],
  "bcc": [],
  "subject": "テスト",
  "date": "2026-03-27T10:00:00+09:00",
  "x_mailer": "Apple Mail",
  "received": [
    "from mx2.example.com by mx1.corp.com; Mon, 30 Mar 2026 09:00:00 +0900",
    "from sender.example.org by mx2.example.com; Mon, 30 Mar 2026 08:59:55 +0900"
  ],
  "encoding": "ISO-2022-JP",
  "body": [
    {"type": "text/plain", "content": "本文テキスト..."},
    {"type": "text/html",  "content": "<html>...</html>"}
  ],
  "attachments": [
    {"filename": "report.pdf", "mime_type": "application/pdf", "size": 102400}
  ]
}
```

**本文の順序:**
1. `text/plain` が存在する場合は常に先頭
2. 次に `text/html`
3. HTML のみのメールは `text/html` が1件だけ出力される

**encoding フィールド:**
主要テキストパートの元の文字コードを記録（例: `ISO-2022-JP`）。
UTF-8 または ASCII の場合は省略されます。

**attachments フィールド:**
メタデータのみ（ファイル名・MIME タイプ・デコード後のバイトサイズ）。バイナリは埋め込みません。

## フラグ

| フラグ | デフォルト | 説明 |
|--------|-----------|------|
| `-pretty` | false | JSONL ではなく整形 JSON で出力 |
| `-version` | — | バージョンを表示して終了 |

## ビルド

```sh
make build       # 現在のプラットフォーム向け
make build-all   # 全リリースプラットフォーム向け（dist/ に出力）
make test        # テスト実行
make check       # vet + lint + test + build + govulncheck
```

## ドキュメント

- [docs/ja/design/overview.md](docs/ja/design/overview.md) — 設計概要
- [docs/ja/setup.md](docs/ja/setup.md) — セットアップガイド
- [docs/dependencies.md](docs/dependencies.md) — 外部依存ライブラリ

## PST ファイルの処理

Outlook PST (Personal Storage Table) ファイルは
[readpst](https://www.five-ten-sg.com/libpst/) で EML に変換してから
eml-to-jsonl で処理できます:

```sh
# readpst のインストール
# macOS:         brew install libpst
# Debian/Ubuntu: apt install pst-utils

# PST → EML に変換
readpst -e -o /tmp/pst-output/ archive.pst

# 展開された EML を一括パース
eml-to-jsonl /tmp/pst-output/

# パイプライン: PST → EML → JSONL → LLM 分析
readpst -e -o /tmp/pst-output/ archive.pst && \
  eml-to-jsonl /tmp/pst-output/ | gem-cli --batch -s "各メールを要約して"
```

[msg-to-jsonl](https://github.com/nlink-jp/msg-to-jsonl) と組み合わせる場合は
`readpst -m` で MSG 形式で出力できます。

## util-series について

eml-to-jsonl は [util-series](https://github.com/nlink-jp/util-series) の一部です。
util-series はローカル・クラウド LLM と連携する軽量 CLI ツール群です。
