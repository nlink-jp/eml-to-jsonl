# eml-to-jsonl

シェルパイプライン向け EML パーサー。
`.eml` ファイルを読み込み、構造化された JSONL（1メール1行）を stdout に出力します。
[lagent](https://github.com/nlink-jp/lagent) などのツールとパイプで組み合わせ、
メール分析パイプラインを構築することを想定して設計されています。

## 機能

- ヘッダーの抽出: `from`, `to`, `cc`, `bcc`, `reply_to`, `sender`, `return_path`, `subject`, `date`, `message_id`, `in_reply_to`, `references`, `x_mailer`, `received`
- アドレスのフィールドは出てきたものをすべて読む: 2 つ目の `From` も報告し（`from_all`）、`To` / `Cc` もすべて読む
- アドレスを解析してから表示名を復号する。符号化された表示名がアドレスに化けることはない
- 転送されたメール（`message/rfc822`、または別の型で添付されたメール）を `attachments[].message` に開く
- すべての本文を **UTF-8** にデコード。元のエンコードを `encoding` フィールドに記録（宣言された文字コード、HTML の `<meta charset>`、または宣言の無い日本語の推定）
- マルチパート対応: `text/plain` 優先、`text/html` も含めて出力
- 転送エンコーディングの対応: `base64`、`quoted-printable`、`7bit`、`8bit`。壊れたデータもデコードできた分は残す
- 日本語文字コードに対応: ISO-2022-JP、Shift_JIS、EUC-JP（その他 IANA 登録済み文字コードも対応）
- 添付ファイルのメタデータ（ファイル名、MIME タイプ、サイズ）をバイナリ埋め込みなしで出力
- 黙って捨てない: コロンの無いヘッダ行、閉じない境界、壊れた Base64、推定した文字コードは `warnings` に残し、メールの残りは読み続ける
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

各メッセージが1行の JSON として出力されます:

```json
{
  "source": "inbox/message.eml",
  "message_id": "<abc123@example.com>",
  "in_reply_to": "<xyz@example.com>",
  "references": "<xyz@example.com>",
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

メールにあるときだけ出るフィールド:

```json
{
  "from_all": ["ceo@example.org", "attacker@example.net"],
  "sender": ["attacker@example.net"],
  "reply_to": ["ヘルプデスク <help@example.net>"],
  "return_path": ["<bounce@example.org>"],
  "attachments": [
    {"filename": "original.eml", "mime_type": "message/rfc822", "size": 855,
     "message": {"from": "Support <support@example.net>", "subject": "...", "body": [], "attachments": []}}
  ],
  "warnings": [
    {"code": "unclosed_boundary", "detail": "a multipart ended without its closing delimiter; the parts read so far are kept", "part": "2"}
  ]
}
```

メールの読み取りは、攻撃者が作ったメールを前提に書かれたライブラリ
[mailparse](https://github.com/nlink-jp/mailparse) が行います。

**アドレスのフィールド**（`from`・`to`・`cc`・`bcc`・`reply_to`・`sender`・`return_path`）:
1 つのメールボックスは RFC 5322 の形（`表示名 <アドレス>` か、アドレスだけ）で書き、読み直すと同じ 1 件に
戻ります。表示名が文字・数字・atom の記号・1 つずつの空白のほかを含むとき — `<`、`@`、`,`、
`John Q. Public` のような `.`、全角空白（U+3000）— は引用符で囲みます:
`"Bank <security@bank.example>" <attacker@example.net>`。表示名が 2 つ目のアドレスに見えることはありません。
名前だけでアドレスの無いメールボックスは、引用した名前だけを書きます。
フィールドは出てきたものをすべて読みます。`To` が 2 つあれば、両方のメールボックスを 1 つのリストにします。
`from` は 1 つ目の `From` で、`from_all` はすべての `From` のすべてのメールボックスです（2 つ以上あるときだけ出ます）。
RFC 5322 として解析できないフィールドは寛容に読みます（`invalid_address` の警告つき）。
null の `Return-Path` は `"<>"` です。

**入れ子のメール:**
それ自体がメールである添付 — `message/rfc822`、または `application/octet-stream` や `.eml` の名前で送られ、
中身でメールと判別できたもの — は、解析したメールを同じ形（`source` は無し）で `message` に持ちます。
3 層まで開きます。

**warnings（警告）:**
解析が失敗せずに扱ったこと — コロンの無いヘッダ行、重複した `From`、補修した符号化語、閉じない境界、
不正な文字を含む Base64、推定した文字コード、上限で切ったもの — を、安定した `code`（mailparse のコード。
`warnings.go` に一覧）と、当てはまる MIME パート（`part`。メールのヘッダなら省略）とともに `warnings` に並べます。
報告することが無ければフィールドごと省きます。同じコードが 1 通で 100 件を超えたら、そこで打ち切ります（`warning_limit`）。

**入力の判別:**
Outlook の `.msg` と TNEF（`winmail.dat`）は、msg-to-jsonl を案内して拒否します。ヘッダのフィールドが 1 つも
無い入力も拒否します。メールが普通持つフィールドを持たない最小限のメール（cron の `From: root`、`To` と
`Subject` だけ）は、`not_detected_as_mail` の警告つきで読みます。ヘッダの前の UTF-8 の BOM は飛ばします
（`leading_bom`）。64 MiB を超えるメールは拒否します。

**本文の順序:**
1. `text/plain` が存在する場合は常に先頭
2. 次に `text/html`
3. HTML のみのメールは `text/html` が1件だけ出力される

ファイル名の無い `text/plain` と `text/html` のパートだけが本文です。ほかのパート — `text/calendar`、
`text/csv`、名前のあるテキストのパート — は添付です。

**encoding フィールド:**
UTF-8 でも ASCII でもない最初の本文パートの元の文字コードを記録（例: `ISO-2022-JP`）。
宣言された文字コード、HTML の `<meta charset>`、または推定した文字コード（`charset_guessed` の警告が付く）です。
UTF-8 または ASCII の場合は省略されます。

**attachments フィールド:**
メタデータのみ（ファイル名・MIME タイプ・デコード後のバイトサイズ）と、入れ子のメールなら `message`。
バイナリは埋め込みません。名前の無い添付のファイル名は `"attachment"` です。

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
