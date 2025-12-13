# Changelog

## [0.2.0](https://github.com/kizuna-org/akari/compare/v0.1.1...v0.2.0) (2025-12-13)


### Features

* **akari/internal/message:** Create Discord channel if not exist ([#108](https://github.com/kizuna-org/akari/issues/108)) ([5e728ff](https://github.com/kizuna-org/akari/commit/5e728ffe51054fa2e9fa1db79d6834bbc511d279))
* **akari/internal/message:** Create Discord guild if not exist ([#109](https://github.com/kizuna-org/akari/issues/109)) ([3ad82e0](https://github.com/kizuna-org/akari/commit/3ad82e019a16b12b449ba5b4325250f8affbb648))
* **akari/internal/message:** Create Discord user if not exist ([#110](https://github.com/kizuna-org/akari/issues/110)) ([61fdc57](https://github.com/kizuna-org/akari/commit/61fdc57b2b1ccf3c6652dcdcd9d50679c6b662fc))
* **akari/pkg/database:** Set user ID to conversation as well ([#115](https://github.com/kizuna-org/akari/issues/115)) ([f155b85](https://github.com/kizuna-org/akari/commit/f155b854ebc811509a32f822ba64d38dbf7f42e1))
* **akari:** Store all Discord messages ([#99](https://github.com/kizuna-org/akari/issues/99)) ([9eea981](https://github.com/kizuna-org/akari/commit/9eea981b494d6080495cf3d094afaf678f70d819))


### Bug Fixes

* **deps:** update module google.golang.org/genai to v1.36.0 ([#92](https://github.com/kizuna-org/akari/issues/92)) ([2529e37](https://github.com/kizuna-org/akari/commit/2529e37bdee818a9b9705c8a97475ae8b95a5d23))
* **deps:** update module google.golang.org/genai to v1.37.0 ([#105](https://github.com/kizuna-org/akari/issues/105)) ([10105e0](https://github.com/kizuna-org/akari/commit/10105e013225deb687c59fa0856cdf3d14fa34b8))
* **deps:** update module google.golang.org/genai to v1.38.0 ([#117](https://github.com/kizuna-org/akari/issues/117)) ([9918f7d](https://github.com/kizuna-org/akari/commit/9918f7d0a4f154bc2408e00bae6806e57ae7262a))

## [0.1.1](https://github.com/kizuna-org/akari/compare/v0.1.0...v0.1.1) (2025-11-20)


### Bug Fixes

* **ci:** update release workflow to use release tag name for image tagging ([#88](https://github.com/kizuna-org/akari/issues/88)) ([51fba90](https://github.com/kizuna-org/akari/commit/51fba90091ed92789d06403ef227a92162daddf3))

## 0.1.0 (2025-11-20)


### Features

* **akari/cmd:** Merge all packages to work ([#56](https://github.com/kizuna-org/akari/issues/56)) ([f6d7e7e](https://github.com/kizuna-org/akari/commit/f6d7e7e6cce652e9a09acf4ff6d4fd292c30b064))
* **akari/ent:** Add relation from Akari user to Discord user ([#76](https://github.com/kizuna-org/akari/issues/76)) ([225db56](https://github.com/kizuna-org/akari/commit/225db563af5105b7515b9b494dc62add4ed0e476))
* **akari/ent:** Create a database schema and migration ([#35](https://github.com/kizuna-org/akari/issues/35)) ([bf1ea72](https://github.com/kizuna-org/akari/commit/bf1ea728c04cc40a0a7b12b5dea0d3a0da08e02e))
* **akari/pkg/database:** Create Akari user management package ([#74](https://github.com/kizuna-org/akari/issues/74)) ([bed8f80](https://github.com/kizuna-org/akari/commit/bed8f80687356cc264a6f6d3a76cd301f0b9038e))
* **akari/pkg/database:** Create character database schema ([#62](https://github.com/kizuna-org/akari/issues/62)) ([eda048d](https://github.com/kizuna-org/akari/commit/eda048d511d2eeb0e238425d0c4c1fe4279de146))
* **akari/pkg/database:** Create conversation management package ([#71](https://github.com/kizuna-org/akari/issues/71)) ([7a00ffb](https://github.com/kizuna-org/akari/commit/7a00ffbe99bed54f0a126c32341f8fc56848a908))
* **akari/pkg/database:** Create Discord message management package ([#64](https://github.com/kizuna-org/akari/issues/64)) ([d5ff3d3](https://github.com/kizuna-org/akari/commit/d5ff3d3da8c82a38c255a05b79dd7472dc044e03))
* **akari/pkg/database:** Create Discord user management package ([#73](https://github.com/kizuna-org/akari/issues/73)) ([8144193](https://github.com/kizuna-org/akari/commit/8144193c3b90a68709aca87f5c38e265fd8c739e))
* **akari/pkg/database:** Implement database package ([#36](https://github.com/kizuna-org/akari/issues/36)) ([666f919](https://github.com/kizuna-org/akari/commit/666f919ed57dcad3a76df2a4fd2ca2c9aa6a9a98))
* **ci:** Add ko-build-check and releaser ([#80](https://github.com/kizuna-org/akari/issues/80)) ([1b66b9e](https://github.com/kizuna-org/akari/commit/1b66b9ea1c566ea60adca65d65a1b21ece32bd1f))
* **discord:** impl discord ([#34](https://github.com/kizuna-org/akari/issues/34)) ([beacb6c](https://github.com/kizuna-org/akari/commit/beacb6c7eb27c11730176d0656b4ce1c55a702e7))
* init pj ([#1](https://github.com/kizuna-org/akari/issues/1)) ([82c9176](https://github.com/kizuna-org/akari/commit/82c9176ba23af4380684b91c6004a88d200e90c1))
* **llm:** implement Gemini for chat message handling ([#25](https://github.com/kizuna-org/akari/issues/25)) ([5468081](https://github.com/kizuna-org/akari/commit/546808188488a723b57506e941376aa7c96da249))


### Bug Fixes

* **deps:** update module google.golang.org/genai to v1.32.0 ([#30](https://github.com/kizuna-org/akari/issues/30)) ([e8ccccf](https://github.com/kizuna-org/akari/commit/e8ccccf64ae6a28270baf17fd71f232b9c1e474e))
* **deps:** update module google.golang.org/genai to v1.33.0 ([#33](https://github.com/kizuna-org/akari/issues/33)) ([bb5135f](https://github.com/kizuna-org/akari/commit/bb5135f6a69fcd7fc785b3a0a513135c1720d859))
* **deps:** update module google.golang.org/genai to v1.34.0 ([#49](https://github.com/kizuna-org/akari/issues/49)) ([7030cd4](https://github.com/kizuna-org/akari/commit/7030cd4b2007ae3de852eb86a46cd5be6033f366))
* **deps:** update module google.golang.org/genai to v1.35.0 ([#77](https://github.com/kizuna-org/akari/issues/77)) ([aa823a9](https://github.com/kizuna-org/akari/commit/aa823a9680993f41be1b5f07cd9d8f51174a329c))


### Miscellaneous Chores

* release 0.1.0 ([45a03b3](https://github.com/kizuna-org/akari/commit/45a03b366a91bcfaecebb7bfc38e083996d9c914))
