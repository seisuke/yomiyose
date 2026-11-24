# yomiyose の開発手順

コード変更ごとに [Ponytail スキル](https://github.com/DietrichGebert/ponytail/blob/main/skills/ponytail/SKILL.md) を使って差分をレビューする。変更した処理の呼び出し元、公開 API への影響、フルネームのマスク範囲を確認する。

Go のコードやテストを変更したら、リポジトリのルートで次を実行する。golangci-lint は v2 を使う。

```sh
go fmt ./...
gofmt -l .
go vet ./...
golangci-lint run ./...
go test -race -count=1 ./...
```

`gofmt -l .` の出力が空で、lint とテストが通ることを確認する。修正後は上のコマンドをすべて再実行する。

変更に応じて、次の資料も更新する。

- 公開 API・既定値：[README.md](README.md)
- マスク性能・評価方法：[EVALUATION.md](EVALUATION.md)
- 使用データ・文献・利用条件：[DATA_SOURCES.md](DATA_SOURCES.md)
