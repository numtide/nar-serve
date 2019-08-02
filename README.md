# $ nar-serve # unpack and serve NAR file content

Put this in front of a Nix binary cache to serve it's content unpacked.

Since you want to push things to a binary cache, might as well avoid a second
publishing step for release artifacts.

## Use cases

* Inspect the content of a NAR file
* Publish a static website
* Publish any static assets

## Development

To start the build loop, run `nix-shell --run ./start-dev`.
