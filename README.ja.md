# eml-to-jsonl

シェルパイプライン向け EML パーサー。
`.eml` ファイルを読み込み、構造化された JSONL（1メール1行）を stdout に出力します。
[lagent](https://github.com/nlink-jp/lagent) などのツールとパイプで組み合わせ、
メール分析パイプラインを構築することを想定して設計されています。

## 機能

- メールは組織のメールパーサー [mailparse](https://github.com/nlink-jp/mailparse)（攻撃者が作ったメールを前提に
  書かれている）で読む。msg-to-jsonl と同じ形を出力する
- ヘッダー: `from`、`sender`、`reply_to`、`return_path`、`to`、`cc`、`bcc`、`subject`、`date`、`message_id`、
  `in_reply_to`、`references`、`x_mailer`、`received`。アドレスの欄はすべての出現を読み、mailbox は RFC 5322 の形で
  書く（符号化した表示名がアドレスに化けない）
- すべての本文を **UTF-8** にデコード。元の文字コードを `encoding` フィールドに記録
- 日本語文字コードに対応: ISO-2022-JP、Shift_JIS、EUC-JP（その他 `golang.org/x/text` が知る文字コード）
- 添付ファイルのメタデータ（ファイル名、MIME タイプ、サイズ）。転送されたメールは同じ形に解析する
- `warnings`: 解析が失敗せずに処理したこと（mailparse の安定したコード）
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

# lagent へパイプしてメール分析（入力全体を 1 つの添付として渡す。上限 256 KiB）
eml-to-jsonl inbox/ | lagent -p "各メールを1文で要約してください。"
```

## 出力フォーマット

各メッセージが1行の JSON として出力されます（eml-to-jsonl と msg-to-jsonl は同じ形）:

```json
{
  "source": "inbox/message.eml",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "references": "<first@example.com> <xyz@example.com>",
  "from": "山田太郎 <yamada@example.co.jp>",
  "sender": "list@example.org",
  "reply_to": ["\"営業, 東京\" <sales@example.co.jp>"],
  "return_path": "bounce@example.co.jp",
  "to": ["鈴木花子 <suzuki@example.co.jp>"],
  "subject": "テスト",
  "date": "2026-03-27T10:00:00+09:00",
  "x_mailer": "Apple Mail",
  "received": [
    "from mx2.example.com by mx1.example.net; Mon, 30 Mar 2026 09:00:00 +0900",
    "from sender.example.org by mx2.example.com; Mon, 30 Mar 2026 08:59:55 +0900"
  ],
  "encoding": "ISO-2022-JP",
  "body": [
    {"type": "text/plain", "content": "本文テキスト..."},
    {"type": "text/html",  "content": "<html>...</html>"}
  ],
  "attachments": [
    {"filename": "report.pdf", "mime_type": "application/pdf", "size": 102400},
    {"filename": "", "mime_type": "message/rfc822", "size": 2048,
     "message": {"message_id": "<fwd@example.org>", "subject": "元のメール", "body": [], "attachments": [], "warnings": []}}
  ],
  "warnings": [
    {"code": "duplicate_field", "detail": "From appears 2 times; every occurrence is kept"}
  ]
}
```

**アドレス:** 欄のすべての出現の、すべての mailbox。`from` と `sender` は 1 本の文字列（`, ` で連ねる）、`to`・`cc`・
`bcc`・`reply_to` は配列。表示名に atom の文字と空白以外が含まれれば引用符で囲むので、復号すると
`Bank <security@bank.example>` になる表示名も表示名のまま。`return_path` はバウンスなら `<>`。

**本文:** `text/plain` と `text/html` のパートを、メールの中の順に。その他のテキストのパート（`text/calendar`、
`text/csv`）は添付になる。

**encoding フィールド:** UTF-8・US-ASCII 以外の最初の本文の元の文字コード（例: `ISO-2022-JP`）。それ以外は省略。

**attachments フィールド:** メタデータのみ（ファイル名・MIME タイプ・デコード後のバイトサイズ）。バイナリは
埋め込まない。メールに名前が無ければファイル名は空。添付されたメール（`message/rfc822`、または中身でメールと
分かるもの）は `message` に同じ形（`source` は無し）で入る。3 層まで。

**warnings:** mailparse がこのメールで失敗せずに処理したこと（重複したヘッダー、壊れたアドレス、壊れた符号化、
上限など）を `{code, detail, part}` で。コードは安定している（[mailparse の README](https://github.com/nlink-jp/mailparse)）。
常に配列。`not_detected_as_mail` は、メールと認められなかったが読んだ最小限のメールに付く。

**受け付けない入力:** Outlook の `.msg` と `winmail.dat`（msg-to-jsonl を使う）、ヘッダーが 1 つも無い入力、
64 MiB を超えるメール。エラーは stderr に出し、他のファイルの処理は続ける。

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

- [docs/ja/design/overview.ja.md](docs/ja/design/overview.ja.md) — 設計概要
- [docs/ja/setup.ja.md](docs/ja/setup.ja.md) — セットアップガイド
- [docs/ja/dependencies.ja.md](docs/ja/dependencies.ja.md) — 外部依存ライブラリ

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

# パイプライン: PST → EML → JSONL → LLM 分析（1 通ごとに gem-agent を 1 回実行）
# （メールボックス全体は、1 回のパイプで渡せる 256 KiB を超える）
readpst -e -o /tmp/pst-output/ archive.pst && \
  eml-to-jsonl /tmp/pst-output/ | while IFS= read -r mail; do
    printf '%s\n' "$mail" | gem-agent -p "このメールを要約して"
  done
```

[msg-to-jsonl](https://github.com/nlink-jp/msg-to-jsonl) と組み合わせる場合は
`readpst -m` で MSG 形式で出力できます。

## util-series について

eml-to-jsonl は [util-series](https://github.com/nlink-jp/util-series) の一部です。
util-series はローカル・クラウド LLM と連携する軽量 CLI ツール群です。
