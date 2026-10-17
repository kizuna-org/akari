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
[管理コンソール](./design/10-console.md)にはdocsの関係図・検索・図の表示と、稼働状態の概要を用意しました。
[開発ワークスペース](./design/11-development-workspace.md)では、SQLiteのタスク管理、GitのMarkdown編集、
Akari MCPとREST APIを同じコンソールへ接続しています。
**常設Channel、実LLM、Akariが外部の道具を使うMCP接続と、開発タスクの実行ワーカーは未接続です。**
`/healthz` はホストの生存だけを示し、Akariが思考できることは示しません。

## Go / GoLand

Go 1.26.2を使います。リポジトリをGoLandで開き、共有Run構成の `Akari` を選びます。
Go SDKには1.26.2を設定します。
ルートの `go.work` で一つのGoモジュールを参照します。Gitコマンドが必要です。
クラウド認証やDBサーバーは不要で、タスク用SQLiteをローカルに作成します。

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

## 管理コンソール

起動後、[http://127.0.0.1:8082](http://127.0.0.1:8082)を開きます。
GoLand / `make run`では、`akari/data/admin.token`にある管理用コードでログインします。
`AKARI_DATA_DIR`を変更している場合は、そのディレクトリの`admin.token`を確認してください。
コードは起動ごとに更新します。開発タスクとMarkdownは編集でき、Akariの稼働状態は閲覧専用です。

- 設計マップ：docs / designの参照関係から本文へ移動。
- ドキュメント：全文検索、章の目次、関連資料、Mermaid図、Markdownの追加・編集とGit保存。
- 開発タスク：残作業・判断待ち・進捗・優先度・メモの管理。MCP/APIと同じDBを利用。
- 稼働状態：採用版・保存版・outboxの概要。

管理アドレスは`AKARI_ADMIN_ADDR`で変更します。通常のhealthポートとは分離しています。
`AKARI_ADMIN_AUTH_DISABLED=true`を設定すると、管理画面のアプリケーション認証を無効にします。
この設定では`admin.token`を作成せず、管理画面のデータAPIをトークンなしで利用できます。
管理アドレスを信頼できるネットワークや認証付きプロキシの内側に限定してください。
TLS終端プロキシを使う場合、同一オリジン検証とSecure Cookieには`X-Forwarded-Proto: https`を使います。
文書は初回だけ同梱データからGitリポジトリを作成し、以降はGitのHEADから読みます。
`AKARI_DOCS_REPO`で既存のworktreeも指定できます。画面の利用にNodeや外部CDNは不要です。
MCPの接続先は`http://127.0.0.1:8082/mcp`です。
認証・ツール一覧・REST APIは[接続手順](./design/11-development-workspace.md)を参照してください。

## Docker

リポジトリのルートで実行します。

```sh
docker compose up --build -d --wait
curl http://127.0.0.1:8080/healthz
docker compose down
```

管理画面は同じく[http://127.0.0.1:8082](http://127.0.0.1:8082)です。
起動中に`docker compose exec akari cat /data/admin.token`でコードを確認します。
管理側のポートは`AKARI_ADMIN_PORT`で変更できます。

公開先は既定でlocalhostです。ポートは `AKARI_PORT=8081 docker compose up --build -d --wait`
で変えられます。非rootで実行し、root filesystemは読み取り専用です。
`akari-data` volumeの`/data`に人格・内面・記憶・outboxを保存し、再起動時に復元します。
開発タスクの`development.db`と、Markdownの`documentation/.git`も同じvolumeへ保持します。
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

ブラウザ検証は`make build`後、`scripts/browser`で`npm ci`、
`npx playwright install chromium`、`npm test`を実行します。Nodeは検証時だけ必要です。
