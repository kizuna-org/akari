# Changelog

## [0.2.0](https://github.com/kizuna-org/akari/compare/v0.1.1...v0.2.0) (2026-02-08)


### Features

* **akari/internal/message:** Create Discord channel if not exist ([#108](https://github.com/kizuna-org/akari/issues/108)) ([5e728ff](https://github.com/kizuna-org/akari/commit/5e728ffe51054fa2e9fa1db79d6834bbc511d279))
* **akari/internal/message:** Create Discord guild if not exist ([#109](https://github.com/kizuna-org/akari/issues/109)) ([3ad82e0](https://github.com/kizuna-org/akari/commit/3ad82e019a16b12b449ba5b4325250f8affbb648))
* **akari/internal/message:** Create Discord user if not exist ([#110](https://github.com/kizuna-org/akari/issues/110)) ([61fdc57](https://github.com/kizuna-org/akari/commit/61fdc57b2b1ccf3c6652dcdcd9d50679c6b662fc))
* **akari/pkg/database:** Set user ID to conversation as well ([#115](https://github.com/kizuna-org/akari/issues/115)) ([f155b85](https://github.com/kizuna-org/akari/commit/f155b854ebc811509a32f822ba64d38dbf7f42e1))
* **akari/pkg/kiseki:** Initialize polling methods ([#148](https://github.com/kizuna-org/akari/issues/148)) ([c9a6960](https://github.com/kizuna-org/akari/commit/c9a6960e937b1191ba670d61be400b8dc9648b08))
* **akari:** add AkariUser integration to DiscordUserRepository ([#135](https://github.com/kizuna-org/akari/issues/135)) ([f75030e](https://github.com/kizuna-org/akari/commit/f75030e68315bb87275d882c99145803ea7d06f0))
* **akari:** Generate kiseki client code ([#143](https://github.com/kizuna-org/akari/issues/143)) ([44e529e](https://github.com/kizuna-org/akari/commit/44e529eb0f3db9b274c9aee0c7b46a7dfe67663d))
* **akari:** Store all Discord messages ([#99](https://github.com/kizuna-org/akari/issues/99)) ([9eea981](https://github.com/kizuna-org/akari/commit/9eea981b494d6080495cf3d094afaf678f70d819))
* **deploy:** add Docker configuration for Akari deploy ([#131](https://github.com/kizuna-org/akari/issues/131)) ([c8542d7](https://github.com/kizuna-org/akari/commit/c8542d7a52ff7459ed53546ff07e84f6cd0cc54a))
* **deploy:** add nocodb and cloudflare-tunnel services to compose.yml ([#136](https://github.com/kizuna-org/akari/issues/136)) ([4b91ea7](https://github.com/kizuna-org/akari/commit/4b91ea78e3a70f5277f85086f78d4ee8e7453242))
* **openapi:** add openapi ([#141](https://github.com/kizuna-org/akari/issues/141)) ([8aafc20](https://github.com/kizuna-org/akari/commit/8aafc205b09abc99412b6f76372fed631b8dda50))


### Bug Fixes

* **akari/internal/message:** Read bot name RegExp from database ([#140](https://github.com/kizuna-org/akari/issues/140)) ([f6c52a8](https://github.com/kizuna-org/akari/commit/f6c52a83d429c5cc619be3c7e8d6b622c32eb42d))
* **akari:** Copy kiseki OpenAPI config to the build ([#146](https://github.com/kizuna-org/akari/issues/146)) ([cf36e52](https://github.com/kizuna-org/akari/commit/cf36e52cd12fdba8c62063eb5c9348cf54b76015))
* **deps:** update module github.com/brianvoe/gofakeit/v7 to v7.14.0 ([#134](https://github.com/kizuna-org/akari/issues/134)) ([22f8e58](https://github.com/kizuna-org/akari/commit/22f8e5836ef16fa617d390a0898c225e6e16780e))
* **deps:** update module github.com/labstack/echo/v4 to v4.15.0 ([#142](https://github.com/kizuna-org/akari/issues/142)) ([85d729f](https://github.com/kizuna-org/akari/commit/85d729f4af474beadb6f16c74379e705279926d2))
* **deps:** update module github.com/lib/pq to v1.11.1 ([#175](https://github.com/kizuna-org/akari/issues/175)) ([f085ab0](https://github.com/kizuna-org/akari/commit/f085ab0dd533184e7aab6e7057a395f35474bc2e))
* **deps:** update module google.golang.org/genai to v1.36.0 ([#92](https://github.com/kizuna-org/akari/issues/92)) ([2529e37](https://github.com/kizuna-org/akari/commit/2529e37bdee818a9b9705c8a97475ae8b95a5d23))
* **deps:** update module google.golang.org/genai to v1.37.0 ([#105](https://github.com/kizuna-org/akari/issues/105)) ([10105e0](https://github.com/kizuna-org/akari/commit/10105e013225deb687c59fa0856cdf3d14fa34b8))
* **deps:** update module google.golang.org/genai to v1.38.0 ([#117](https://github.com/kizuna-org/akari/issues/117)) ([9918f7d](https://github.com/kizuna-org/akari/commit/9918f7d0a4f154bc2408e00bae6806e57ae7262a))
* **deps:** update module google.golang.org/genai to v1.39.0 ([#121](https://github.com/kizuna-org/akari/issues/121)) ([552fe2d](https://github.com/kizuna-org/akari/commit/552fe2d9e5e57d15216dd21e2300705a2eb1302d))
* **deps:** update module google.golang.org/genai to v1.40.0 ([#126](https://github.com/kizuna-org/akari/issues/126)) ([0448837](https://github.com/kizuna-org/akari/commit/0448837bf62ae8ce34689f66157930984e38137b))
* **deps:** update module google.golang.org/genai to v1.41.0 ([#153](https://github.com/kizuna-org/akari/issues/153)) ([eb198f3](https://github.com/kizuna-org/akari/commit/eb198f3f0272ffb09d8e709daaa67893f7db4199))
* **deps:** update module google.golang.org/genai to v1.41.1 ([#159](https://github.com/kizuna-org/akari/issues/159)) ([ec9ab4e](https://github.com/kizuna-org/akari/commit/ec9ab4eec2de2741a71f38cac64037428bea2b45))
* **deps:** update module google.golang.org/genai to v1.42.0 ([#161](https://github.com/kizuna-org/akari/issues/161)) ([38a305c](https://github.com/kizuna-org/akari/commit/38a305c1f0404d2c07ea85625b42059dd1cade13))
* **deps:** update module google.golang.org/genai to v1.43.0 ([#166](https://github.com/kizuna-org/akari/issues/166)) ([b7bcd96](https://github.com/kizuna-org/akari/commit/b7bcd96d415dd190f9bdc050cc38f78abe53cb77))
* **deps:** update module google.golang.org/genai to v1.44.0 ([#176](https://github.com/kizuna-org/akari/issues/176)) ([fa4093c](https://github.com/kizuna-org/akari/commit/fa4093c115120ecd4b762bf413891cdb36b48c54))
* **deps:** update module google.golang.org/genai to v1.45.0 ([#181](https://github.com/kizuna-org/akari/issues/181)) ([9a8dbc3](https://github.com/kizuna-org/akari/commit/9a8dbc3ccddb6f5351615093f38dd667923dad1a))
* **kiseki-gen:** Regenerate PostMemoryPolling ([#145](https://github.com/kizuna-org/akari/issues/145)) ([37f742f](https://github.com/kizuna-org/akari/commit/37f742f008939a568f4838d484c39ef3bbaee02d))
* Make NocoDB workable ([#139](https://github.com/kizuna-org/akari/issues/139)) ([ff44f98](https://github.com/kizuna-org/akari/commit/ff44f989684ef79ca03f03936b2acefe270b88ef))
* update discordUserRepository to implement DiscordUserRepository interface ([#130](https://github.com/kizuna-org/akari/issues/130)) ([779167c](https://github.com/kizuna-org/akari/commit/779167c4c73fa047dd2a7347ee48846c54d97e09))


### Reverts

* **akari:** Remove Go work ([#149](https://github.com/kizuna-org/akari/issues/149)) ([c25019c](https://github.com/kizuna-org/akari/commit/c25019cd29334c0dc29bffadb7e69696123bbee4))

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
