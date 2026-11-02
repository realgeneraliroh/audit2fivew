Name:           audit2fivew
Version:        1.0.0
Release:        1%{?dist}
Summary:        Terminal UI for SELinux AVC denial troubleshooting

# Base application: AGPL-3.0-or-later
# Vendored dependencies:
# - MIT: Bubble Tea, Lipgloss, Bubbles, muesli, uniseg, etc.
# - BSD-3-Clause: modernc.org (libc, mathutil, memory), google/uuid, golang.org/x/sys
# - blessing: SQLite dedication in modernc.org/sqlite
License:        AGPL-3.0-or-later AND MIT AND BSD-3-Clause AND blessing
URL:            https://github.com/realgeneraliroh/audit2fivew
Source0:        https://github.com/realgeneraliroh/%{name}/archive/v%{version}/%{name}-%{version}.tar.gz
# Companion vendor archive containing vendored Go dependencies for offline builds
Source1:        https://github.com/realgeneraliroh/%{name}/releases/download/v%{version}/%{name}-%{version}-vendor.tar.gz

# Go toolchain — EPEL ships golang ≥ 1.21 on EL9.
BuildRequires:  golang >= 1.21
BuildRequires:  make

# audit is needed at runtime for ausearch(8).
Requires:       audit

# Helpful but not mandatory.
Recommends:     policycoreutils-python-utils
Recommends:     setools-console

# Pure Go — no debuginfo, no library deps.
%global debug_package %{nil}

%description
audit2fivew is a non-destructive, fully-offline Terminal UI (TUI)
utility for SELinux troubleshooting.  It parses AVC denials from the
audit log, stores them in a local SQLite database for historical
tracking, and presents them in a readable master-detail view that shows:

  • When the denial occurred
  • Who (source context and process)
  • Where (target context and file path)
  • What (class and permissions denied)
  • Way to Fix (auto-generated policy rules and module loading commands)

%prep
%setup -q
%setup -q -T -D -a 1

%build
# Vendor directory is unpacked by %%setup -a 1 — no network needed.
export GOFLAGS="-mod=vendor"
export CGO_ENABLED=0
export GOOS=linux
%if "%{_target_cpu}" == "aarch64"
export GOARCH=arm64
%else
export GOARCH=amd64
%endif
make build VERSION=%{version}

%check
export GOFLAGS="-mod=vendor"
export CGO_ENABLED=0
go test -v ./...

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_mandir}/man1
install -Dpm 0755 %{name} %{buildroot}%{_bindir}/%{name}
install -Dpm 0644 %{name}.1 %{buildroot}%{_mandir}/man1/%{name}.1

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%{_mandir}/man1/%{name}.1*

%changelog
* Mon Nov 02 2026 J <maintainers@audit2fivew.org> - 1.0.0-1
- Initial release for EPEL
- Add manual page audit2fivew(1)
- Add companion vendor archive for offline hermetic builds
- Update SPDX license tag to include vendored dependencies

