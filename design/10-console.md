# 10. 管理コンソールとドキュメントの可視化

管理画面をGoの別listenerに接続しました。`internal/console`は管理側だけが使い、
ChannelやModelへ管理権限を渡しません。外部向けの状態表示とは別です。

## 起動と権限

通常のホストは`127.0.0.1:8080`、管理画面は`127.0.0.1:8082`です。
管理アドレスは`AKARI_ADMIN_ADDR`、Composeのホスト側ポートは`AKARI_ADMIN_PORT`で変更できます。
Composeは管理ポートもlocalhostへ公開し、既存の非root・読み取り専用root filesystemを維持します。
GoLandの既存Run構成で、そのまま両方を起動します。

状態復元とプロセスlease取得後、管理listenerをbindして、管理用コードを生成します。
コードは保存ディレクトリの`admin.token`へ0600で書き、標準ログやURLへ出しません。
起動ごとに更新し、前回のコードとcookieは使えなくなります。
ログインは同一originのPOSTで行い、HttpOnly / SameSite=Strictのセッションcookieを設定します。
APIのBearer認証も同じコードを使います。ログアウトはcookieを破棄します。
同一origin以外からのログイン・ログアウトを拒否し、APIをCORSへ公開しません。

`AKARI_ADMIN_AUTH_DISABLED=true`を指定すると、管理用コードを作らず、管理APIのアプリケーション認証を
無効にします。管理listenerは信頼できるネットワーク内、またはCloudflare Accessなどの認証付きプロキシの
内側に限定してください。既定値はfalseです。

TLS終端プロキシからHTTPで転送する場合は、`X-Forwarded-Proto: https`を使って外部のschemeを確認し、
セッションcookieにもSecure属性を付けます。プロキシから到達可能な管理listenerだけで利用してください。

現在の権限はobserver一種類です。認証後も閲覧だけを許可し、採用・承認・取消・dispatchのHTTP機能を
追加していません。生のExportをHTTPへ返さず、原文・人格値・記憶・宛先・ツール引数・結果を表示しません。
管理listenerの起動失敗は全体の起動失敗として状態所有者を閉じます。
listenerが異常終了した場合も、ホストへ停止を伝えます。

この初期実装の既定値はローカル開発向けです。管理画面を公開する場合は、TLS終端プロキシと認証ポリシーを
構成し、管理listener自体は外部へ直接公開しません。ユーザーごとの権限はまだありません。

## 画面

| 画面 | 表示と操作 |
|---|---|
| 設計マップ | docsとdesignを切り替え、本文の相互参照をSVGで表示。関連ノードの強調、拡大縮小、本文への移動 |
| ドキュメント | タイトルと本文の検索、仕様・技術設計の絞り込み、Markdown原文の表示 |
| 本文 | 見出しの目次、関連資料、文書間・章内リンク、Mermaidの図と拡大縮小・原文確認 |
| 稼働状態 | 採用版、保存Sequence、Schema、承認待ち、outboxの段階別件数、共有文脈・作業記憶・関心の件数 |

関係図の線は原文のリンクです。実装依存や心理的な因果関係を推測して描いたものではありません。
本文の図はdocsにあるMermaidを描画し、図の原文も確認できます。
描画できない図では、原文を残して失敗を表示します。
現在の状態は画面を開いた時と更新操作で取得します。常設Channelやモデルの稼働を仮の値で表示しません。
Foundationの状態基盤と、未接続のChannel・LLM・MCPを画面にも明示します。

## 同梱と更新

docsとdesignのMarkdownを`make generate`でJSONにまとめ、画面と一緒にGoバイナリへ埋め込みます。
`make build` / `make run`でも生成します。Dockerは生成済みの同じ内容を使います。
実行時にリポジトリや任意のファイルをHTTPから読むことはできません。
CIは再生成による差分を確認し、原文と同梱データのずれを拒否します。

画面はHTML / CSS / JavaScriptです。Nodeはブラウザ検証だけに使い、GoLandやDockerの実行には不要です。
Markdown・SVGをサニタイズし、scriptの同一origin制限、外部画像の制限、frame禁止、no-storeを設定します。
Mermaid、Marked、DOMPurifyはバージョン固定のローカル配信で、実行中のCDN・外部サービスを使いません。
ライブラリとライセンスは[同梱記録](../akari/internal/console/THIRD_PARTY.md)にあります。

## 検証

Goのテストで未認証拒否、同一originのログイン・ログアウト、cookie属性、本文の同梱、
概要に機密原文を混ぜないこと、コードの更新と0600、bind・書込み失敗、listener停止を確認します。
実バイナリのsmokeでは、管理ポートの認証と通常ポートに管理APIがないことを確認します。
ブラウザ検証は検索、相互参照、docsのすべてのMermaid図、拡大縮小、状態表示、モバイル幅、
ログアウト、JavaScript例外と外部リクエストがないことを確認します。CIにも組み込みました。

承認・取消・照合の操作画面、Channelの一覧・開始時刻、錠への到達、モデルの遅延・コスト、
管理ユーザーと権限の拡張は後続です。基礎APIにある機能を、未確認の権限でそのまま公開しません。
