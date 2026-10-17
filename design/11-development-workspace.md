# 11. 開発タスク・Git文書・Akari MCP

Akari自体を開発するための作業を、管理コンソール・REST API・Akari MCPから管理します。
タスクはSQLite、ドキュメントはGitリポジトリ内のMarkdownが正本です。
Akari自身の目標・内面・outboxとは別の開発者用データです。

## 現在使えるもの

- 開発タスクの追加・編集・状態変更・削除、優先度・分類・検索・メモ・関連資料。
- 未着手 / 進行中 / 保留・ブロック / 判断待ち / 完了のボードと集計、JSON出力。
- コンソールとMCPが同じDBを読み書きし、再起動・プロセス強制終了後も保存済みの更新を復元。
- Markdownの閲覧・追加・編集。保存時にローカルGitコミットを作成し、以前の本文を履歴に保持。
- MCPのツール一覧取得と、タスク・文書を操作する8ツール。

初期タスクは確定仕様と未実装項目を整理した雛形で、DBの初期化時だけ登録します。
その後はコンソールやAPIの編集が正本になり、再起動・再ビルドで雛形に戻しません。
docsの文章からタスクや合意をAIが自動生成・確定する機能はありません。
判断待ちを完了にしても、確定仕様の変更や外部サービスの利用承認にはなりません。

## タスクの保存と競合

`AKARI_DATA_DIR/development.db`に保存します。追加のDBサーバーは不要です。
pure-Go SQLiteドライバーを使い、Go 1.26.2 / CGOなし / Dockerの非root実行を維持します。
DBは0600で作成し、schemaはSQLiteの`user_version`で管理します。
タスクごとの行と版、一覧の版と更新日時を一つのトランザクションで変更します。
不正な値、古い版、破損DB、未対応schemaを空の状態へ上書きしません。
schema番号が未設定でも既存のテーブルがあるDBは初期化を拒否し、編集・削除済みのデータを保持します。

MCP・APIはタスクごとの版を照合します。同じタスクへの同じ版の更新は一つだけ成功し、
別のタスクの更新は互いの版を無効にしません。更新に失敗した画面は下書きを保持します。
一覧を更新しただけで開いている下書きの版を最新にすり替えません。

一つのホストが既存のinstallation leaseを持ち、複数クライアントがそのAPIへ接続します。
DBのトランザクション中にLLMや外部コマンドの仕事を実行しません。
正常停止時は管理用HTTPの処理を終えてからinstallation leaseを解放します。
停止が期限切れになった場合もleaseを保持し、処理中の書き込みと別ホストの起動を重ねません。
この場合は処理とGit子プロセスの終了を待ってから最終保存し、期限超過を報告して終了します。
Akariはタスクの管理を担当し、タスクの状態変更で実行ワーカーを起動しません。

JSON出力は保存済みタスクの確認・持ち出し用です。JSONの一括インポートは未実装です。
完全なバックアップはホストを停止して、DBと文書Gitリポジトリを保存します。

## MarkdownのGitリポジトリ

既定では`AKARI_DATA_DIR/documentation`に専用のローカルGitリポジトリを初回だけ作成し、
バイナリに同梱した`docs/*.md`と`design/*.md`を初期コミットします。
以降はGitのHEADにコミットされたMarkdownを読みます。
コンソールの「資料を更新」で取得し直せるため、編集の反映に再ビルドは不要です。

既存リポジトリを使う場合は、`AKARI_DOCS_REPO`にそのworktreeのルートを指定します。
この場合は初期化や雛形のコピーをしません。保存対象は`docs/`または`design/`直下のMarkdownです。
絶対パス・親ディレクトリへの移動・Git内部・任意ファイルの読み書きは許可しません。
ユーザー指定の文書名をGitコマンドの引数に渡しません。既存名はGitのtreeから取得し、
ステージ対象は固定のdocs/designに限定します。コミット前に差分が指定された一文書だけであることも確認します。
Gitコマンドはシェルを介さず、コミット時のhooksと署名コマンドを無効にします。

