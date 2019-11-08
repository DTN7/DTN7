package bundle

import (
	"bytes"
	"reflect"
	"testing"
)

func TestBundleBuilderSimple(t *testing.T) {
	bndl, err := Builder().
		CRC(CRC32).
		Source("dtn://myself/").
		Destination("dtn://dest/").
		CreationTimestampEpoch().
		Lifetime("10m").
		HopCountBlock(64).
		BundleAgeBlock(0).
		PayloadBlock([]byte("hello world!")).
		Build()

	if err != nil {
		t.Fatalf("Builder errored: %v", err)
	}

	buff := new(bytes.Buffer)
	if err := bndl.MarshalCbor(buff); err != nil {
		t.Fatal(err)
	}
	bndlCbor := buff.Bytes()

	bndl2 := Bundle{}
	if err = bndl2.UnmarshalCbor(buff); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(bndl, bndl2) {
		t.Fatalf("Bundle changed after serialization: %v, %v", bndl, bndl2)
	}

	bndl3, err := NewBundle(
		NewPrimaryBlock(
			0,
			MustNewEndpointID("dtn://dest/"),
			MustNewEndpointID("dtn://myself/"),
			NewCreationTimestamp(DtnTimeEpoch, 0),
			1000000*60*10),
		[]CanonicalBlock{
			NewCanonicalBlock(2, 0, NewHopCountBlock(64)),
			NewCanonicalBlock(3, 0, NewBundleAgeBlock(0)),
			NewCanonicalBlock(1, 0, NewPayloadBlock([]byte("hello world!")))})
	if err != nil {
		t.Fatal(err)
	}

	buff.Reset()
	bndl3.PrimaryBlock.ReportTo = bndl3.PrimaryBlock.SourceNode
	bndl3.SetCRCType(CRC32)

	if err := bndl3.MarshalCbor(buff); err != nil {
		t.Fatal(err)
	}
	bndl3Cbor := buff.Bytes()

	if !bytes.Equal(bndlCbor, bndl3Cbor) {
		t.Fatalf("CBOR has changed:\n%x\n%x", bndlCbor, bndl3Cbor)
	}

	if !reflect.DeepEqual(bndl, bndl3) {
		t.Fatalf("Bundles differ: %v, %v", bndl, bndl3)
	}
}

func TestBldrParseEndpoint(t *testing.T) {
	eidIn, _ := NewEndpointID("dtn://foo/bar/")
	if eidTmp, _ := bldrParseEndpoint(eidIn); eidTmp != eidIn {
		t.Fatalf("Endpoint does not match: %v != %v", eidTmp, eidIn)
	}

	if eidTmp, _ := bldrParseEndpoint("dtn://foo/bar/"); eidTmp != eidIn {
		t.Fatalf("Parsed endpoint does not match: %v != %v", eidTmp, eidIn)
	}

	if _, errTmp := bldrParseEndpoint(23.42); errTmp == nil {
		t.Fatalf("Invalid endpoint type does not resulted in an error")
	}
}

func TestBldrParseLifetime(t *testing.T) {
	tests := []struct {
		val interface{}
		us  uint64
		err bool
	}{
		{1000, 1000, false},
		{uint64(1000), 1000, false},
		{"1000µs", 1000, false},
		{"1000us", 1000, false},
		{-23, 0, true},
		{"-10m", 0, true},
		{true, 0, true},
	}

	for _, test := range tests {
		us, err := bldrParseLifetime(test.val)

		if test.err == (err == nil) {
			t.Fatalf("Error value for %v was unexpected: %v != %v",
				test.val, test.err, err)
		}

		if test.us != us {
			t.Fatalf("Value for %v was unexpected: %v != %v", test.val, test.us, us)
		}
	}
}
