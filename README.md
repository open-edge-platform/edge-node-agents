# frameworks.edge.one-intel-edge.maestro-infra.inbm

Prerequisite for development: install Earthly. See https://earthly.dev/get-earthly. 

To run lint checks:

```bash
earthly +lint
```

To build:

```bash
`earthly +build`
```

To run tests: (Includes +lint)

```bash
earthly +test
```

To run server:

```bash
`sudo ./build/inbd -s /tmp/inbd.sock`
```

This will wait for a client connection on `/tmp/inbd.sock`.

To run client:

```bash
sudo ./build/inbc --socket /tmp/inbd.sock sota --mode full --reboot=false
```

This will connect to the server `inbd` via `/tmp/inbd.sock` and send a simple gRPC query.

To run server as a service:

* Build `inbd` -- see above.
* Copy the binary to /usr/bin: `cp ./build/inbd /usr/bin`
* Copy the service file to systemd folder: `cp configs/systemd/inbd.service /lib/systemd/system/inbd.service`
* Start the service: `systemctl start inbd`
* Check the service's log: `journalctl -fu inbd`
* To start inbd automatically on system boot: `systemctl enable inbd`

## BUILD INSTALL PACKAGE INSTRUCTIONS

### How to build

```shell
./build.sh
```

### Build output

* When build is complete, build output will be in the `dist` folder.
* See `dist/README.txt` for a description of the build output.

## Note
This Framework manages TLS certificate generation for secure communication between inbd (INBM Daemon) and inbc (INBM Client). By default, it generates a local Certificate Authority (CA) and uses it to sign certificates for both inbd and inbc, enabling mutual TLS authentication for development and testing.
### Security Recommendation:
For production deployments, do NOT use self-signed or locally generated certificates. Always obtain certificates from a trusted Certificate Authority (CA) to ensure proper security, trust, and interoperability.