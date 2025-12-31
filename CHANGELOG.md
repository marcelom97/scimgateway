# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.2.2] - 2025-12-31


## Changelog
### Bug Fixes
* fix: clean up CHANGELOG.md formatting and improve release automation (@marcelom97)

**Full Changelog**: https://github.com/marcelom97/scimgateway/compare/v0.2.1...v0.2.2

[v0.2.2]: https://github.com/marcelom97/scimgateway/compare/v0.2.1...v0.2.2


## [v0.2.1] - 2025-12-31

### Features
* feat: automate CHANGELOG.md updates on release (@marcelom97)

[v0.2.1]: https://github.com/marcelom97/scimgateway/compare/v0.2.0...v0.2.1

## [v0.2.0] - 2025-12-31

### Breaking Changes
* feat!: remove unused Type and BaseEntity fields from PluginConfig (@marcelom97)
* feat!: remove unused baseEntity parameter from Plugin interface (@marcelom97)

### Features
* feat: add GoReleaser integration for automated releases (@marcelom97)
* feat: add thread safety and comprehensive documentation to plugin package (@marcelom97)

### Performance
* perf(scim): optimize SortResources with value caching (@marcelom97)

### Refactoring
* refactor: rename root package to scimgateway for consistency with module path (@marcelom97)

### Documentation
* docs: add comprehensive PLUGIN_DEVELOPMENT.md guide (@marcelom97)

### Tests
* test: add integration tests for single resource attribute selection (@marcelom97)

[v0.2.0]: https://github.com/marcelom97/scimgateway/compare/v0.1.0...v0.2.0

## [v0.1.0] - 2025-10-26

### Features
* feat: initial implementation of SCIM Gateway
* feat: add support for string bool
* feat: implement SCIM-compliant error handling with proper status codes
* feat: support boolean-to-string comparison in SCIM filters
* feat: create array elements for ADD/REPLACE with filtered paths
* feat: add case-insensitive filtering for Microsoft SCIM compatibility

### Bug Fixes
* fix: support PATCH operations on filtered array sub-attributes
* fix: support SCIM enterprise extension attributes in PATCH operations

[v0.1.0]: https://github.com/marcelom97/scimgateway/releases/tag/v0.1.0
