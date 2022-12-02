#!/bin/bash
set -e

function do_checks {

  info_msg="The build script must be executed from the projects base directory!"

  if [ -z "$VERSION" ]; then
    echo "ERROR: Build failed! VERSION file not found" >&2
    echo "INFO: $info_msg"
    exit 1
  fi
  
  if [ ! -d "$CONTRIB_DIR" ]; then
    echo "ERROR: Build failed! Directory does not exist: $CONTRIB_DIR" >&2
    echo "INFO: $info_msg"
    exit 1
  fi

}

export VERSION=$(cat VERSION)
export BUILD_DIR=$HOME/rpmbuild
export CONTRIB_DIR="contrib/rpm"
export PACKAGE_DIR=prometheus-ipmi-exporter-$VERSION

do_checks

make build

mkdir -p $BUILD_DIR/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
mkdir -p $BUILD_DIR/SOURCES/$PACKAGE_DIR/usr/bin
mkdir -p $BUILD_DIR/SOURCES/$PACKAGE_DIR/usr/lib/systemd/system
mkdir -p $BUILD_DIR/SOURCES/$PACKAGE_DIR/etc/sysconfig
mkdir -p $BUILD_DIR/SOURCES/$PACKAGE_DIR/etc/sudoers.d

sed "s/VERSION/$VERSION/" $CONTRIB_DIR/prometheus-ipmi-exporter.spec > $BUILD_DIR/SPECS/prometheus-ipmi-exporter.spec

cp $CONTRIB_DIR/systemd/prometheus-ipmi-exporter.service $BUILD_DIR/SOURCES/$PACKAGE_DIR/usr/lib/systemd/system/
cp $CONTRIB_DIR/sudoers/prometheus-ipmi-exporter $BUILD_DIR/SOURCES/$PACKAGE_DIR/etc/sudoers.d/
cp $CONTRIB_DIR/config/prometheus-ipmi-exporter.yml $BUILD_DIR/SOURCES/$PACKAGE_DIR/etc/sysconfig/
cp ipmi_exporter $BUILD_DIR/SOURCES/$PACKAGE_DIR/usr/bin/

cd $BUILD_DIR/SOURCES
tar -czvf $PACKAGE_DIR.tar.gz $PACKAGE_DIR
cd $BUILD_DIR
echo Build dir is: $BUILD_DIR
ls -la $BUILD_DIR/SOURCES
rpmbuild -ba $BUILD_DIR/SPECS/prometheus-ipmi-exporter.spec
