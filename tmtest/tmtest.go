/*

Package tmtest provides helpers for testing using tendermint server.

*/
package tmtest

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/iov-one/weave/weavetest/assert"
)

// TestReporter is the minimal subset of testing.TB needed to run these test helpers
type TestReporter interface {
	assert.Tester
	Skipf(string, ...interface{})
	Logf(string, ...interface{})
}

// RunTendermint starts a tendermit process. Returned cleanup function will
// ensure the process has stopped and will block until.
//
// Set FORCE_TM_TEST=1 environment variable to fail the test if the binary is
// not available. This might be desired when running tests by CI.
//
// Set TM_DEBUG=1 environmental variable to output all tm logs
func RunTendermint(ctx context.Context, t TestReporter, home string) (cleanup func()) {
	t.Helper()

	tmpath, err := exec.LookPath("tendermint")
	if err != nil {
		if os.Getenv("FORCE_TM_TEST") != "1" {
			t.Skipf("Tendermint binary not found. Set FORCE_TM_TEST=1 to fail this test.")
		} else {
			t.Fatalf("Tendermint binary not found. Do not set FORCE_TM_TEST=1 to skip this test.")
		}
	}

	cmd := exec.CommandContext(ctx, tmpath, "node", "--home", home)
	// log tendermint output for verbose debugging....
	if os.Getenv("TM_DEBUG") != "" {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Tendermint process failed: %s", err)
	}

	// Give tendermint time to setup.
	time.Sleep(2 * time.Second)
	t.Logf("Running %s pid=%d", tmpath, cmd.Process.Pid)

	// Return a cleanup function, that will wait for the tendermint to stop.
	// We also auto-kill when the context is Done
	done := make(chan struct{})

	var once sync.Once
	cleanup = func() {
		once.Do(func() {
			t.Logf("tendermint cleanup called")
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			close(done)
		})

		// Block until the tendermint server process is gone.
		<-done
	}

	go func() {
		select {
		case <-ctx.Done():
			cleanup()
		case <-done:
		}
	}()

	return cleanup
}

// RunApp is like RunTendermint, just executes the application executable, assuming a prepared home directory
func RunApp(ctx context.Context, t TestReporter, appName string, home string) (cleanup func()) {
	t.Helper()

	appPath, err := exec.LookPath(appName)
	if err != nil {
		if os.Getenv("FORCE_TM_TEST") != "1" {
			t.Skipf("%s binary not found. Set FORCE_TM_TEST=1 to fail this test.", appName)
		} else {
			t.Fatalf("%s binary not found. Do not set FORCE_TM_TEST=1 to skip this test.", appName)
		}
	}

	cmd := exec.CommandContext(ctx, appPath, "-home", home, "start")
	// log tendermint output for verbose debugging....
	if os.Getenv("TM_DEBUG") != "" {
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("%s process failed: %s", appName, err)
	}

	// Give tendermint time to setup.
	time.Sleep(2 * time.Second)
	t.Logf("Running %s pid=%d", appPath, cmd.Process.Pid)

	// Return a cleanup function, that will wait for app to stop.
	// We also auto-kill when the context is Done
	cleanup = func() {
		t.Logf("%s cleanup called", appName)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	go func() {
		<-ctx.Done()
		cleanup()
	}()
	return cleanup
}

// SetupConfig creates a homedir to run inside,
// and copies demo tendermint files there.
//
// these files reside in sourceDir and can be created
// via `tendermint init` (sourceDir can usually be "testdata")
//
// second argument is cleanup call
func SetupConfig(t assert.Tester, sourceDir string) (string, func()) {
	rootDir, err := ioutil.TempDir("", "mock-sdk-cmd")
	assert.Nil(t, err)
	cleanup := func() { os.RemoveAll(rootDir) }

	err = copyFiles(sourceDir, rootDir, "config")
	if err != nil {
		cleanup()
		t.Fatalf("Cannot copy config files: %+v", err)
	}
	err = copyFiles(sourceDir, rootDir, "data")
	if err != nil {
		cleanup()
		t.Fatalf("Cannot copy data files: %+v", err)
	}
	return rootDir, cleanup
}

func copyFiles(sourceDir, rootDir, subDir string) error {
	// make the output dir
	outDir := filepath.Join(rootDir, subDir)
	err := os.Mkdir(outDir, 0755)
	if err != nil {
		return err
	}

	// copy everything over from testdata
	inDir := filepath.Join(sourceDir, subDir)
	files, err := ioutil.ReadDir(inDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		input := filepath.Join(inDir, f.Name())
		output := filepath.Join(outDir, f.Name())
		// t.Logf("Copying %s to %s", input, output)
		err = fileCopy(input, output, f.Mode())
		if err != nil {
			return err
		}
	}

	return nil
}

func fileCopy(input, output string, mode os.FileMode) error {
	from, err := os.Open(input)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE, mode)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	return err
}
