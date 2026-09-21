# 設計概要

## 目的

eml-to-jsonl は RFC 2822 形式の EML ファイルをパースし、構造化された JSONL を stdout に出力します。
lite-llm などのダウンストリームツールへ入力するための Unix フィルターとして設計されています。

## 入力処理

| 入力ソース | 方法 |
|-----------|------|
| stdin | 引数なしの場合のデフォルト |
| ファイル引数 | `eml-to-jsonl file.eml` |
| ディレクトリ引数 | `eml-to-jsonl dir/` — ディレクトリ直下の `*.eml` を一括処理（再帰なし） |
| 混在 | `eml-to-jsonl dir/ extra.eml` — 指定順に処理 |

個別ファイルのエラーは stderr に出力し、残りのファイルの処理を継続します。
1件でも失敗した場合、終了コードは 1 になります。

## 出力フォーマット

stdout にメッセージ1件につき1行の JSON オブジェクトを出力します。
`--pretty` フラグでインデント付き JSON に切り替えられます。
`json.Encoder.SetEscapeHTML(false)` を使用しており、本文中の HTML はエンティティエスケープされません。

## パースパイプライン

```
reader（ファイル / stdin）
  └─ net/mail.ReadMessage()       — RFC 2822 ヘッダーと本文の分割
       ├─ extractHeaders()        — RFC 2047 ヘッダーを UTF-8 にデコード
       │    └─ mime.WordDecoder   — golang.org/x/text によるカスタム CharsetReader
       └─ parseBody()             — MIME 本文の振り分け
            ├─ 単純パート         — decodeTransfer() → decodeToUTF8() → BodyPart
            └─ multipart/*        — 再帰的な parseMultipart()
                 ├─ テキストパート → BodyPart（text/plain を先頭、text/html を後）
                 └─ 添付ファイル  → Attachment（ファイル名・MIME タイプ・デコード後サイズ）
```

## 文字コード処理

すべてのテキストコンテンツは出力前に UTF-8 に変換されます。

**ヘッダー:** RFC 2047 エンコードワード（`=?charset?encoding?text?=`）は
`mime.WordDecoder` によってデコードされます。カスタム `CharsetReader` は
`golang.org/x/text/encoding/htmlindex` を使用し、ISO-2022-JP・Shift_JIS・EUC-JP を含む
IANA 登録済みのすべての文字コードに対応します。

**本文パート:** `Content-Type` の `charset` パラメーターから `htmlindex.Get()` でデコーダーを取得し、
`golang.org/x/text/transform` 経由で変換します。

**encoding フィールド:** 最初に検出された非 ASCII テキストパートの文字コードを大文字で記録します。
すべてのパートが UTF-8 または ASCII の場合は省略されます。

## 転送エンコーディング

| エンコーディング | 処理 |
|----------------|------|
| `base64` | 空白除去ラッパー付き `base64.NewDecoder` |
| `quoted-printable` | `mime/quotedprintable.NewReader` |
| `7bit` / `8bit` / `binary` / 未指定 | そのまま読み込み |

## 添付ファイルの判定

以下の条件を満たす MIME パートは添付ファイルとして扱われます:

- `Content-Disposition` が `attachment` で始まる、または
- `Content-Disposition` が `inline` で始まり `filename` パラメーターを持つ、または
- メディアタイプが `text/*` でない

バイナリコンテンツは出力に含めず、メタデータのみ（ファイル名・MIME タイプ・デコード後バイトサイズ）を収集します。

## 本文の順序

`multipart/alternative` の場合、パートはメッセージ内の出現順に出力されます。
EML 仕様では `text/plain` が `text/html` より先に配置されるため、
この順序は自然に保たれます。並び替えは行いません。
