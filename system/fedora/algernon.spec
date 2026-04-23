Name:           algernon
Version:        1.17.5
Release:        1%{?dist}
Summary:        Web server with Lua, Markdown, QUIC, Redis and PostgreSQL support

License:        BSD-3-Clause
URL:            https://algernon.roboticoverlords.org/
Source0:        https://github.com/xyproto/%{name}/archive/v%{version}/%{name}-%{version}.tar.gz

ExclusiveArch:  x86_64

# Go strips symbols via -ldflags "-s -w" below, so there is nothing for the
# debug subpackage to contain.
%global debug_package %{nil}

BuildRequires:  golang >= 1.25
BuildRequires:  systemd-rpm-macros

Requires(post):    systemd
Requires(preun):   systemd
Requires(postun):  systemd

Recommends:     valkey
Suggests:       mariadb
Suggests:       ollama
Suggests:       postgresql-server

%description
Algernon is a small, self-contained web server with built-in support for
QUIC, HTTP/2, Lua, Teal, Markdown, Pongo2, HyperApp, Amber, Sass (SCSS),
GCSS, JSX, Ollama (LLMs), BoltDB, Redis, Valkey, PostgreSQL, SQLite,
MariaDB, MySQL, MSSQL, IPv6, rate limiting, graceful shutdown, plugins,
users and permissions.

%prep
%autosetup -n %{name}-%{version}

%build
# Build flags mirror the Arch PKGBUILD: PIE, trimpath, vendored deps, no VCS stamping
export CGO_ENABLED=1
export GOFLAGS="-mod=vendor -trimpath -buildmode=pie -buildvcs=false"
go build \
    -ldflags="-s -w -linkmode=external -extldflags '%{__global_ldflags}'" \
    -o %{name} .

%install
install -Dm0755 %{name}                                    %{buildroot}%{_bindir}/%{name}
install -Dm0755 desktop/mdview                             %{buildroot}%{_bindir}/mdview

install -Dm0644 system/algernon.service                    %{buildroot}%{_unitdir}/%{name}.service
install -Dm0644 system/algernon_dev.service                %{buildroot}%{_docdir}/%{name}/algernon_dev.service.example
install -Dm0644 system/logrotate                           %{buildroot}%{_sysconfdir}/logrotate.d/%{name}
install -Dm0644 system/serverconf.lua                      %{buildroot}%{_sysconfdir}/%{name}/serverconf.lua

install -Dm0644 desktop/algernon.desktop                   %{buildroot}%{_datadir}/applications/algernon.desktop
install -Dm0644 desktop/algernon_md.desktop                %{buildroot}%{_datadir}/applications/algernon_md.desktop
install -Dm0644 desktop/markdown.png                       %{buildroot}%{_datadir}/pixmaps/markdown.png

install -Dm0644 algernon.1                                 %{buildroot}%{_mandir}/man1/algernon.1

install -d %{buildroot}%{_docdir}/%{name}/samples
cp -a samples/. %{buildroot}%{_docdir}/%{name}/samples/

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%license LICENSE
%doc README.md TUTORIAL.md ChangeLog.md
%{_bindir}/%{name}
%{_bindir}/mdview
%{_unitdir}/%{name}.service
%config(noreplace) %{_sysconfdir}/logrotate.d/%{name}
%dir %{_sysconfdir}/%{name}
%config(noreplace) %{_sysconfdir}/%{name}/serverconf.lua
%{_datadir}/applications/algernon.desktop
%{_datadir}/applications/algernon_md.desktop
%{_datadir}/pixmaps/markdown.png
%{_mandir}/man1/algernon.1*
%{_docdir}/%{name}/algernon_dev.service.example
%{_docdir}/%{name}/samples

%changelog
* Thu Apr 23 2026 Alexander F. Rødseth <xyproto@archlinux.org> - 1.17.5-1
- Initial Fedora spec, covering Fedora 43 and Fedora 44 on x86_64.
