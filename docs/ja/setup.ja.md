# セットアップガイド

## 前提条件

- Go 1.22 以降

LLM API は不要です。eml-to-jsonl はネットワーク接続を持たない純粋なパーサーです。

## インストール

```sh
git clone https://github.com/nlink-jp/eml-to-jsonl.git
cd eml-to-jsonl
make build
# dist/ を PATH に追加するか、dist/eml-to-jsonl を PATH 上のディレクトリにコピーしてください
```

## Git フックのインストール

```sh
make setup
```

`pre-commit`（vet + lint）と `pre-push`（フルチェック）フックをインストールします。

## クイックスタート

```sh
# 単一 EML ファイルのパース
eml-to-jsonl message.eml

# ディレクトリ内の全 EML ファイルをパース
eml-to-jsonl ~/Downloads/exported-mail/

# 整形出力で確認
eml-to-jsonl --pretty message.eml | head -40

# lite-llm へパイプして分析
eml-to-jsonl inbox/ | lite-llm -p "各メールの送信者と件名を一覧にしてください。"
```
