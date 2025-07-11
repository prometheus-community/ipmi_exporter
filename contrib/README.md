# Building a RPM Package

You can build package with Docker in [mock container](//github.com/mmornati/docker-mock-rpmbuilder)

```console
docker run --rm --privileged --volume="${PWD}/contrib:/rpmbuild" -e MOUNT_POINT="/rpmbuild" -e MOCK_CONFIG="centos-stream-9-aarch64" -e SPEC_FILE="prometheus-ipmi-exporter.spec" -e NETWORK="true" mock ./build-rpm.sh
```
