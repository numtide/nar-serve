# Tests

## Serve a bucket

A bucket is a directory below the one being served, so creating it is creating
that directory.

```shell
mkdir -p nar/nsbucket
rclone serve s3 ./nar --addr 127.0.0.1:9000 --auth-key accesskey,secretkey
```

## Fill it

```shell
AWS_ACCESS_KEY_ID=accesskey AWS_SECRET_ACCESS_KEY=secretkey nix copy --to "s3://nsbucket?region=us-east-1&endpoint=127.0.0.1:9000&scheme=http" /nix/store/irfa91bs2wfqyh2j9kl8m3rcg7h72w4m-curl-7.71.1-bin
```

## Run the test

```shell
go test -tags integration ./tests/
```
