# 同梱ブラウザライブラリ

実行時の外部配信は使いません。次のnpm配布物のブラウザ用bundleとライセンスを同梱しています。

| ライブラリ | バージョン | 元ファイル | 同梱先 |
|---|---|---|---|
| Mermaid | 11.12.2 | dist/mermaid.min.js | assets/mermaid.js |
| Marked | 17.0.1 | lib/marked.umd.js | assets/marked.js |
| DOMPurify | 3.3.0 | dist/purify.min.js | assets/purify.js |

ライセンスは同じassetsディレクトリの各バージョン名の`.license`ファイルです。
更新時は対応バージョンの`npm pack`からbundleとライセンスを取り出し、Go検証と
`scripts/browser`のブラウザ検証を実行します。npmやNodeを本番イメージへ追加しません。
