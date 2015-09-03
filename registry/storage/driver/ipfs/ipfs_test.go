package ipfs

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	storagedriver "github.com/docker/distribution/registry/storage/driver"
	"github.com/docker/distribution/registry/storage/driver/testsuites"
	. "gopkg.in/check.v1"

	"github.com/docker/distribution/context"

	u "github.com/ipfs/go-ipfs/util"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

var testAddr, testRoot string
var skipCheck func() string

func init() {
	testAddr = os.Getenv("IPFS_ADDR")
	testRoot = os.Getenv("IPFS_ROOT")

	ipfsDriverConstructor := func() (storagedriver.StorageDriver, error) {
		return New(testAddr, testRoot), nil
	}

	// Skip ipfs storage driver tests if environment variable parameters are not provided
	skipCheck = func() string {
		if testAddr == "" || testRoot == "" {
			return fmt.Sprintf("Must set IPFS_ADDR and IPFS_ROOT environment variables to run IPFS tests")
		}
		return ""
	}

	// BUG(stevvooe): IPC is broken so we're disabling for now. Will revisit later.
	// testsuites.RegisterIPCSuite(driverName, map[string]string{"rootdirectory": root}, skipCheck)
	testsuites.RegisterSuite(ipfsDriverConstructor, skipCheck)
}

// TODO this is here until testing with the storage testsuite is fixed. To test run:
// go test -test.short -run 'TestBasic' github.com/docker/distribution/registry/storage/driver/ipfs
func TestBasic(t *testing.T) {
	if skipCheck() != "" {
		t.Skip(skipCheck())
	}

	d := New(testAddr, testRoot)

	err := d.PutContent(context.Background(), "/a/b/c", []byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}

	out, err := d.GetContent(context.Background(), "/a/b/c")
	if err != nil {
		t.Fatal(err)
	}

	if string(out) != "hello world" {
		t.Fatal("wrong data")
	}

	err = d.Delete(context.Background(), "/a/b/c")
	if err != nil {
		t.Fatal(err)
	}

	out, err = d.GetContent(context.Background(), "/a/b/c")
	if err == nil {
		t.Fatal("expected not found")
	}
	if _, ok := err.(storagedriver.PathNotFoundError); !ok {
		t.Fatal("expected path not found error but got: ", err)
	}

	// alright, that stuff works, lets turn it up a notch

	b := int64(10000000)
	buf := make([]byte, b)
	u.NewTimeSeededRand().Read(buf)
	r := bytes.NewReader(buf)

	coolpath := "/this/is/a/cool/path"
	nn, err := d.WriteStream(context.Background(), coolpath, 0, r)
	if err != nil {
		t.Fatal(err)
	}
	if nn != b {
		t.Fatal("didnt write enough bytes", nn, b)
	}

	dataout, err := d.GetContent(context.Background(), coolpath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(dataout, buf) {
		t.Fatal("data wasnt right!")
	}

	// now lets *move* that data, yeah, difficult stuff here
	coolerpath := "/slightly/cooler/path"
	err = d.Move(context.Background(), coolpath, coolerpath)
	if err != nil {
		t.Fatal(err)
	}

	dataout, err = d.GetContent(context.Background(), coolerpath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(dataout, buf) {
		t.Fatal("data wasnt right!")
	}
}
