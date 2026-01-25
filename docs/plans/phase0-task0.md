# Phase 0 Task 0: スキル・エージェント動作確認

## 概要

dev-flowスキル動作確認のためのダミータスク。
設計者エージェント→実装者エージェント→レビュアーの一連のフローを検証する。

## 実装内容

`internal/dummy/hello.go` に `Hello()` 関数を実装する。

```go
package dummy

// Hello returns a greeting message.
func Hello() string {
    return "Hello, World!"
}
```

## テストケース

| # | テスト内容 | 期待結果 |
|---|-----------|---------|
| 1 | `Hello()` を呼び出す | `"Hello, World!"` が返される |

## ファイル構成

```
internal/
  dummy/
    hello.go       # Hello関数の実装
    hello_test.go  # テストコード
```

## 実装手順

1. `internal/dummy/` ディレクトリを作成
2. `hello_test.go` を作成（TDD: テストファースト）
3. `hello.go` を作成し、テストをパスさせる
4. `go test ./internal/dummy/...` で動作確認

## 備考

このタスクはdev-flowスキルの動作確認用であり、実際のプロダクト機能には含まれない。