編集は読み取り時のGit revisionとcleanなworktreeが必要です。
Gitの表示設定に関係なく未追跡ファイルも検査し、保存中はフォームへの追加入力を無効にします。
同時編集や別のコミットを検出したら拒否し、未コミットの作業を取り込みません。
Gitのignore対象であっても、同名のローカル下書きを新規文書で上書きしません。
APIによる更新は一つのホスト内で直列化します。
外部のGitコマンドによる書き込み・branch切り替えは、ホストを止めて行ってください。
Gitの保存に失敗した場合は最新HEADとworktreeを確認し、成功したと見なして再送しません。
失敗時は復旧に使える本文やindexを残すため、Gitの自動resetは行いません。
未コミットの変更が残った場合は以後の保存を拒否します。ホストを停止し、Gitのlog・status・diffを確認して、
必要な内容を退避・コミットしたうえでcleanなworktreeへ戻してください。
自動fetch・push・公開設定の変更は行いません。

Dockerの既定構成ではDBと文書Gitの両方が既存の`/data` volumeへ保存されます。
既存リポジトリをDockerから使う場合は、別途worktreeをbind mountし、
コンテナ内のパスを`AKARI_DOCS_REPO`へ指定します。Gitは実行イメージに同梱します。

## REST API

管理listenerの認証を共用します。通常listenerには公開しません。
Bearerは`admin.token`の値、ブラウザは既存のHttpOnly cookieを使用します。
RESTの更新操作には、管理画面と一致する`Origin`も必要です。

| 操作 | エンドポイント | 更新条件 |
|---|---|---|
| タスク一覧 | `GET /api/development/tasks` | 一覧・各タスクの版を返す |
| タスク取得 | `GET /api/development/tasks/{id}` | ETagはタスクの版 |
| タスク追加 | `POST /api/development/tasks` | `If-Match`に一覧の版、JSONにタスク項目 |
| タスク更新 | `PUT /api/development/tasks/{id}` | `If-Match`にそのタスクの版、JSONのidはURLと一致 |
| タスク削除 | `DELETE /api/development/tasks/{id}` | `If-Match`にそのタスクの版 |
| 文書一覧 | `GET /api/docs` | Markdownの配列と、ETagにGit HEADを返す |
| 文書保存 | `PUT /api/docs` | `path`, `content`, `revision`を指定 |

タスクの更新操作は更新後の一覧を返します。競合は409、不正な入力は400、
見つからないタスクは404、保存を確認できない場合は503です。
保存後のHTTP応答を失った場合も、再送前にGETで現在の版と内容を確認します。

## Akari MCPへの接続

Streamable HTTPのエンドポイントは既定で`http://127.0.0.1:8082/mcp`です。
公式Go MCP SDKを使い、JSON-RPC、初期化、ツール一覧、型付きの入出力を扱います。
stateless・JSON応答で、サーバーからクライアントへのsamplingや実行要求は行いません。

本番の接続先は`https://akari-mcp.shiron.dev/mcp`です。このホストはCloudflare Tunnelから
管理listenerへ接続し、Cloudflare Access Managed OAuthと`akari:admin`ロールで保護します。
Cloudflare Accessの不透明なOAuthトークンはエッジで認証され、originへ渡る署名付き
`Cf-Access-Jwt-Assertion`はAkariがissuer・audience・有効期限・RS256署名を検証します。
`AKARI_ACCESS_JWT_ISSUER`と`AKARI_ACCESS_JWT_AUD`は対で設定してください。前者はAccess team issuer URL、
後者はこのAccessアプリのAUD tagです。鍵はissuerの`/cdn-cgi/access/certs`から取得し、5分間キャッシュします。
本番では`AKARI_ADMIN_AUTH_DISABLED=true`を使いますが、Access JWT設定の片方だけが欠ける場合は起動に失敗します。
AkariのlistenerはDocker内部だけで待ち受け、ホストポートを公開せず、Tunnel以外から直接到達できない構成です。
リポジトリには次のCodexプロジェクト設定を置いています。リポジトリを信頼すると読み込まれ、
`codex mcp login akari`を実行するとCloudflare Accessの認証画面が開きます。

