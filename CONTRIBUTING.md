# Contributing

## Commit messages

Use Conventional Commits with the package name as scope:

    feat(transport): ...   new behavior
    fix(driver): ...       bug fix
    test(config): ...      tests only
    docs: ...              documentation only
    refactor(store): ...   no behavior change
    chore: ...             tooling, dependencies, housekeeping

## Branches

- `master` is the published branch.
- Each phase is developed on `phase/N-name` and merged into `master`.

## Releases

- Versions follow semantic versioning. While the major version is 0, every release is a pre-release.
- Each release has an annotated tag, `vX.Y.Z`.
- The release notes are the matching section of CHANGELOG.md.