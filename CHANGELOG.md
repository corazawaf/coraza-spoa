# Changelog

## [0.3.0](https://github.com/corazawaf/coraza-spoa/compare/v0.2.0...v0.3.0) (2025-07-02)


### Features

* add SetServerName to transaction ([c5252cc](https://github.com/corazawaf/coraza-spoa/commit/c5252cc794600345c028de58f3047baedf732300))

## 0.2.0 (2025-06-26)


### Features

* add -version flag for printing version and build info ([#214](https://github.com/corazawaf/coraza-spoa/issues/214)) ([153d4fb](https://github.com/corazawaf/coraza-spoa/commit/153d4fb4677ed0ea4bedfb15cc6469ab89cb17ec)), closes [#117](https://github.com/corazawaf/coraza-spoa/issues/117)
* replace request hook in example ([07483bc](https://github.com/corazawaf/coraza-spoa/commit/07483bc005c0c9b28aed8f0e0cdb2cd595339ef6)), closes [#111](https://github.com/corazawaf/coraza-spoa/issues/111)
* Reuse haproxy unique_id if present ([bfd8b24](https://github.com/corazawaf/coraza-spoa/commit/bfd8b2466ecd6f52e7193a26250710ed803fe1ca))
* support the users to configure the traffic fields they need to forward in HAProxy configuration file ([11c9415](https://github.com/corazawaf/coraza-spoa/commit/11c9415375c76d9edfd43d711f2f9cfc890abe5d))


### Bug Fixes

* 5 by validating null queries ([d828d74](https://github.com/corazawaf/coraza-spoa/commit/d828d74f60896568c5bbf71eb0de045b986e8182))
* **build:** set arch in magefile ([8482824](https://github.com/corazawaf/coraza-spoa/commit/8482824b360c5d29c0d85296b55dfb22322c7439))
* **ci:** minor corrections from code review ([e3e7b9d](https://github.com/corazawaf/coraza-spoa/commit/e3e7b9df73ab5af3b67fb9705a9a3226cc25df5a))
* **ci:** only use main branch for tags ([69403c6](https://github.com/corazawaf/coraza-spoa/commit/69403c6b8a1124d30f37375cc03df3fb812a8fd7))
* **ci:** run build on all branches ([e8f614b](https://github.com/corazawaf/coraza-spoa/commit/e8f614ba55f49dbf965e7f10c90edb54d37dc9dd))
* **ci:** set correct build output dir ([5dba63d](https://github.com/corazawaf/coraza-spoa/commit/5dba63d9522688884cdad71c4d5ac643a698742c))
* **ci:** use variable instead of fixed name ([e619181](https://github.com/corazawaf/coraza-spoa/commit/e619181264be4ea9cc83463b165c6b0aeea132ec))
* **config:** image build ([#100](https://github.com/corazawaf/coraza-spoa/issues/100)) ([b93d995](https://github.com/corazawaf/coraza-spoa/commit/b93d995fca765c8f27db651fb57dafed84eec34a))
* **deps:** update all non-major dependencies in go.mod ([#207](https://github.com/corazawaf/coraza-spoa/issues/207)) ([1dfb95f](https://github.com/corazawaf/coraza-spoa/commit/1dfb95fad3a7efc7f40a71bef5dd4b47a16ce869))
* **deps:** update all non-major dependencies to v2.16.1 in go.mod ([cfcabe5](https://github.com/corazawaf/coraza-spoa/commit/cfcabe5b78150d0e953adcab945714fe32ac0978))
* **deps:** update all non-major dependencies to v2.17.1 in go.mod ([#193](https://github.com/corazawaf/coraza-spoa/issues/193)) ([e7b0f46](https://github.com/corazawaf/coraza-spoa/commit/e7b0f46dbb28d154e3938b2c8e0f4e118a580bc7))
* **deps:** update all non-major dependencies to v2.18.0 in go.mod ([#199](https://github.com/corazawaf/coraza-spoa/issues/199)) ([a76c32f](https://github.com/corazawaf/coraza-spoa/commit/a76c32fea62f4abbb20b7f3063d4f8f85a7bda4d))
* **deps:** update all non-major dependencies to v2.18.2 in go.mod ([#229](https://github.com/corazawaf/coraza-spoa/issues/229)) ([581a429](https://github.com/corazawaf/coraza-spoa/commit/581a429be6e556291afb8e6e3261c5b2962786d5))
* **deps:** update all non-major dependencies to v2.18.3 in go.mod ([#231](https://github.com/corazawaf/coraza-spoa/issues/231)) ([fbe673b](https://github.com/corazawaf/coraza-spoa/commit/fbe673bbf258eb0bb37a512ce335562be9dc0f08))
* **deps:** update github.com/magefile/mage digest to 32e0107 ([#141](https://github.com/corazawaf/coraza-spoa/issues/141)) ([543600d](https://github.com/corazawaf/coraza-spoa/commit/543600d94a5f331786a84c00a99da17a37abad09))
* **deps:** update github.com/magefile/mage digest to 78acbaf in go.mod ([#232](https://github.com/corazawaf/coraza-spoa/issues/232)) ([7acc427](https://github.com/corazawaf/coraza-spoa/commit/7acc427f246bdb469aaba9fa75ce69ca7c660286))
* **deps:** update module github.com/corazawaf/coraza-coreruleset/v4 to v4.14.0 in go.mod ([#218](https://github.com/corazawaf/coraza-spoa/issues/218)) ([6933218](https://github.com/corazawaf/coraza-spoa/commit/6933218a419f34996d3c6e83fdae1a8ce27360bf))
* **deps:** update module github.com/corazawaf/coraza-coreruleset/v4 to v4.15.0 in go.mod ([#236](https://github.com/corazawaf/coraza-spoa/issues/236)) ([72f72ea](https://github.com/corazawaf/coraza-spoa/commit/72f72ea27c7e202386e2bca2acd85321bfaa8acb))
* **deps:** update module github.com/corazawaf/coraza/v3 to v3.2.2 ([#131](https://github.com/corazawaf/coraza-spoa/issues/131)) ([de7faf4](https://github.com/corazawaf/coraza-spoa/commit/de7faf458f041a24b1dc9c391bc7d6a9d4ea1caa))
* **deps:** update module github.com/corazawaf/coraza/v3 to v3.3.0 ([#154](https://github.com/corazawaf/coraza-spoa/issues/154)) ([87d7dde](https://github.com/corazawaf/coraza-spoa/commit/87d7dde4fa95dc03a5c7aa5cb549c94943a33024))
* **deps:** update module github.com/corazawaf/coraza/v3 to v3.3.2 ([7bb4c86](https://github.com/corazawaf/coraza-spoa/commit/7bb4c86ee715ded8e28c5fd23093a4dcb704148b))
* **deps:** update module github.com/corazawaf/coraza/v3 to v3.3.3 [security] ([39a02d6](https://github.com/corazawaf/coraza-spoa/commit/39a02d68bd636a106859f2b6702268cb7d393a9b))
* **deps:** update module github.com/dropmorepackets/haproxy-go to v0.0.6 in go.mod ([735c7af](https://github.com/corazawaf/coraza-spoa/commit/735c7afb042e89d16d1c11922fae790210560e3a))
* **deps:** update module github.com/dropmorepackets/haproxy-go to v0.0.7 in go.mod ([#226](https://github.com/corazawaf/coraza-spoa/issues/226)) ([5aa72f0](https://github.com/corazawaf/coraza-spoa/commit/5aa72f0f3d3951cfa520d4545782c6402e9d43b0))
* **deps:** update module github.com/mccutchen/go-httpbin/v2 to v2.16.0 ([#172](https://github.com/corazawaf/coraza-spoa/issues/172)) ([b0e8fdc](https://github.com/corazawaf/coraza-spoa/commit/b0e8fdc1c7d4c9c119b24ab2cf5598a4ffd5a3b9))
* **deps:** update module github.com/pires/go-proxyproto to v0.8.0 ([#119](https://github.com/corazawaf/coraza-spoa/issues/119)) ([1046c72](https://github.com/corazawaf/coraza-spoa/commit/1046c725b17f056eae5e7e3334b357ac06be4662))
* **deps:** update module github.com/rs/zerolog to v1.34.0 in go.mod ([#202](https://github.com/corazawaf/coraza-spoa/issues/202)) ([cc7b577](https://github.com/corazawaf/coraza-spoa/commit/cc7b5772da1c203a9aa8f43d696c5b348b4f1e3c))
* renovate config ([6e33b60](https://github.com/corazawaf/coraza-spoa/commit/6e33b6016b87248e339e76620d980b95258f1e9e))
* revert golang major upgrade ([3bfad4f](https://github.com/corazawaf/coraza-spoa/commit/3bfad4f53b166be1c1711e6d6510e3d0f275ab77))
* run mage lint ([7321cc4](https://github.com/corazawaf/coraza-spoa/commit/7321cc460c8297e4eb03d66aaabf1a60495eee7c))


### Miscellaneous Chores

* release 0.2.0 ([#239](https://github.com/corazawaf/coraza-spoa/issues/239)) ([e9ce67e](https://github.com/corazawaf/coraza-spoa/commit/e9ce67e2b246de124b8dc0debefa352375ce284a))
