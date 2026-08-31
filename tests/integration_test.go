//go:build integration

// This test drives a real `minio` and a store populated by hand, so it is
// behind a tag rather than part of what `go test ./...` runs. See README.md
// for how to set the two up.
package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/numtide/nar-serve/pkg/libstore"
	"github.com/stretchr/testify/assert"
)

func cmd(env []string, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	return cmd
}

// waitForServer blocks until addr accepts connections, or gives up.
func waitForServer(addr string) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			return conn.Close()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s never came up: %w", addr, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestHappyPath(t *testing.T) {
	assert := assert.New(t)
	accessKeyID := "Q3AM3UQ867SPQQA43P2F"
	secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"

	addr := "127.0.0.1:9000"

	tempDir, err := ioutil.TempDir("", "nar-serve")
	if err != nil {
		t.Fatal("tmpdir error:", err)
	}
	defer os.RemoveAll(tempDir)

	dataDir := tempDir + "/data"
	homeDir := tempDir + "/home"

	// `nix copy` remembers per store URL which paths that store holds, and this
	// one is served from a directory that is empty every run.
	env := append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+accessKeyID,
		"AWS_SECRET_ACCESS_KEY="+secretAccessKey,
		"HOME="+homeDir,
	)

	// A bucket is a directory below the one being served, so creating it is
	// creating that directory.
	err = os.MkdirAll(filepath.Join(dataDir, "nsbucket"), 0o755)
	if err != nil {
		t.Fatal("bucket error:", err)
	}

	// Start the server
	server := cmd(env, "rclone", "serve", "s3", dataDir,
		"--addr", addr,
		"--auth-key", accessKeyID+","+secretAccessKey,
	)
	err = server.Start()
	if err != nil {
		t.Fatal("rclone error:", err)
	}
	defer func() {
		server.Process.Kill()
		server.Wait()
	}()

	err = waitForServer(addr)
	if err != nil {
		t.Fatal("rclone error:", err)
	}

	nix_copy := cmd(env, "nix", "copy", "--to", "s3://nsbucket?region=us-east-1&endpoint=127.0.0.1:9000&scheme=http", "/nix/store/irfa91bs2wfqyh2j9kl8m3rcg7h72w4m-curl-7.71.1-bin")
	err = nix_copy.Run()
	if err != nil {
		t.Fatal("nix-copy error:", err)
	}

	ctx := context.Background()

	tmpfile := filepath.Join(dataDir, "nsbucket/irfa91bs2wfqyh2j9kl8m3rcg7h72w4m.narinfo")
	_, err = os.Stat(tmpfile)

	if err != nil {
		if os.IsNotExist(err) {
			t.Fatal("File not exists")
		} else {
			t.Fatal("ERROR:", err)
		}
	}
	content, err := ioutil.ReadFile(tmpfile)

	if err != nil {
		t.Fatal(err)
	}

	// S3 binary cache storage
	r, err := libstore.NewBinaryCacheReader(ctx, "s3://nsbucket?region=us-east-1&endpoint=http://127.0.0.1:9000&scheme=http")
	if err != nil {
		t.Fatal("new binary cache error:", err)
	}

	os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secretAccessKey)
	obj, err := r.GetFile(ctx, "irfa91bs2wfqyh2j9kl8m3rcg7h72w4m.narinfo")
	if err != nil {
		t.Fatal("get file error:", err)
	}

	obj_content, err_read := ioutil.ReadAll(obj)
	if err_read != nil {
		t.Fatal(err_read)
	}

	same_content := bytes.Equal(content, obj_content)
	assert.True(same_content, "The content is not the same")

	is_exist, err := r.FileExists(ctx, "irfa91bs2wfqyh2j9kl8m3rcg7h72w4m.narinfo")
	if err != nil {
		t.Fatal("file exist error:", err)
	}
	assert.True(is_exist, "File is not existed")
	// Stop the server
	server.Process.Kill()
}
