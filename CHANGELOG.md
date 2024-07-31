
0.7.0 / 2024-07-31
==================

  * feat: allow mapping hashes to subdomains (#48)
  * feat: show to which cache the server is bound
  * feat: add HTTP_ADDR env var
  * feat: add support for zstd decoding (#43)
  * feat: expose the executable bit into a HTTP header (#27)
  * change: NAR_CACHE_URL -> NIX_CACHE_URL

0.5.0 / 2021-07-16
==================

  * use goreleaser to manage releases (#16)

0.4.0 / 2021-07-16
==================

  * fix build
  * use the go-nix library again
  * bump dependencies
  * deploy: add AWS apprunner example
  * add support for local directory as a backend (#14)

0.3.0 / 2020-10-24
==================

  * main: fix PORT to addr logic
  * fix nix build
  * Add integration tests for nar-serve (#13)
  * Make nar-serve and go-nix monorepo (#12)
  * ci: no need to pull dependencies

0.2.0 / 2020-08-18
==================

  * Change default port to 8383 and NIX_CACHE_URI to NIX_CACHE_URL
  * Update vendorSha256 value from base-64 to base-32

0.1.0 / 2020-08-11
==================

  * update go-nix hash and refactor index.go to satisfy the new go-nix (#9)
  * Create go.yml
  * overlay: fix naming
  * fix vendorSha256
  * add overlay.nix file
  * fix the build
  * use the BinaryCacheReader interface
  * update gopath after ownership change
  * Merge pull request #6 from numtide/docker-image
  * add /healthz endpoint
  * add Dockerfile
  * Revert "Revert "stream the directory listing""
  * flakeify
  * cleanup
  * remove now.sh deployment
  * Revert "stream the directory listing"
  * stream the directory listing
  * README: move issues to GitHub issues
  * README: add note on .ls files
  * add directory listing
  * implement symlinks as HTTP redirects
  * README: one more known issue
  * introduce MountPath for the handlers
  * add robots.txt
  * README: fixes
  * work on the presentation for a bit
  * split up the api and public files
  * add ./start-dev script
  * add shell.nix
  * fix the deployment
  * fix file listing
  * make the cache configurable
  * now: respect the go modules pinning
  * Create LICENSE
  * fix deployment
  * init project
