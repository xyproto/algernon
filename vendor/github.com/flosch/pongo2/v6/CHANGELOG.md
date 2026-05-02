# Changelog

All notable changes to this project will be documented in this file.

## [6.1.0] - 2026-05-02

This release is primarily a **security and bug-fix update**. All users are
encouraged to update.

### Fixed

- **`removetags` filter**: tag names containing regex metacharacters no longer
  panic the renderer.
- **`{% cycle %}` tag**: cycle index is now tracked per template execution
  instead of mutated on the parsed AST node. Concurrent renders of a cached
  template no longer race, and sequential renders no longer leak state from
  a previous execution.
- **`{% ifchanged %}` tag**: `lastValues`/`lastContent` are now tracked per
  template execution instead of mutated on the parsed AST node, fixing both
  a data race under concurrent renders and state leaking between sequential
  renders of a cached template.
- **`{% ifchanged %}` tag**: rendering an `{% ifchanged %}` block without
  an `{% else %}` branch no longer crashes with a nil-pointer dereference
  when the watched value is unchanged. Matches Django's behavior of
  producing no output.
- **`{% filter %}` tag**: `BanFilter` is now enforced inside `{% filter %}`
  blocks.

### Changed

- **`{% ssi %}` plaintext mode** now reads the included file through the
  configured `TemplateLoader` chain instead of `ioutil.ReadFile`, so
  non-filesystem loaders (`FSLoader`, `HttpFilesystemLoader`, custom)
  can serve SSI content.
- **Template error reporting** (`RawLine`) now reads source lines through
  the template's loader chain instead of opening files directly with
  `os.Open`. Error line extraction now works for any `TemplateLoader`.

### Removed

- **`SandboxedFilesystemLoader` and `NewSandboxedFilesystemLoader`** have
  been removed. They were marked WIP, never wired into any enforcement
  path, and behaved as a thin pass-through to `LocalFilesystemLoader`.
  Callers should use `LocalFilesystemLoader` directly; sandboxing should
  be implemented via a custom `TemplateLoader`.

  Note: this is technically an API-breaking removal, but the type was
  unused WIP code that never provided sandboxing.

### Documentation

- Clarified that pongo2 does **not** provide a true sandbox. `BanTag` and
  `BanFilter` only refuse to compile templates that reference banned
  names; they do not isolate Go execution, restrict filesystem access,
  or contain malicious templates. README, `TemplateSet` field comment,
  `DefaultLoader` comment, and parser error messages updated accordingly.
- Added a Security section to the README documenting that template
  loaders (`LocalFilesystemLoader`, `HttpFilesystemLoader`, `FSLoader`)
  do not clamp paths to a base directory and that template filenames
  must be treated as trusted input.

[6.1.0]: https://github.com/flosch/pongo2/compare/v6.0.0...v6.1.0
