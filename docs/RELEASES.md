# Releases

Higher uses stable Semantic Versioning tags in the `vMAJOR.MINOR.PATCH` format. Release tags with prerelease suffixes, such as `v1.0.0-rc.1`, are not supported by the current release workflow.

## Prepare a release

Start from an up-to-date, clean `main` branch. Run the project checks and build all release targets locally:

```bash
go test ./...
go vet ./...
go build ./...
./scripts/build.sh 1.2.3
```

The build script creates Linux amd64, Linux arm64, and Windows amd64 binaries in `dist/`, with the version, commit, and UTC build date embedded. Check the Linux binary with `./dist/higher-linux-amd64 version`.

Create and push an annotated tag from the validated commit:

```bash
git tag -a v1.2.3 -m "release: v1.2.3"
git push origin v1.2.3
```

The tag workflow repeats tests and vet, rebuilds all targets, verifies the embedded version, and publishes a GitHub Release with these assets:

- `higher-linux-amd64`
- `higher-linux-arm64`
- `higher-windows-amd64.exe`

The release workflow requires only the repository's `GITHUB_TOKEN`; it does not update package-manager manifests.
