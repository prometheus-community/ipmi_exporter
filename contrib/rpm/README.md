# Building a RPM Package

The RPM package build targets to run the exporter locally as Prometheus user with sudo permissions to expose most metrics.

For building a RPM package a build script and [Docker](https://www.docker.com/) build container are available.

NOTE:  
> The build script and the Docker build image must be executed from the project base directory!

## CentOS with rpmbuild

A Build script is located in `contrib/rpm/build.sh` to be executed on a CentOS-based host with rpmbuild tool.

The RPM package will be available under `$HOME/rpmbuild/`.

## Docker Build Container

A Docker build container is provided for CentOS7.

```bash
sudo docker build -t centos7_rpmbuild_ipmi_exporter -f contrib/rpm/docker/Dockerfile-centos7 .
sudo docker run -v $PWD/contrib/rpm/build:/outdir -it centos7_rpmbuild_ipmi_exporter
```

The RPM package will be available under `contrib/rpm/build/`.
