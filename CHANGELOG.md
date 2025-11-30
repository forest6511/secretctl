# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Core vault implementation with AES-256-GCM encryption
- Argon2id key derivation for master password
- SQLite-based secret storage
- HMAC-chained audit logging with tamper detection
- Metadata support (notes, tags, URL, expiration)
- Master password strength validation
- File permission validation (0600/0700)
- CLI commands: init, lock, unlock, set, get, list, delete
- Audit commands: list, verify
