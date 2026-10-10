# Akari

感情・記憶・関心・目標を備え、人間らしく自律的に考える一人の存在を作ります。

- [振る舞いの仕様](./docs/README.md)
- [組み直した基礎設計](./design/README.md)
- [実装順と未決の項目](./design/05-delivery.md)

このブランチには基礎設計、最小のホスト、並列実行・共通の行為境界と内面状態の基礎実装が入っています。
[内面状態の接続](./design/07-inner-state.md) は疑似モデルを使って検証しています。
[保存・復元・outbox](./design/08-durable-state.md) はローカルファイルと疑似の行為で検証し、
プロセスの強制終了後も結果不明の行為を再送しない基盤を実装しています。
[ホスト起動・停止](./design/09-host-lifecycle.md)にも保存復元を接続しました。
**常設Channel、実LLM・実MCPはまだ接続していません。**
`/healthz` はホストの生存だけを示し、Akariが思考できることは示しません。

## Go / GoLand

Go 1.26.2を使います。リポジトリをGoLandで開き、共有Run構成の `Akari` を選びます。
Go SDKには1.26.2を設定します。
ルートの `go.work` で一つのGoモジュールを参照します。クラウド認証・DBは不要です。

```sh
cd akari
make init
make build
make run
```

別の端末で確認します。

```sh
curl http://127.0.0.1:8080/healthz
# {"status":"alive","mode":"foundation"}
```

終了はCtrl+Cで、採用済み状態を保存して終了します。起動時には前回の状態を復元します。
保存先は既定で作業ディレクトリの `data` で、`AKARI_DATA_DIR` で変更できます。
アドレスを変える場合は `AKARI_ADDR=127.0.0.1:8081 make run` とします。
`.env.example` は設定例です。ホストはdotenvを自動では読みません。

## Docker

リポジトリのルートで実行します。

```sh
docker compose up --build -d --wait
curl http://127.0.0.1:8080/healthz
docker compose down
```

公開先は既定でlocalhostです。ポートは `AKARI_PORT=8081 docker compose up --build -d --wait`
で変えられます。非rootで実行し、root filesystemは読み取り専用です。
`akari-data` volumeの`/data`に人格・内面・記憶・outboxを保存し、再起動時に復元します。
破損や人格不一致では起動を拒否し、空の状態に上書きしません。
既存DB volumeの削除・変換は行いません。旧環境からの移行は別途扱います。

## 検証

```sh
cd akari
make generate
make lint
make test
make coverage
make smoke
make docker-check
```

lintは既存のgolangci-lint（default: all）を継続し、初回はツールのダウンロードが必要です。
coverageはinternalパッケージを対象にし、薄いプロセス入口は実バイナリの起動・停止で検証します。
HTTPのホスト基盤・並列実行の契約と、後続で実装する人格の振る舞いの検証は分けて報告します。
