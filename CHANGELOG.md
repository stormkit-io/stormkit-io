# Changelog

## v2026.03.20.1...v2026.03.20.2

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v2026.03.20.1...v2026.03.20.2)

### 🚀 Enhancements

- Add public API endpoint to publish a deployment ([ec05fd2](https://github.com/stormkit-io/stormkit-io/commit/ec05fd2))
- Audit deployment publish actions ([bff13ba](https://github.com/stormkit-io/stormkit-io/commit/bff13ba))

### 🩹 Fixes

- Failing spec regarding nil license ([bdbefff](https://github.com/stormkit-io/stormkit-io/commit/bdbefff))
- Failing spec ([6e4a7a7](https://github.com/stormkit-io/stormkit-io/commit/6e4a7a7))

### 💅 Refactors

- Replace nil license checks with IsEmpty method ([69c621c](https://github.com/stormkit-io/stormkit-io/commit/69c621c))

### 🏡 Chore

- Update changelog for v2026.03.20.1 ([cbd3b6a](https://github.com/stormkit-io/stormkit-io/commit/cbd3b6a))
- Remove reference to percentage field ([341f0fd](https://github.com/stormkit-io/stormkit-io/commit/341f0fd))
- Use shorthand syntax ([3a93302](https://github.com/stormkit-io/stormkit-io/commit/3a93302))
- Do not return nil for license ([56e6ae2](https://github.com/stormkit-io/stormkit-io/commit/56e6ae2))
- Allow only successful deployments to be published ([38d86d0](https://github.com/stormkit-io/stormkit-io/commit/38d86d0))
- Update docs ([ea82c20](https://github.com/stormkit-io/stormkit-io/commit/ea82c20))
- Remove cloud check ([5efb724](https://github.com/stormkit-io/stormkit-io/commit/5efb724))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v2026.03.19.2...v2026.03.20.1

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v2026.03.19.2...v2026.03.20.1)

### 🚀 Enhancements

- Add GET /v1/deployments/{id} public API endpoint ([3fef53d](https://github.com/stormkit-io/stormkit-io/commit/3fef53d))

### 📖 Documentation

- Announce volumes api ([bb89f19](https://github.com/stormkit-io/stormkit-io/commit/bb89f19))

### 🏡 Chore

- Update changelog for v2026.03.19.2 ([ff6e2fe](https://github.com/stormkit-io/stormkit-io/commit/ff6e2fe))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v2026.03.19.1...v2026.03.19.2

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v2026.03.19.1...v2026.03.19.2)

### 🚀 Enhancements

- Public api for uploading files ([f4acafa](https://github.com/stormkit-io/stormkit-io/commit/f4acafa))

### 📖 Documentation

- V2026.03.19.1 ([21339dc](https://github.com/stormkit-io/stormkit-io/commit/21339dc))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v2026.03.17.1...v2026.03.19.1

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v2026.03.17.1...v2026.03.19.1)

### 🚀 Enhancements

- Include env and app ids from form values ([01f1a41](https://github.com/stormkit-io/stormkit-io/commit/01f1a41))
- Major improvements to public api ([d738eda](https://github.com/stormkit-io/stormkit-io/commit/d738eda))

### 🩹 Fixes

- Debug payload ([04950f3](https://github.com/stormkit-io/stormkit-io/commit/04950f3))

### 📖 Documentation

- V2026.03.17.1 ([76079ea](https://github.com/stormkit-io/stormkit-io/commit/76079ea))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v2026.03.10.1...v2026.03.17.1

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v2026.03.10.1...v2026.03.17.1)

### 🚀 Enhancements

- **public/v1:** Add WithAPIKey middleware and GET /v1/apps/{appId} endpoint ([e4cf28e](https://github.com/stormkit-io/stormkit-io/commit/e4cf28e))

### 🩹 Fixes

- Host name ([bc9e065](https://github.com/stormkit-io/stormkit-io/commit/bc9e065))

### 📖 Documentation

- V2026.03.10.1 ([42a43e6](https://github.com/stormkit-io/stormkit-io/commit/42a43e6))
- V2026.03.10.1 ([b054380](https://github.com/stormkit-io/stormkit-io/commit/b054380))
- Announce arm support ([ca5b5a5](https://github.com/stormkit-io/stormkit-io/commit/ca5b5a5))

### 🏡 Chore

- Improve logging ([2b13fbb](https://github.com/stormkit-io/stormkit-io/commit/2b13fbb))
- Additional logs ([afb990c](https://github.com/stormkit-io/stormkit-io/commit/afb990c))
- Additional logs ([20e0a98](https://github.com/stormkit-io/stormkit-io/commit/20e0a98))
- Additional logs ([e3facba](https://github.com/stormkit-io/stormkit-io/commit/e3facba))
- Improve debugging ([3a489d2](https://github.com/stormkit-io/stormkit-io/commit/3a489d2))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.16...v2026.03.10.1

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.16...v2026.03.10.1)

### 🚀 Enhancements

- Allow deleting production environments ([d8162c8](https://github.com/stormkit-io/stormkit-io/commit/d8162c8))
- Handle redirect and session flow ([6d83558](https://github.com/stormkit-io/stormkit-io/commit/6d83558))
- **api-keys:** Hash stored keys with SHA-256 and extract shared UI component ([5b4bb12](https://github.com/stormkit-io/stormkit-io/commit/5b4bb12))
- Api endpoint for retrieving apps ([4ad0bbb](https://github.com/stormkit-io/stormkit-io/commit/4ad0bbb))

### 🩹 Fixes

- Optional chaining ([b757bd6](https://github.com/stormkit-io/stormkit-io/commit/b757bd6))
- Import vitest utils ([dfed853](https://github.com/stormkit-io/stormkit-io/commit/dfed853))
- Failing specs ([b0d5364](https://github.com/stormkit-io/stormkit-io/commit/b0d5364))

### 📖 Documentation

- V1.26.16 ([ed77c08](https://github.com/stormkit-io/stormkit-io/commit/ed77c08))
- Announce calver ([0ea25cd](https://github.com/stormkit-io/stormkit-io/commit/0ea25cd))

### 🏡 Chore

- Remove unused file ([674099c](https://github.com/stormkit-io/stormkit-io/commit/674099c))
- Rename object ([7eda680](https://github.com/stormkit-io/stormkit-io/commit/7eda680))
- Allow specifying success url callback ([49c24d8](https://github.com/stormkit-io/stormkit-io/commit/49c24d8))
- Reorder items ([4fda639](https://github.com/stormkit-io/stormkit-io/commit/4fda639))
- Allow modifying chip sx ([ed75c1a](https://github.com/stormkit-io/stormkit-io/commit/ed75c1a))
- Minor ui changes ([5bdba7f](https://github.com/stormkit-io/stormkit-io/commit/5bdba7f))
- Do not expose authconf ([18327c0](https://github.com/stormkit-io/stormkit-io/commit/18327c0))
- Update ip address ([e09c33d](https://github.com/stormkit-io/stormkit-io/commit/e09c33d))
- Improve code ([805c7b6](https://github.com/stormkit-io/stormkit-io/commit/805c7b6))
- Reuse store ([80b04bd](https://github.com/stormkit-io/stormkit-io/commit/80b04bd))
- Invert test case ([ae782bb](https://github.com/stormkit-io/stormkit-io/commit/ae782bb))
- Remove redundant mock ([cce8d5f](https://github.com/stormkit-io/stormkit-io/commit/cce8d5f))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.15...v1.26.16

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.15...v1.26.16)

### 🚀 Enhancements

- Implement pcke verification for x auth ([7b3d330](https://github.com/stormkit-io/stormkit-io/commit/7b3d330))

### 📖 Documentation

- V1.26.15 ([746ce70](https://github.com/stormkit-io/stormkit-io/commit/746ce70))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.14...v1.26.15

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.14...v1.26.15)

### 🩹 Fixes

- Failing spec ([bcf2c0f](https://github.com/stormkit-io/stormkit-io/commit/bcf2c0f))

### 📖 Documentation

- V1.26.14 ([0776d8c](https://github.com/stormkit-io/stormkit-io/commit/0776d8c))

### 🏡 Chore

- Add client-specific oauth2 methods ([4af324f](https://github.com/stormkit-io/stormkit-io/commit/4af324f))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.13...v1.26.14

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.13...v1.26.14)

### 🚀 Enhancements

- Guide user through the authentication process ([4ae97fd](https://github.com/stormkit-io/stormkit-io/commit/4ae97fd))
- Stop the flow for disabled providers ([aa7b9f3](https://github.com/stormkit-io/stormkit-io/commit/aa7b9f3))

### 📖 Documentation

- V1.26.13 ([cfb38ba](https://github.com/stormkit-io/stormkit-io/commit/cfb38ba))

### 🏡 Chore

- Improve provider and auth logic ([471830e](https://github.com/stormkit-io/stormkit-io/commit/471830e))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.12...v1.26.13

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.12...v1.26.13)

### 🚀 Enhancements

- Refresh list after saving provider ([b33d51a](https://github.com/stormkit-io/stormkit-io/commit/b33d51a))

### 📖 Documentation

- V1.26.12 ([ab350af](https://github.com/stormkit-io/stormkit-io/commit/ab350af))

### 🏡 Chore

- Follow api standards for auth public apis ([6c97c36](https://github.com/stormkit-io/stormkit-io/commit/6c97c36))
- Improve documentation steps ([043264b](https://github.com/stormkit-io/stormkit-io/commit/043264b))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.11...v1.26.12

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.11...v1.26.12)

### 🏡 Chore

- Make sure to close the connection ([fea7e33](https://github.com/stormkit-io/stormkit-io/commit/fea7e33))
- Add auth behind feature flag ([05d9a33](https://github.com/stormkit-io/stormkit-io/commit/05d9a33))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.10...v1.26.11

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.10...v1.26.11)

### 🩹 Fixes

- Zip deployments with empty configs ([728d82d](https://github.com/stormkit-io/stormkit-io/commit/728d82d))

### 📖 Documentation

- V1.26.10 ([ae0136e](https://github.com/stormkit-io/stormkit-io/commit/ae0136e))

### 🏡 Chore

- Add specs for fetch config flow ([c8b6141](https://github.com/stormkit-io/stormkit-io/commit/c8b6141))
- Remove empty line ([cd15b41](https://github.com/stormkit-io/stormkit-io/commit/cd15b41))
- Minor code improvements ([ec2ecf5](https://github.com/stormkit-io/stormkit-io/commit/ec2ecf5))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.9...v1.26.10

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.9...v1.26.10)

### 🩹 Fixes

- User api key removal ([e353584](https://github.com/stormkit-io/stormkit-io/commit/e353584))
- Typography ([d9bb705](https://github.com/stormkit-io/stormkit-io/commit/d9bb705))

### 📖 Documentation

- V1.26.9 ([edf6310](https://github.com/stormkit-io/stormkit-io/commit/edf6310))
- V1.26.9 ([bdad06a](https://github.com/stormkit-io/stormkit-io/commit/bdad06a))
- V1.26.10 ([31a6b4f](https://github.com/stormkit-io/stormkit-io/commit/31a6b4f))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.8...v1.26.9

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.8...v1.26.9)

### 🚀 Enhancements

- Add support for user-level api keys ([2a2d054](https://github.com/stormkit-io/stormkit-io/commit/2a2d054))

### 🩹 Fixes

- Parameter order ([1648526](https://github.com/stormkit-io/stormkit-io/commit/1648526))
- Failing spec ([8e46e30](https://github.com/stormkit-io/stormkit-io/commit/8e46e30))

### 📖 Documentation

- User level api key ([4f29f09](https://github.com/stormkit-io/stormkit-io/commit/4f29f09))
- Announce user level api keys ([2c728d6](https://github.com/stormkit-io/stormkit-io/commit/2c728d6))

### 🏡 Chore

- Improve code resilience ([e633cce](https://github.com/stormkit-io/stormkit-io/commit/e633cce))
- Better log ([853b642](https://github.com/stormkit-io/stormkit-io/commit/853b642))
- Add debug statements for failed queries ([5dc33cc](https://github.com/stormkit-io/stormkit-io/commit/5dc33cc))
- Do not populate user id automatically ([2324382](https://github.com/stormkit-io/stormkit-io/commit/2324382))
- Improve test coverage ([bc53c3a](https://github.com/stormkit-io/stormkit-io/commit/bc53c3a))
- Improve test resilience ([d7b05cc](https://github.com/stormkit-io/stormkit-io/commit/d7b05cc))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.7...v1.26.8

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.7...v1.26.8)

### 🚀 Enhancements

- Job to dump schema into structure.sql ([c298a30](https://github.com/stormkit-io/stormkit-io/commit/c298a30))

### 🩹 Fixes

- Handle long referrer and request paths ([9cf9e07](https://github.com/stormkit-io/stormkit-io/commit/9cf9e07))
- Null values ([eafdb00](https://github.com/stormkit-io/stormkit-io/commit/eafdb00))
- Redirect to correct page ([a8f3351](https://github.com/stormkit-io/stormkit-io/commit/a8f3351))
- Failing spec ([02cf649](https://github.com/stormkit-io/stormkit-io/commit/02cf649))

### 📖 Documentation

- V1.26.7 ([d6f1c24](https://github.com/stormkit-io/stormkit-io/commit/d6f1c24))
- V1.26.8 ([59e94c3](https://github.com/stormkit-io/stormkit-io/commit/59e94c3))

### 🏡 Chore

- Use advisory lock instead of lock table ([29e98d9](https://github.com/stormkit-io/stormkit-io/commit/29e98d9))
- Rename files ([c14a214](https://github.com/stormkit-io/stormkit-io/commit/c14a214))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.6...v1.26.7

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.6...main)

### 🚀 Enhancements

- Move mailer to its own section ([e8d57ba](https://github.com/stormkit-io/stormkit-io/commit/e8d57ba))
- Inject mailer variables into the deployment ([1b65511](https://github.com/stormkit-io/stormkit-io/commit/1b65511))

### 🩹 Fixes

- **hosting:** Reduce lock contention in FetchAppConf ([e6f6775](https://github.com/stormkit-io/stormkit-io/commit/e6f6775))
- Failing specs ([3035baf](https://github.com/stormkit-io/stormkit-io/commit/3035baf))
- URL-encode credentials in SMTP connection string ([4360187](https://github.com/stormkit-io/stormkit-io/commit/4360187))
- Add missing dependencies to useEffect hook ([ee719c0](https://github.com/stormkit-io/stormkit-io/commit/ee719c0))
- Update mailer url description value ([c1a9103](https://github.com/stormkit-io/stormkit-io/commit/c1a9103))

### 💅 Refactors

- Move mailer under buildconf package ([7918faa](https://github.com/stormkit-io/stormkit-io/commit/7918faa))

### 📖 Documentation

- Document MAILER_URL injection and override behavior ([c5dca2d](https://github.com/stormkit-io/stormkit-io/commit/c5dca2d))
- Update MAILER_URL description to SMTP connection string ([d3dac79](https://github.com/stormkit-io/stormkit-io/commit/d3dac79))
- V1.26.6 ([7a39aa9](https://github.com/stormkit-io/stormkit-io/commit/7a39aa9))

### 🏡 Chore

- New helper component to display info tables ([985abcd](https://github.com/stormkit-io/stormkit-io/commit/985abcd))
- Minor ui improvements ([325e1c8](https://github.com/stormkit-io/stormkit-io/commit/325e1c8))
- Add clarification on existing vars ([d68a2c8](https://github.com/stormkit-io/stormkit-io/commit/d68a2c8))
- Reset form errors on submission ([5a1cf85](https://github.com/stormkit-io/stormkit-io/commit/5a1cf85))
- Ensure object is initialized ([d8d9452](https://github.com/stormkit-io/stormkit-io/commit/d8d9452))
- Fallback to default port number ([8269761](https://github.com/stormkit-io/stormkit-io/commit/8269761))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.5...v1.26.6

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.5...v1.26.6)

### 🏡 Chore

- Redact database_url variable ([1a3f829](https://github.com/stormkit-io/stormkit-io/commit/1a3f829))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.4...v1.26.5

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.4...v1.26.5)

### 🚀 Enhancements

- Abort functionality for mise update ([56a9d21](https://github.com/stormkit-io/stormkit-io/commit/56a9d21))
- Stormkit auth ([35db203](https://github.com/stormkit-io/stormkit-io/commit/35db203))
- Inject db environment variables ([2438c96](https://github.com/stormkit-io/stormkit-io/commit/2438c96))

### 🩹 Fixes

- Refresh references after a deployment is complete ([0f2553f](https://github.com/stormkit-io/stormkit-io/commit/0f2553f))
- Syntax highlighting ([3b9b220](https://github.com/stormkit-io/stormkit-io/commit/3b9b220))
- Rendering options in the correct order ([deff36e](https://github.com/stormkit-io/stormkit-io/commit/deff36e))
- Use correct ending tag ([79356e2](https://github.com/stormkit-io/stormkit-io/commit/79356e2))
- Endpoints order ([caff499](https://github.com/stormkit-io/stormkit-io/commit/caff499))
- Race condition ([ba4a9d7](https://github.com/stormkit-io/stormkit-io/commit/ba4a9d7))

### 📖 Documentation

- Add Go runtime application guide ([24e5612](https://github.com/stormkit-io/stormkit-io/commit/24e5612))

### 🏡 Chore

- Add logs ([f78f878](https://github.com/stormkit-io/stormkit-io/commit/f78f878))
- Add info ([ad171be](https://github.com/stormkit-io/stormkit-io/commit/ad171be))
- Use combined output for more information ([0f8d994](https://github.com/stormkit-io/stormkit-io/commit/0f8d994))
- Use scan instead of keys ([b60fa56](https://github.com/stormkit-io/stormkit-io/commit/b60fa56))
- Update packages ([53ad711](https://github.com/stormkit-io/stormkit-io/commit/53ad711))
- Log only if there are artifacts to be deleted ([9cfebdb](https://github.com/stormkit-io/stormkit-io/commit/9cfebdb))
- New switch component ([846b677](https://github.com/stormkit-io/stormkit-io/commit/846b677))
- Update theme ([b8e514a](https://github.com/stormkit-io/stormkit-io/commit/b8e514a))
- Disable stormkit authentication url ([735f51c](https://github.com/stormkit-io/stormkit-io/commit/735f51c))
- Add specs for new functionality in auth callback ([69d07ea](https://github.com/stormkit-io/stormkit-io/commit/69d07ea))
- Improve error message ([172436b](https://github.com/stormkit-io/stormkit-io/commit/172436b))
- Start using version format ([28c0a09](https://github.com/stormkit-io/stormkit-io/commit/28c0a09))
- Add specs for switch component ([b4ceb73](https://github.com/stormkit-io/stormkit-io/commit/b4ceb73))
- Comment out group icon for now ([08ccc29](https://github.com/stormkit-io/stormkit-io/commit/08ccc29))
- Update packages ([7da0353](https://github.com/stormkit-io/stormkit-io/commit/7da0353))
- Add specs ([03f9eaa](https://github.com/stormkit-io/stormkit-io/commit/03f9eaa))
- Wait for selectors ([0747158](https://github.com/stormkit-io/stormkit-io/commit/0747158))
- Add specs ([fc8538f](https://github.com/stormkit-io/stormkit-io/commit/fc8538f))
- Use correct tag ([9bd4b8a](https://github.com/stormkit-io/stormkit-io/commit/9bd4b8a))
- Add global code tag styling and UI improvements ([f77bec5](https://github.com/stormkit-io/stormkit-io/commit/f77bec5))
- Prevent default ([f58d7ab](https://github.com/stormkit-io/stormkit-io/commit/f58d7ab))
- Remove unnecessary file ([5301107](https://github.com/stormkit-io/stormkit-io/commit/5301107))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>
- Robertocommit ([@MilhosOU](https://github.com/MilhosOU))

## v1.26.1...v1.26.4

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.1...v1.26.4)

### 🩹 Fixes

- Sync env across components ([de8ba2b](https://github.com/stormkit-io/stormkit-io/commit/de8ba2b))

### 🏡 Chore

- New migration job ([23d8a2b](https://github.com/stormkit-io/stormkit-io/commit/23d8a2b))
- Limit subquery results ([46bb098](https://github.com/stormkit-io/stormkit-io/commit/46bb098))
- Unused vars ([57b6f4f](https://github.com/stormkit-io/stormkit-io/commit/57b6f4f))
- Upgrade mise version ([32fbb12](https://github.com/stormkit-io/stormkit-io/commit/32fbb12))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.26.0...v1.26.1

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.26.0...v1.26.1)

### 🚀 Enhancements

- Prepare endpoints for stormkit auth ([a57b52f](https://github.com/stormkit-io/stormkit-io/commit/a57b52f))

### 🩹 Fixes

- Failing specs ([4363cba](https://github.com/stormkit-io/stormkit-io/commit/4363cba))
- Json name ([df03e4a](https://github.com/stormkit-io/stormkit-io/commit/df03e4a))
- Add missing mock methods ([98bf2da](https://github.com/stormkit-io/stormkit-io/commit/98bf2da))
- Flaky test ([462e22d](https://github.com/stormkit-io/stormkit-io/commit/462e22d))
- App type ([2ebe8b9](https://github.com/stormkit-io/stormkit-io/commit/2ebe8b9))

### 📖 Documentation

- V1.26.0 ([a516709](https://github.com/stormkit-io/stormkit-io/commit/a516709))
- Update screenshot ([a8020e2](https://github.com/stormkit-io/stormkit-io/commit/a8020e2))
- Add mockery docs ([129cc38](https://github.com/stormkit-io/stormkit-io/commit/129cc38))
- Blog post on recent migration ([cff3b19](https://github.com/stormkit-io/stormkit-io/commit/cff3b19))

### 🏡 Chore

- Remove environments page ([0acd3ac](https://github.com/stormkit-io/stormkit-io/commit/0acd3ac))
- Regenerate files with new mockery version ([65cf718](https://github.com/stormkit-io/stormkit-io/commit/65cf718))
- Update packages ([ba9b445](https://github.com/stormkit-io/stormkit-io/commit/ba9b445))
- Allow using different secrets ([a124732](https://github.com/stormkit-io/stormkit-io/commit/a124732))
- Remove redundant row ([e739b87](https://github.com/stormkit-io/stormkit-io/commit/e739b87))
- Add support for new features ([4f6904e](https://github.com/stormkit-io/stormkit-io/commit/4f6904e))
- Helper functions for scanning bytea columns ([0379570](https://github.com/stormkit-io/stormkit-io/commit/0379570))
- Remove fmt ([8e2506e](https://github.com/stormkit-io/stormkit-io/commit/8e2506e))
- Add eof ([3fa9cba](https://github.com/stormkit-io/stormkit-io/commit/3fa9cba))
- Remove fmt debug ([8891f18](https://github.com/stormkit-io/stormkit-io/commit/8891f18))
- Store account id and make provider unique for each user ([c5b671a](https://github.com/stormkit-io/stormkit-io/commit/c5b671a))
- Remove debug ([1dede74](https://github.com/stormkit-io/stormkit-io/commit/1dede74))
- Take into account the provider status ([231d73b](https://github.com/stormkit-io/stormkit-io/commit/231d73b))
- Handle timed out deployments ([d8f370d](https://github.com/stormkit-io/stormkit-io/commit/d8f370d))
- Remove info statement ([b34a04d](https://github.com/stormkit-io/stormkit-io/commit/b34a04d))
- Allow modifying access keys through env vars ([8e7f397](https://github.com/stormkit-io/stormkit-io/commit/8e7f397))
- Remove log ([8c85d0d](https://github.com/stormkit-io/stormkit-io/commit/8c85d0d))
- Use equality instead of includes ([000e34c](https://github.com/stormkit-io/stormkit-io/commit/000e34c))
- Improve self-hosted license logic ([61fde66](https://github.com/stormkit-io/stormkit-io/commit/61fde66))
- Guard license ([4a9de52](https://github.com/stormkit-io/stormkit-io/commit/4a9de52))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.25.0

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.25.0...main)

### 🚀 Enhancements

- Update pricing ([4a6ed54](https://github.com/stormkit-io/stormkit-io/commit/4a6ed54))
- Endpoint to fetch env schema ([1d8a0c0](https://github.com/stormkit-io/stormkit-io/commit/1d8a0c0))
- Endpoint to create schema ([93f5531](https://github.com/stormkit-io/stormkit-io/commit/93f5531))
- Attach schema to env ([44aac9f](https://github.com/stormkit-io/stormkit-io/commit/44aac9f))
- Create dedicated users for schemas ([c27846e](https://github.com/stormkit-io/stormkit-io/commit/c27846e))
- Upgrade mise ([bacd0c9](https://github.com/stormkit-io/stormkit-io/commit/bacd0c9))
- Configure schema migrations ([331ad21](https://github.com/stormkit-io/stormkit-io/commit/331ad21))
- Move env navigation on the side ([d3a0d67](https://github.com/stormkit-io/stormkit-io/commit/d3a0d67))
- Configure schema ([f419f41](https://github.com/stormkit-io/stormkit-io/commit/f419f41))
- Store migrations path for deployment ([0cd61b4](https://github.com/stormkit-io/stormkit-io/commit/0cd61b4))
- Migrate deployment upload results to jsonb structure ([0537d48](https://github.com/stormkit-io/stormkit-io/commit/0537d48))
- Store migrations zip with deployment artifacts ([9e2e2ca](https://github.com/stormkit-io/stormkit-io/commit/9e2e2ca))
- Include only sql files in the migrations zip ([c188bb5](https://github.com/stormkit-io/stormkit-io/commit/c188bb5))
- Run schema migrations ([34a2000](https://github.com/stormkit-io/stormkit-io/commit/34a2000))
- Display migration details in deployment logs ([79f5cd2](https://github.com/stormkit-io/stormkit-io/commit/79f5cd2))
- Endpoint to drop schema ([8181def](https://github.com/stormkit-io/stormkit-io/commit/8181def))
- Add support for deleting schemas ([674e205](https://github.com/stormkit-io/stormkit-io/commit/674e205))
- Track audit logs ([0c8f81c](https://github.com/stormkit-io/stormkit-io/commit/0c8f81c))
- Enable database integrations for self-hosted environments ([b93214a](https://github.com/stormkit-io/stormkit-io/commit/b93214a))
- Stop next steps if migrations fail ([72424cc](https://github.com/stormkit-io/stormkit-io/commit/72424cc))

### 🩹 Fixes

- Comment ([09f68ee](https://github.com/stormkit-io/stormkit-io/commit/09f68ee))
- Broken links ([a6bccfb](https://github.com/stormkit-io/stormkit-io/commit/a6bccfb))
- Asset paths ([bf1f74a](https://github.com/stormkit-io/stormkit-io/commit/bf1f74a))
- Ttl ([d5663ed](https://github.com/stormkit-io/stormkit-io/commit/d5663ed))
- Failing spec ([dff8826](https://github.com/stormkit-io/stormkit-io/commit/dff8826))

### 💅 Refactors

- Use mui instead of tailwind ([12b0ceb](https://github.com/stormkit-io/stormkit-io/commit/12b0ceb))
- Code styling ([56b1a20](https://github.com/stormkit-io/stormkit-io/commit/56b1a20))
- Code styling ([9298e5b](https://github.com/stormkit-io/stormkit-io/commit/9298e5b))
- Use switch ([ed38d35](https://github.com/stormkit-io/stormkit-io/commit/ed38d35))
- Rename field ([6ee3d72](https://github.com/stormkit-io/stormkit-io/commit/6ee3d72))
- Improve code quality ([f9aacea](https://github.com/stormkit-io/stormkit-io/commit/f9aacea))

### 📖 Documentation

- Document new feature ([df44126](https://github.com/stormkit-io/stormkit-io/commit/df44126))
- Introduce database integration ([782d813](https://github.com/stormkit-io/stormkit-io/commit/782d813))
- Add self-hosted banner ([89d6c7d](https://github.com/stormkit-io/stormkit-io/commit/89d6c7d))
- Add screenshots ([edf6956](https://github.com/stormkit-io/stormkit-io/commit/edf6956))
- Attach version ([a23c419](https://github.com/stormkit-io/stormkit-io/commit/a23c419))

### 🏡 Chore

- Minor styling improvements ([fce11d4](https://github.com/stormkit-io/stormkit-io/commit/fce11d4))
- Implement ui structure for database access ([a257062](https://github.com/stormkit-io/stormkit-io/commit/a257062))
- Enable schema endpoints for development ([dc47148](https://github.com/stormkit-io/stormkit-io/commit/dc47148))
- Use appropriate names for variables ([fc6f670](https://github.com/stormkit-io/stormkit-io/commit/fc6f670))
- Add restart command ([f773731](https://github.com/stormkit-io/stormkit-io/commit/f773731))
- Return nil when schema does not exist ([b4fefe3](https://github.com/stormkit-io/stormkit-io/commit/b4fefe3))
- Use a clearer language for feature description ([ea42ab2](https://github.com/stormkit-io/stormkit-io/commit/ea42ab2))
- Handle empty schemas ([59a50d3](https://github.com/stormkit-io/stormkit-io/commit/59a50d3))
- Use restart instead of stop and start ([97d91aa](https://github.com/stormkit-io/stormkit-io/commit/97d91aa))
- Parameterize ssl mode ([37878e5](https://github.com/stormkit-io/stormkit-io/commit/37878e5))
- Quote role names ([181ad43](https://github.com/stormkit-io/stormkit-io/commit/181ad43))
- Remove if not exists ([d9a1be8](https://github.com/stormkit-io/stormkit-io/commit/d9a1be8))
- Return status conflict when schema already exists ([59a8269](https://github.com/stormkit-io/stormkit-io/commit/59a8269))
- Improve code quality ([15f77de](https://github.com/stormkit-io/stormkit-io/commit/15f77de))
- Add limits to app user ([b8581f9](https://github.com/stormkit-io/stormkit-io/commit/b8581f9))
- Rename variable ([c6f461d](https://github.com/stormkit-io/stormkit-io/commit/c6f461d))
- Improve subtitle ([d2bff82](https://github.com/stormkit-io/stormkit-io/commit/d2bff82))
- Better mobile support ([7af6c2f](https://github.com/stormkit-io/stormkit-io/commit/7af6c2f))
- Remove unused fields and endpoints ([ef6c172](https://github.com/stormkit-io/stormkit-io/commit/ef6c172))
- Remove unused field ([0f8ffa5](https://github.com/stormkit-io/stormkit-io/commit/0f8ffa5))
- Remove unused types and methods ([6a11b78](https://github.com/stormkit-io/stormkit-io/commit/6a11b78))
- Remove development guard ([6466954](https://github.com/stormkit-io/stormkit-io/commit/6466954))
- Add support for zipping only certain files ([992c521](https://github.com/stormkit-io/stormkit-io/commit/992c521))
- New zip iterator method ([5663245](https://github.com/stormkit-io/stormkit-io/commit/5663245))
- Default ssl mode to disable ([a85ce3e](https://github.com/stormkit-io/stormkit-io/commit/a85ce3e))
- Get file should download zip content only for sk-client files ([9ddbf17](https://github.com/stormkit-io/stormkit-io/commit/9ddbf17))
- Helper method to create zips in memory ([6d0d05d](https://github.com/stormkit-io/stormkit-io/commit/6d0d05d))
- Export variable ([a13b350](https://github.com/stormkit-io/stormkit-io/commit/a13b350))
- Ignore vite folder ([e930e43](https://github.com/stormkit-io/stormkit-io/commit/e930e43))
- Add migration id column ([5b4fc80](https://github.com/stormkit-io/stormkit-io/commit/5b4fc80))
- Use path instead of filepath ([b1c6ec3](https://github.com/stormkit-io/stormkit-io/commit/b1c6ec3))
- Return 204 ([c0fddb5](https://github.com/stormkit-io/stormkit-io/commit/c0fddb5))
- Move mutex one level above ([32c0775](https://github.com/stormkit-io/stormkit-io/commit/32c0775))
- Handle close properly ([62f1a69](https://github.com/stormkit-io/stormkit-io/commit/62f1a69))
- Skip migrations when not on default branch ([965dd84](https://github.com/stormkit-io/stormkit-io/commit/965dd84))
- Return error ([57ccce2](https://github.com/stormkit-io/stormkit-io/commit/57ccce2))
- Handle error responses better ([fb187ca](https://github.com/stormkit-io/stormkit-io/commit/fb187ca))
- Store error in db ([e948bf9](https://github.com/stormkit-io/stormkit-io/commit/e948bf9))
- Sanitize inputs ([71d7fc1](https://github.com/stormkit-io/stormkit-io/commit/71d7fc1))
- Make sure response is not nil ([1e586d8](https://github.com/stormkit-io/stormkit-io/commit/1e586d8))
- Pass down glob pattern ([2b6b7bf](https://github.com/stormkit-io/stormkit-io/commit/2b6b7bf))
- Improve sanitization ([6dab458](https://github.com/stormkit-io/stormkit-io/commit/6dab458))
- Remove app_members table which is no longer used ([48e01e1](https://github.com/stormkit-io/stormkit-io/commit/48e01e1))
- Minor improvements to team members logic ([db7f99c](https://github.com/stormkit-io/stormkit-io/commit/db7f99c))
- Add reset-data command ([8e1bdf6](https://github.com/stormkit-io/stormkit-io/commit/8e1bdf6))
- Use method from deploy package ([2780c5c](https://github.com/stormkit-io/stormkit-io/commit/2780c5c))
- Add helper methods for acquiring db locks ([0ae324c](https://github.com/stormkit-io/stormkit-io/commit/0ae324c))
- Remove flaky statement ([090ef1e](https://github.com/stormkit-io/stormkit-io/commit/090ef1e))
- Use background context ([44a2c45](https://github.com/stormkit-io/stormkit-io/commit/44a2c45))
- Return result even when storing migration fails ([c6b103b](https://github.com/stormkit-io/stormkit-io/commit/c6b103b))
- Use single test to not break transactions ([28073f5](https://github.com/stormkit-io/stormkit-io/commit/28073f5))
- Helper method to fetch a single team member ([3962269](https://github.com/stormkit-io/stormkit-io/commit/3962269))
- Check if member exists ([73accdb](https://github.com/stormkit-io/stormkit-io/commit/73accdb))
- Remove casting to seconds ([1e12475](https://github.com/stormkit-io/stormkit-io/commit/1e12475))
- Use route53 pckg for dns management ([38d3723](https://github.com/stormkit-io/stormkit-io/commit/38d3723))
- Disable aws logs ([5ae1214](https://github.com/stormkit-io/stormkit-io/commit/5ae1214))
- Use google trust certificate ([4223b5d](https://github.com/stormkit-io/stormkit-io/commit/4223b5d))
- Update issuer ca ([72c7e8c](https://github.com/stormkit-io/stormkit-io/commit/72c7e8c))
- Make ca configurable ([4ac7774](https://github.com/stormkit-io/stormkit-io/commit/4ac7774))
- Logs should be opt-in ([61c2bbc](https://github.com/stormkit-io/stormkit-io/commit/61c2bbc))
- Specify hosted zone ([038701e](https://github.com/stormkit-io/stormkit-io/commit/038701e))
- Use static credentials ([c624999](https://github.com/stormkit-io/stormkit-io/commit/c624999))
- Remove unused file ([bc0b092](https://github.com/stormkit-io/stormkit-io/commit/bc0b092))
- Remove unused field ([1ee30c6](https://github.com/stormkit-io/stormkit-io/commit/1ee30c6))
- Inject postgres vars when migrations are enabled ([9d450c0](https://github.com/stormkit-io/stormkit-io/commit/9d450c0))
- Minor tweaks for db integration ([b963f44](https://github.com/stormkit-io/stormkit-io/commit/b963f44))
- Include database_url environment variable ([5d52cc5](https://github.com/stormkit-io/stormkit-io/commit/5d52cc5))
- Introduce database page ([69847f1](https://github.com/stormkit-io/stormkit-io/commit/69847f1))
- Use sql highlighter ([a900153](https://github.com/stormkit-io/stormkit-io/commit/a900153))
- Display database icon ([54911e4](https://github.com/stormkit-io/stormkit-io/commit/54911e4))
- Use correct parameters ([629d7fb](https://github.com/stormkit-io/stormkit-io/commit/629d7fb))
- Display alert for cloud users ([96f79b1](https://github.com/stormkit-io/stormkit-io/commit/96f79b1))
- Do not use prepared statements for migrations ([51bf8fc](https://github.com/stormkit-io/stormkit-io/commit/51bf8fc))
- Update package locks ([1a30bb1](https://github.com/stormkit-io/stormkit-io/commit/1a30bb1))
- Use filepath instead of path ([7d067ab](https://github.com/stormkit-io/stormkit-io/commit/7d067ab))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.25.0

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.24.0...main)

### 🚀 Enhancements

- Allow managing auth config ([57faf80](https://github.com/stormkit-io/stormkit-io/commit/57faf80))
- Add whitelist logic ([bf4eba2](https://github.com/stormkit-io/stormkit-io/commit/bf4eba2))
- Endpoint to retrieve pending users ([1044573](https://github.com/stormkit-io/stormkit-io/commit/1044573))
- Allow managing pending users ([8c2c253](https://github.com/stormkit-io/stormkit-io/commit/8c2c253))
- Allow modifying pending users ([1865ada](https://github.com/stormkit-io/stormkit-io/commit/1865ada))
- Run www service ([70371e4](https://github.com/stormkit-io/stormkit-io/commit/70371e4))
- Approval mode for ee users ([baebfd2](https://github.com/stormkit-io/stormkit-io/commit/baebfd2))

### 🩹 Fixes

- Make sure not to return 0 ([dbbd99f](https://github.com/stormkit-io/stormkit-io/commit/dbbd99f))
- Grammar ([50e7bec](https://github.com/stormkit-io/stormkit-io/commit/50e7bec))
- Typescript warnings ([2ba67f2](https://github.com/stormkit-io/stormkit-io/commit/2ba67f2))
- Generating doc routes ([135a2bb](https://github.com/stormkit-io/stormkit-io/commit/135a2bb))

### 💅 Refactors

- Reorganize docs ([95ba421](https://github.com/stormkit-io/stormkit-io/commit/95ba421))

### 📖 Documentation

- V1.24.0 ([88ca93c](https://github.com/stormkit-io/stormkit-io/commit/88ca93c))
- Update pull request template ([8775362](https://github.com/stormkit-io/stormkit-io/commit/8775362))
- Update docs for www ([8a6d476](https://github.com/stormkit-io/stormkit-io/commit/8a6d476))
- Update authentication documentation ([ac14aa1](https://github.com/stormkit-io/stormkit-io/commit/ac14aa1))
- New sign up mode feature ([3348a0c](https://github.com/stormkit-io/stormkit-io/commit/3348a0c))
- Styling ([3bb0560](https://github.com/stormkit-io/stormkit-io/commit/3bb0560))

### 🏡 Chore

- Remove unused import ([5c2e2fb](https://github.com/stormkit-io/stormkit-io/commit/5c2e2fb))
- Add watch mode for fe tests ([e2877ed](https://github.com/stormkit-io/stormkit-io/commit/e2877ed))
- Minor style adjustments ([4a47b3f](https://github.com/stormkit-io/stormkit-io/commit/4a47b3f))
- Use any instead of interface ([21334ed](https://github.com/stormkit-io/stormkit-io/commit/21334ed))
- Allow modifying modal title ([ac282de](https://github.com/stormkit-io/stormkit-io/commit/ac282de))
- Improve approval logic ([030ebed](https://github.com/stormkit-io/stormkit-io/commit/030ebed))
- Add specs for new method ([72f9f3a](https://github.com/stormkit-io/stormkit-io/commit/72f9f3a))
- Allow setting config in test envs ([3712e12](https://github.com/stormkit-io/stormkit-io/commit/3712e12))
- Remove wrapping brackets ([8f7291c](https://github.com/stormkit-io/stormkit-io/commit/8f7291c))
- Move landing page to this repository ([3bad0f1](https://github.com/stormkit-io/stormkit-io/commit/3bad0f1))
- Remove license ([863602e](https://github.com/stormkit-io/stormkit-io/commit/863602e))
- Parse port flag ([9bc251f](https://github.com/stormkit-io/stormkit-io/commit/9bc251f))
- Move file to scripts ([54f96f3](https://github.com/stormkit-io/stormkit-io/commit/54f96f3))
- Add support for new docs location ([32ad078](https://github.com/stormkit-io/stormkit-io/commit/32ad078))
- Ignore dist folder ([818aee5](https://github.com/stormkit-io/stormkit-io/commit/818aee5))
- Update path ([f7dbb9c](https://github.com/stormkit-io/stormkit-io/commit/f7dbb9c))
- Add more routes to prerender ([8d65542](https://github.com/stormkit-io/stormkit-io/commit/8d65542))
- Check content ([722eea6](https://github.com/stormkit-io/stormkit-io/commit/722eea6))
- Update sort order ([1e4eaaa](https://github.com/stormkit-io/stormkit-io/commit/1e4eaaa))
- Mv ui to home folder ([0f44788](https://github.com/stormkit-io/stormkit-io/commit/0f44788))
- Use different location ([43f356e](https://github.com/stormkit-io/stormkit-io/commit/43f356e))
- Apply sign up check for all platforms ([8107faa](https://github.com/stormkit-io/stormkit-io/commit/8107faa))
- Improve design ([d6180d7](https://github.com/stormkit-io/stormkit-io/commit/d6180d7))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>

## v1.24.0

[compare changes](https://github.com/stormkit-io/stormkit-io/compare/v1.23.0...main)

### 🚀 Enhancements

- Add user management ([782e78e](https://github.com/stormkit-io/stormkit-io/commit/782e78e))
- Prepare make file for windows environment ([19d6cc6](https://github.com/stormkit-io/stormkit-io/commit/19d6cc6))
- Add support for unix environments ([3aa8fe4](https://github.com/stormkit-io/stormkit-io/commit/3aa8fe4))
- Implement build-tag based image optimization decoupling ([fb1d309](https://github.com/stormkit-io/stormkit-io/commit/fb1d309))
- Further windows optimization ([b6fd035](https://github.com/stormkit-io/stormkit-io/commit/b6fd035))
- Add support for rsyc on windows ([1d12483](https://github.com/stormkit-io/stormkit-io/commit/1d12483))
- Remove stormkit ui auto installation ([4949340](https://github.com/stormkit-io/stormkit-io/commit/4949340))

### 🩹 Fixes

- **auth:** Display backend validation errors on signup ([e0a3399](https://github.com/stormkit-io/stormkit-io/commit/e0a3399))
- **ui:** Link to correct page ([e391258](https://github.com/stormkit-io/stormkit-io/commit/e391258))
- File name ([b5e4e7b](https://github.com/stormkit-io/stormkit-io/commit/b5e4e7b))
- Ui cmd for windows ([bcd4109](https://github.com/stormkit-io/stormkit-io/commit/bcd4109))
- Link path ([d5b5d03](https://github.com/stormkit-io/stormkit-io/commit/d5b5d03))
- Testing link path ([50c783a](https://github.com/stormkit-io/stormkit-io/commit/50c783a))

### 📖 Documentation

- Add notice on installing mise ([d17f427](https://github.com/stormkit-io/stormkit-io/commit/d17f427))
- Adds section for hosts file config and troubleshooting ([#11](https://github.com/stormkit-io/stormkit-io/pull/11))
- Update documentation on image optimization ([373a58f](https://github.com/stormkit-io/stormkit-io/commit/373a58f))
- Fix path ([dd6a683](https://github.com/stormkit-io/stormkit-io/commit/dd6a683))
- Use make instead of custom scripts ([9370806](https://github.com/stormkit-io/stormkit-io/commit/9370806))
- Document how to test and run stormkit locally ([091211a](https://github.com/stormkit-io/stormkit-io/commit/091211a))
- Move troubleshooting to its dedicated page ([26637a4](https://github.com/stormkit-io/stormkit-io/commit/26637a4))

### 🏡 Chore

- Run go mod tidy ([9dfe407](https://github.com/stormkit-io/stormkit-io/commit/9dfe407))
- Add frontend tests to the workflow ([cf48df9](https://github.com/stormkit-io/stormkit-io/commit/cf48df9))
- Rename workflow file ([67316f1](https://github.com/stormkit-io/stormkit-io/commit/67316f1))
- Remove only modifiers ([fa43d36](https://github.com/stormkit-io/stormkit-io/commit/fa43d36))
- Remove hardcoded platform ([eeacf60](https://github.com/stormkit-io/stormkit-io/commit/eeacf60))
- Wait for db and redis to be ready ([d271e1e](https://github.com/stormkit-io/stormkit-io/commit/d271e1e))
- Delete file ([3bff991](https://github.com/stormkit-io/stormkit-io/commit/3bff991))
- Auto generate .env file on start ([c8ea299](https://github.com/stormkit-io/stormkit-io/commit/c8ea299))
- Expand and reorganize BotList entries ([#25](https://github.com/stormkit-io/stormkit-io/pull/25))
- Clean up bot list ([85140d8](https://github.com/stormkit-io/stormkit-io/commit/85140d8))
- Placeholder app secret ([5b5fee4](https://github.com/stormkit-io/stormkit-io/commit/5b5fee4))
- Pass build flags env variable ([0e642ca](https://github.com/stormkit-io/stormkit-io/commit/0e642ca))
- Use make to run tests ([b968d8f](https://github.com/stormkit-io/stormkit-io/commit/b968d8f))
- Add build tag to test file ([276ebd8](https://github.com/stormkit-io/stormkit-io/commit/276ebd8))
- Use cross platform sleep ([881d98b](https://github.com/stormkit-io/stormkit-io/commit/881d98b))
- Build runner ([350b13b](https://github.com/stormkit-io/stormkit-io/commit/350b13b))
- Skip mise installation in development env ([6b82ae8](https://github.com/stormkit-io/stormkit-io/commit/6b82ae8))
- Export missing env variables for runner ([11b32aa](https://github.com/stormkit-io/stormkit-io/commit/11b32aa))
- Add windows build tag ([c7e1857](https://github.com/stormkit-io/stormkit-io/commit/c7e1857))
- Add support for windows commands ([48d027e](https://github.com/stormkit-io/stormkit-io/commit/48d027e))
- Add helper for detecting windows os ([ed9f6a4](https://github.com/stormkit-io/stormkit-io/commit/ed9f6a4))
- Add specs for npm installer ([324ba20](https://github.com/stormkit-io/stormkit-io/commit/324ba20))
- Use filepath instead of path ([3aa7522](https://github.com/stormkit-io/stormkit-io/commit/3aa7522))
- Add support for single file transfers ([ed02a72](https://github.com/stormkit-io/stormkit-io/commit/ed02a72))
- Use filepath dir instead of path dir ([ca49d1c](https://github.com/stormkit-io/stormkit-io/commit/ca49d1c))
- Use a better path ([e2567de](https://github.com/stormkit-io/stormkit-io/commit/e2567de))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>
- Taiizor <galata.80@hotmail.com>
- Roberto.commit ([@MilhosOU](https://github.com/MilhosOU))
- Buggato <ivan.lori@protonmail.com>
- Copilot ([@MicrosoftCopilot](https://github.com/MicrosoftCopilot))

## v1.23.0

### 🩹 Fixes

- App secret length ([07e7ea4](https://github.com/stormkit-io/stormkit-io/commit/07e7ea4))
- Return false when statement is not yet ready ([f55169b](https://github.com/stormkit-io/stormkit-io/commit/f55169b))
- Auth wall login api should be public ([984353b](https://github.com/stormkit-io/stormkit-io/commit/984353b))
- Handle empty hash case ([a2e3164](https://github.com/stormkit-io/stormkit-io/commit/a2e3164))
- Storing certificates in redis cache ([36e493e](https://github.com/stormkit-io/stormkit-io/commit/36e493e))

### 💅 Refactors

- Use any instead of interface ([9432d43](https://github.com/stormkit-io/stormkit-io/commit/9432d43))
- Pass down parameters in correct order ([27522b9](https://github.com/stormkit-io/stormkit-io/commit/27522b9))
- Rewrite spec for readability and maintainability ([db716d4](https://github.com/stormkit-io/stormkit-io/commit/db716d4))

### 📖 Documentation

- Add steps for installing tools ([412ac5d](https://github.com/stormkit-io/stormkit-io/commit/412ac5d))
- Remove reference to dnsmasq ([48d0e04](https://github.com/stormkit-io/stormkit-io/commit/48d0e04))

### 🏡 Chore

- Open source ([b631970](https://github.com/stormkit-io/stormkit-io/commit/b631970))
- Remove unused field ([80dc4d4](https://github.com/stormkit-io/stormkit-io/commit/80dc4d4))
- Ignore socket file ([f6e8ba5](https://github.com/stormkit-io/stormkit-io/commit/f6e8ba5))
- Update packages ([1e37994](https://github.com/stormkit-io/stormkit-io/commit/1e37994))
- Update packages ([64d2ef6](https://github.com/stormkit-io/stormkit-io/commit/64d2ef6))
- Disable maintenance notifications ([49e5738](https://github.com/stormkit-io/stormkit-io/commit/49e5738))
- Use new structs from updated version ([49b0596](https://github.com/stormkit-io/stormkit-io/commit/49b0596))
- Add eof line ([56cfdc1](https://github.com/stormkit-io/stormkit-io/commit/56cfdc1))
- Use correct references ([e689493](https://github.com/stormkit-io/stormkit-io/commit/e689493))
- Revert order ([d735df9](https://github.com/stormkit-io/stormkit-io/commit/d735df9))
- Remove unnecessary statement ([ae7b975](https://github.com/stormkit-io/stormkit-io/commit/ae7b975))
- New script to generate git tags ([a974ba4](https://github.com/stormkit-io/stormkit-io/commit/a974ba4))
- New workflow to run tests on each pr ([87a37bf](https://github.com/stormkit-io/stormkit-io/commit/87a37bf))
- Set secret ([5718309](https://github.com/stormkit-io/stormkit-io/commit/5718309))
- Add extra check ([bc611f0](https://github.com/stormkit-io/stormkit-io/commit/bc611f0))
- Update debug message ([7260cce](https://github.com/stormkit-io/stormkit-io/commit/7260cce))
- Use localhost instead of custom domain ([735d58d](https://github.com/stormkit-io/stormkit-io/commit/735d58d))
- Remove unused variable ([1b946d7](https://github.com/stormkit-io/stormkit-io/commit/1b946d7))
- Remove script as it is no longer needed ([485c134](https://github.com/stormkit-io/stormkit-io/commit/485c134))
- Debug domain info on startup ([f406c4f](https://github.com/stormkit-io/stormkit-io/commit/f406c4f))
- Move request debug to middleware ([2d0728a](https://github.com/stormkit-io/stormkit-io/commit/2d0728a))
- Use utils ptr instead of aws string ([3f1a7d9](https://github.com/stormkit-io/stormkit-io/commit/3f1a7d9))

### ❤️ Contributors

- Savas Vedova <savas@stormkit.io>
