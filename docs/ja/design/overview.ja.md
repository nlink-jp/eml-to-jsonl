# 設計概要

## 目的

eml-to-jsonl は RFC 5322（.eml）のメールを読み、構造化した JSONL を stdout に出力する。
lagent などの下流の分析ツールへつなぐ Unix フィルタとして作られている。Outlook の .msg には msg-to-jsonl が同じ形を出力する。

## 入力処理

| 入力元 | 方法 |
|--------|-----|
| stdin | 引数が無いときの既定 |
| ファイル引数 | `eml-to-jsonl file.eml` |
| ディレクトリ引数 | `eml-to-jsonl dir/` — ディレクトリ直下の `*.eml` をグロブ（再帰しない） |
| 混在 | `eml-to-jsonl dir/ extra.eml` — 順に処理 |

個々のファイルのエラーは stderr に出し、残りのファイルの処理を続ける。1 つでも失敗すれば終了コードは 1。

## 出力フォーマット

1 メールにつき 1 つの JSON オブジェクトを stdout に出す。`--pretty` で字下げした複数行の JSON になる。
本文の HTML が実体参照にならないよう `json.Encoder.SetEscapeHTML(false)` を使う。

## 解析

解析はすべて [mailparse](https://github.com/nlink-jp/mailparse) が行う。このツールは自前の解析を持たない。

```
reader（ファイル / stdin）
  └─ 64 MiB + 1 バイトまで読む             — mailparse.DefaultMaxSize。超えれば ErrTooLarge
       ├─ mailparse.Detect()              — .msg / TNEF は msg-to-jsonl を案内して断る
       └─ mailparse.ParseBytes(AssumeEML) — ヘッダー、アドレス、MIME、文字コード、入れ子のメール
            └─ parser.FromMessage()       — 下の共通の写し方
```

呼び出し側が「メールだ」と言っている入力なので `AssumeEML` を立てる。形式の判別がメールと認めない最小限のメール
（Subject だけの cron のメールなど）も、断らずに `not_detected_as_mail` の警告つきで読む。

## 写し方（eml-to-jsonl と msg-to-jsonl で同一）

`internal/parser/email.go` が `mailparse.Message` を JSONL の 1 行に写す。このファイルは 2 つのツールでバイト単位で
同じなので、2 つは同じ形を出力する。違うのは入口（`parse.go`）だけ。

| フィールド | mailparse から |
|---|---|
| `from`・`sender` | すべての出現のすべての mailbox を `Mailbox.String()` にして `, ` で連ねる |
| `to`・`cc`・`bcc`・`reply_to` | すべての出現のすべての mailbox を `Mailbox.String()` で 1 要素ずつ |
| `return_path` | `Mailbox.String()`。空の逆経路は `<>` |
| `date` | `Date` を RFC 3339 で。無ければ `DateRaw` |
| `subject`・`message_id`・`in_reply_to`・`references` | mailparse の値のまま |
| `x_mailer` | 1 つ目の X-Mailer の `Field.Collapsed()` |
| `received` | すべての Received の `Field.Collapsed()`、順のまま |
| `encoding` | UTF-8・US-ASCII 以外の最初の本文の文字コードを大文字で。.msg の Unicode プロパティ（UTF-16LE）は入れ物の保存形式なので数えない |
| `body` | そのメールの本文。圧縮 RTF から復元した本文（`Origin == "rtf"`）は、他の本文が無いときだけ（.msg の RTF はふつう他の本文の写し） |
| `attachments` | すべての添付の `Filename`・`MediaType`・`Size`。添付されたメールは同じ写し方で `message` に（`source` は無し） |
| `warnings` | そのメール自身の `Warnings`（入れ子のメールは自分の分を持つ）。常に配列 |

アドレスを文字にするのは `Mailbox.String()` だけ。atom だけでない表示名を引用符で囲むので、復号するとアドレスの
形になる表示名も表示名のまま。ここでアドレスを整形するコードは持たない。