```toml
[mcp_servers.akari]
url = "https://akari-mcp.shiron.dev/mcp"
```

接続設定にはURLだけを保存します。CloudflareやAkariのトークンはGitへ保存せず、MCPのOAuth認証情報は
Codexのユーザー設定に保存されます。プロジェクト設定は信頼済みリポジトリでのみ読み込まれます。

認証ありでは管理用コードをBearerで送ります。コードは起動ごとに更新されるため、
再起動後はクライアントの環境変数を更新して再接続します。
`AKARI_ADMIN_AUTH_DISABLED=true`は既定では無効です。本番ではCloudflare Accessのエッジ認証、AkariのJWT検証、
Tunnel内の到達制限を併用するため、運用側で明示的に設定しています。

CodexのMCP設定例：

```toml
[mcp_servers.akari]
url = "http://127.0.0.1:8082/mcp"
bearer_token_env_var = "AKARI_MCP_TOKEN"
```

Codexを起動する環境の`AKARI_MCP_TOKEN`へ管理用コードを設定します。
トークンそのものを設定ファイルやGitへ書き込まないでください。

| ツール | 用途 |
|---|---|
| `tasks_list` | 一覧と版の取得。status / priorityで絞り込み |
| `tasks_get` | idでタスクを取得 |
| `tasks_create` | タイトルを指定して追加。状態todo・優先度medium・分類runtimeが既定 |
| `tasks_update` | 取得したTaskのversionを残して編集し、taskとして送る |
| `tasks_delete` | idとexpectedVersionで削除 |
| `documents_list` | `revision`と`documents`を返す。文書が0件でもGit HEADを取得可能 |
| `documents_read` | pathでMarkdownとrevisionを取得 |
| `documents_write` | 編集したDocumentを保存し、ローカルGitコミットを作成 |

このMCPは開発者用の入口です。AkariのChannelへ管理用トークンを渡さず、
外部MCPツールを自動でAkariのGatewayに登録することもしません。
Akariが道具を使う側のMCP接続は、引き続き[行為の境界](./03-effects.md)に従います。
文書が空の既存リポジトリでも、一覧から取得したHEADを`documents_write`のrevisionに指定して
最初のMarkdownを追加できます。コンソールも一覧のETagを使います。

## コーディングエージェントとの責務分担

AkariはタスクDB・Git管理のMarkdown・コンソール・MCP/APIを提供します。
開発タスクの実行と並列化は、MCPへ接続する外部のコーディングエージェントが担います。
Akariに開発実行ワーカー、スケジューラー、実行リース・heartbeat、worktree管理は実装しません。

コーディングエージェントがMCPからタスクと文書を取得し、自身の実行基盤で作業を進め、
状態・メモ・成果に関する文書をMCPへ書き戻します。実行対象の選択、並列数、取消、
再試行、二重実行防止、branchの分離・統合もコーディングエージェント側で管理します。
Akariのタスクの版照合は編集競合の検出であり、実行の排他や予約を保証しません。

Akari自身のChannelによる並列思考と行為のGatewayは、開発タスクの実行基盤とは別の責務です。

## 検証

SQLiteの保存復元・不正schema・競合・不正入力、Git履歴・競合・パス制限をテストします。
公式MCPクライアントで初期化・8ツールの発見・CRUD・Git保存を実際のHTTP通信で確認します。
ブラウザではコンソールとMCPの変更共有、下書き保持、JSON出力、Markdown編集、モバイル、
SIGKILL後の復元、認証切れ後の下書き復帰、保存中の閉じる操作の抑止、空の文書一覧からの追加、
ログアウトと外部リクエストなしを検証します。
