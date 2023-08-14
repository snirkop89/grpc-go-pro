package main

import (
	"bytes"
	"compress/gzip"
	"log"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func compressedSize[M protoreflect.ProtoMessage](msg M) (int, int) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	out, err := proto.Marshal(msg)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := gz.Write(out); err != nil {
		log.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		log.Fatal(err)
	}
	return len(out), len(b.Bytes())
}
