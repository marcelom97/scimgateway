# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
