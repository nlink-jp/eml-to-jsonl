# セットアップガイド

## 前提条件

- Go 1.22 以降

LLM API は不要です。lite-eml はネットワーク接続を持たない純粋なパーサーです。

## インストール

```sh
git clone https://github.com/nlink-jp/lite-eml.git
cd lite-eml
make build
# bin/ を PATH に追加するか、bin/lite-eml を PATH 上のディレクトリにコピーしてください
```

## Git フックのインストール

```sh
make setup
```

`pre-commit`（vet + lint）と `pre-push`（フルチェック）フックをインストールします。

## クイックスタート

```sh
# 単一 EML ファイルのパース
lite-eml message.eml

# ディレクトリ内の全 EML ファイルをパース
lite-eml ~/Downloads/exported-mail/

# 整形出力で確認
lite-eml --pretty message.eml | head -40

# lite-llm へパイプして分析
lite-eml inbox/ | lite-llm -p "各メールの送信者と件名を一覧にしてください。"
```
