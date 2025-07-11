%global debug_package %{nil}
%define _build_id_links none
Summary: Export IPMI metrics to Prometheus local or remotly
Name: ipmi_exporter
%define _name prometheus-ipmi-exporter
%define _name2 ipmi-exporter
Version: 1.10.1
Release: 1%{?dist}
License: MIT
Requires: freeipmi
BuildRequires: git systemd-rpm-macros
BuildRequires: golang >= 1.23.0
%define _uri github.com/prometheus-community
%define _archversion db67a63b0fe42c5d7551145b566669f5960e02ed
%define _archuri https://gitlab.archlinux.org/archlinux/packaging/packages/%{_name}/-/raw/%{_archversion}
Url: https://%{_uri}/%{name}
Source0: %{url}/archive/refs/tags/v%{version}.tar.gz
Source1: %{_archuri}/sudoers.conf
Source2: %{_archuri}/sysusers.conf
Source3: %{_archuri}/systemd.service
Source4: %{_archuri}/config.yml
Source5: %{_archuri}/config.env

%description
Prometheus IPMI Exporter

%prep
%setup -c -q
export GOPATH="%{_builddir}/gopath"
export GOBIN="${GOPATH}/bin"
mkdir -p "${GOPATH}/src/%{_uri}"
ln -snf "%{_builddir}/%{name}-%{version}/%{name}-%{version}" \
"${GOPATH}/src/%{_uri}/%{name}"

%build
export GOPATH="%{_builddir}/gopath"
export GOBIN="${GOPATH}/bin"
cd "${GOPATH}/src/%{_uri}/%{name}"

eval "$(go env | grep -e "GOHOSTOS" -e "GOHOSTARCH")"
GOOS="${GOHOSTOS}" GOARCH="${GOHOSTARCH}" BUILDTAGS="netgo static_build" \
  go build -x \
  -buildmode="pie" \
  -trimpath \
  -mod="readonly" \
  -modcacherw \
  -ldflags "-linkmode external \
  -X github.com/prometheus/common/version.Version=%{version} \
  -X github.com/prometheus/common/version.Revision=%{release} \
  -X github.com/prometheus/common/version.Branch=tarball \
  -X github.com/prometheus/common/version.BuildUser=$(whoami)@mockbuild \
  -X github.com/prometheus/common/version.BuildDate=$(date +%%Y%%m%%d)"

%install
# remove arch-specific package name
%{__sed} -i \
  -e 's|'-/etc/prometheus/"%{_name2}".env'|'-/etc/conf.d/"%{name}"'|g' \
  -e 's|'/usr/bin/"%{_name}"'|'/usr/bin/"%{name}"'|g' \
  -e 's|'"%{_name2}".yml'|'"%{name}".yml'|g' \
  "%SOURCE3"

%{__sed} -i \
  -e 's|'"%{_bindir}"'|'"%{_sbindir}"'|g' "%SOURCE1"

install -Dm0755 "%{name}-%{version}/%{name}" -t "%{buildroot}%{_bindir}"
install -Dm0644 "%SOURCE1" "%{buildroot}%{_sysconfdir}/sudoers.d/%{_name}"
install -Dm0644 "%SOURCE2" "%{buildroot}%{_sysusersdir}/%{name}.conf"
install -Dm0644 "%SOURCE3" "%{buildroot}%{_unitdir}/%{name}.service"
install -Dm0644 "%SOURCE4" "%{buildroot}%{_sysconfdir}/prometheus/%{name}.yml"
install -Dm0644 "%SOURCE5" "%{buildroot}%{_sysconfdir}/conf.d/%{name}"

%pre
%sysusers_create_package %{name} "%SOURCE2"

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%license %{name}-%{version}/LICENSE
%attr(0755,root,root) %{_bindir}/%{name}
%defattr(0644,root,root,0755)
%config(noreplace) %{_sysconfdir}/prometheus/%{name}.yml
%config(noreplace) %{_sysconfdir}/conf.d/%{name}
%{_sysconfdir}/sudoers.d/%{_name}
%{_sysusersdir}/%{name}.conf
%{_unitdir}/%{name}.service
