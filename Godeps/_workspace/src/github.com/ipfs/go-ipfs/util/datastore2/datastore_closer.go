package datastore2

import (
	"io"

	"github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-datastore"
)

type ThreadSafeDatastoreCloser interface {
	datastore.ThreadSafeDatastore
	io.Closer

	Batch() (datastore.Batch, error)
}

func CloserWrap(ds datastore.ThreadSafeDatastore) ThreadSafeDatastoreCloser {
	return &datastoreCloserWrapper{ds}
}

type datastoreCloserWrapper struct {
	datastore.ThreadSafeDatastore
}

func (w *datastoreCloserWrapper) Close() error {
	return nil // no-op
}

func (w *datastoreCloserWrapper) Batch() (datastore.Batch, error) {
	bds, ok := w.ThreadSafeDatastore.(datastore.BatchingDatastore)
	if !ok {
		return nil, datastore.ErrBatchUnsupported
	}

	return bds.Batch()
}
