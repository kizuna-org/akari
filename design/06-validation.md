# 06. 基礎設計ブランチの検証

2026-10-10に以下を確認しました。

| 確認 | 結果 |
|---|---|
| `make lint` | golangci-lint default: all、0 issues |
| `make test` | race detector付きで成功 |
| `make coverage` | internal/host のstatement coverage 100% |
| `make build` | Go 1.26.2、CGOなしで成功 |
| `make generate` | 生成による変更なし |
| `make smoke` | 不正アドレス・使用中ポートを拒否、health応答、readinessの誤表示なし、SIGTERMで正常終了 |
| `make docker-check` | Composeの構成検証成功 |
| `docker compose build` | multi-stage imageの実ビルド成功 |
| `docker compose up -d --wait` | healthcheck成功 |
| コンテナの実行条件 | UID 1000、/appは読み取り専用、/dataは書き込み可能 |
| コンテナの不正設定 | 非ゼロ終了を確認 |
| `docker compose down` | 正常に停止・コンテナ回収。既存DB volumeの削除なし |

内部パッケージのカバレッジと、薄いmainの実バイナリ検証を分けています。
GoLandのRun構成はXMLとして検査していますが、GoLand GUIでの操作確認は行っていません。
ローカルのコンテナhealthcheckはlocalhostへの確認なので、wgetのproxy利用を明示的にoffにしています。
外部の接続でproxyやTLS検証を無効にする設定は追加していません。

この結果はホスト基盤の確認です。
並列Channel、LLM、MCP、内面の振る舞い、永続化、管理画面は後続実装です。
それらの検証条件は [05](./05-delivery.md) に記載しています。
