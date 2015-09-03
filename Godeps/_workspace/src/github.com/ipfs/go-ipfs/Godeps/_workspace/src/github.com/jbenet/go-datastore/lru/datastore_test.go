package lru_test

import (
	"strconv"
	"testing"

	ds "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	lru "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-datastore/lru" // Hook up gocheck into the "go test" runner.
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type DSSuite struct{}

var _ = Suite(&DSSuite{})

func (ks *DSSuite) TestBasic(c *C) {
	var size = 1000

	d, err := lru.NewDatastore(size)
	c.Check(err, Equals, nil)

	for i := 0; i < size; i++ {
		err := d.Put(ds.NewKey(strconv.Itoa(i)), i)
		c.Check(err, Equals, nil)
	}

	for i := 0; i < size; i++ {
		j, err := d.Get(ds.NewKey(strconv.Itoa(i)))
		c.Check(j, Equals, i)
		c.Check(err, Equals, nil)
	}

	for i := 0; i < size; i++ {
		err := d.Put(ds.NewKey(strconv.Itoa(i+size)), i)
		c.Check(err, Equals, nil)
	}

	for i := 0; i < size; i++ {
		j, err := d.Get(ds.NewKey(strconv.Itoa(i)))
		c.Check(j, Equals, nil)
		c.Check(err, Equals, ds.ErrNotFound)
	}

	for i := 0; i < size; i++ {
		j, err := d.Get(ds.NewKey(strconv.Itoa(i + size)))
		c.Check(j, Equals, i)
		c.Check(err, Equals, nil)
	}
}
