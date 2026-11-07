# 設計概要

## 目的

eml-to-jsonl は RFC 2822 形式の EML ファイルをパースし、構造化された JSONL を stdout に出力します。
lagent などのダウンストリームツールへ入力するための Unix フィルターとして設計されています。

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
  └─ parser.Parse()
       ├─ mailparse.Detect()      — msg / TNEF / OLE は拒否（msg-to-jsonl を案内）
       ├─ mailparse.ParseBytes()  — Options{AssumeEML: true}: Detect が判別しない最小限のメールも
       │                            not_detected_as_mail を付けて読む。順序と重複を保ったヘッダ、表示名の復号より先のアドレスの解析、
       │                            MIME の木、転送エンコーディング、文字コード、入れ子のメール、
       │                            警告（上限は 1 回の解析全体で共有）
       └─ fromMessage()           — mailparse.Message を JSONL のレコードに写す（入れ子は再帰）
```

解析の欠陥は mailparse で直し、ここでは直しません。このパッケージが決めるのは JSONL の形だけです:

| JSONL | mailparse の |
|-------|--------------|
| `from` | 1 つ目の `From` のメールボックス |
| `from_all` | すべての `From` のすべてのメールボックス（2 つ以上のとき） |
| `to`・`cc`・`bcc`・`sender`・`reply_to`・`return_path` | 出てきたすべてのフィールドのすべてのメールボックス |
| `subject` | `Message.Subject`（1 つ目。RFC 2047 を復号・補修済み） |
| `date` | `Message.Date` を RFC 3339 で。解析できなければ `DateRaw` |
| `message_id`・`in_reply_to`・`references`・`x_mailer`・`received` | `Field.Collapsed()`（空白の連なりを 1 つに。net/textproto と同じ） |
| `body` | `Message.Bodies`（ファイル名の無い text/plain と text/html） |
| `encoding` | UTF-8 でも US-ASCII でもない最初の本文の文字コード（大文字） |
| `attachments` | `Message.Attachments`。`message` は `Attachment.Message` から |
| `warnings` | `Message.Warnings` |

メールボックスは `Mailbox.String()` です。読み直すと同じ 1 件に戻る RFC 5322 の形で、表示名が atom の記号と
1 つずつの空白のほかを含むときは引用符で囲み、復号した表示名が 2 つ目のアドレスに見えないようにします。

## 文字コード処理

すべてのテキストは mailparse が UTF-8 に変換します。宣言された文字コード、宣言が無ければ HTML の
`<meta charset>`、宣言の無い日本語なら推定です。推定や代替のたびに警告が付きます。

## 上限

上限はすべて mailparse の既定です: 64 MiB を超えるメールは拒否（25 MB の添付は Base64 で約 34 MB）、入れ子のメール 3 層、MIME の深さ 32、パート 1,000、
デコード後 120 MiB（1 回の解析全体で）。

## 本文の順序

`multipart/alternative` の場合、パートはメッセージ内の出現順に出力されます。
EML 仕様では `text/plain` が `text/html` より先に配置されるため、
この順序は自然に保たれます。並び替えは行いません。
