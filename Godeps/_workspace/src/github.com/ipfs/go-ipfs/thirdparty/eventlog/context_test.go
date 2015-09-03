package eventlog

import (
	"testing"

	"github.com/ipfs/go-ipfs/Godeps/_workspace/src/golang.org/x/net/context"
)

func TestContextContainsMetadata(t *testing.T) {
	t.Parallel()

	m := Metadata{"foo": "bar"}
	ctx := ContextWithLoggable(context.Background(), m)
	got, err := MetadataFromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, exists := got["foo"]
	if !exists {
		t.Fail()
	}
}

func TestContextWithPreexistingMetadata(t *testing.T) {
	t.Parallel()

	ctx := ContextWithLoggable(context.Background(), Metadata{"hello": "world"})
	ctx = ContextWithLoggable(ctx, Metadata{"goodbye": "earth"})

	got, err := MetadataFromContext(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, exists := got["hello"]
	if !exists {
		t.Fatal("original key not present")
	}
	_, exists = got["goodbye"]
	if !exists {
		t.Fatal("new key not present")
	}
}
