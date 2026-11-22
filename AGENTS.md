# yomiyose の開発手順

コード変更時は毎回 [Ponytail スキル](https://github.com/DietrichGebert/ponytail/blob/main/skills/ponytail/SKILL.md) を使って差分をレビューする。変更した処理の呼び出し元、公開APIへの影響、フルネームのマスク範囲を確認する。

GoコードやGoテストを変更したら、リポジトリのルートで次を実行する。golangci-lint は v2 を使う。

```sh
go fmt ./...
gofmt -l .
go vet ./...
golangci-lint run ./...
go test -race -count=1 ./...
```

`gofmt -l .` の出力が空で、lint とテストが通ることを確認してから変更を完了する。指摘を修正したら再実行する。

公開API・既定値の変更は [README.md](README.md)、マスク性能や評価方法の変更は [EVALUATION.md](EVALUATION.md)、使用データ・文献・利用条件の変更は [DATA_SOURCES.md](DATA_SOURCES.md) に反映する。
