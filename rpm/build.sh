#!/bin/bash
set -e
export VERSION=$(cat VERSION)
export promdir=prometheus-ipmi-exporter-$VERSION
export builddir=$HOME/rpmbuild
make build
sed -i "s/VERSION/$(cat VERSION)/" rpm/prometheus-ipmi-exporter.spec
mkdir -p $builddir/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
mkdir -p $builddir/SOURCES/$promdir/usr/bin
mkdir -p $builddir/SOURCES/$promdir/usr/lib/systemd/system
mkdir -p $builddir/SOURCES/$promdir/etc/sysconfig
mkdir -p $builddir/SOURCES/$promdir/etc/sudoers.d
cp rpm/prometheus-ipmi-exporter.spec $builddir/SPECS/
cp systemd/prometheus-ipmi-exporter.service $builddir/SOURCES/$promdir/usr/lib/systemd/system/
cp sudoers/prometheus-ipmi-exporter $builddir/SOURCES/$promdir/etc/sudoers.d/
cp prometheus-ipmi-exporter.yml $builddir/SOURCES/$promdir/etc/sysconfig/
cp ipmi_exporter $builddir/SOURCES/$promdir/usr/bin/
cd $builddir/SOURCES
tar -czvf $promdir.tar.gz $promdir
cd $builddir
echo build dir is $builddir
ls -la $builddir/SOURCES
rpmbuild -ba  $builddir/SPECS/prometheus-ipmi-exporter.spec
