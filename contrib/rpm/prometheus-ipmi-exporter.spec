%define        __spec_install_post %{nil}
%define          debug_package %{nil}
%define        __os_install_post %{_dbpath}/brp-compress

Name:           prometheus-ipmi-exporter
Version:        VERSION
Release:        1.0%{?dist}
Summary:        Remote IPMI exporter for Prometheus
Group:          Monitoring

License:        The MIT License
URL:            https://github.com/prometheus-community/ipmi_exporter
Source0:        %{name}-%{version}.tar.gz

Requires(pre): shadow-utils

Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd
%{?systemd_requires}
BuildRequires:  systemd

BuildRoot:      %{_tmppath}/%{name}-%{version}-1-root

%description
Remote IPMI exporter for Prometheus

%prep
%setup -q

%build
# Empty section.

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}%{_unitdir}/
cp usr/lib/systemd/system/%{name}.service %{buildroot}%{_unitdir}/

# in builddir
cp -a * %{buildroot}

%clean
rm -rf %{buildroot}

%pre
getent group prometheus >/dev/null || groupadd -r prometheus
getent passwd prometheus >/dev/null || \
    useradd -r -g prometheus -d /dev/null -s /sbin/nologin \
    -c "Prometheus exporter user" prometheus
cp etc/sudoers.d/%{name} /etc/sudoers.d/%{name}
exit 0

%post
systemctl enable %{name}.service
systemctl start %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%defattr(-,root,root,-)
%config /etc/sysconfig/prometheus-ipmi-exporter.yml
%attr(0440, root, root) /etc/sudoers.d/prometheus-ipmi-exporter
%{_bindir}/ipmi_exporter
%{_unitdir}/%{name}.service
