# tc-v5

To generate protobuf:

* Install `buf` tool. See https://buf.build/docs/cli/installation/
* Run: `buf generate`

To run server:

* Run: `go run ./cmd/inbd`

This will wait for a client connection on `/tmp/inbd.sock`.

To run client:

* Run: `go run ./cmd/inbc`

This will connect to the server `inbd` via `/tmp/inbd.sock` and send a simple gRPC query.
