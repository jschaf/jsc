+++
slug = "prune-unused-npm-deps-with-pnpm"
date = 2025-07-29
visibility = "published"
+++

# Prune unused npm dependencies with pnpm

pnpm, a npm replacement, follows the pattern of npm and downloads all optional
dependencies. This is normally fine, but as more packages use npm as a binary
distribution platform, the number of optional dependencies grows. For example,
`esbuild` is a Go binary distributed via npm. To work, esbuild lists 24
supported operating systems and architectures as optional dependencies.

```json
{
  "name": "esbuild",
  "version": "0.25.5",
  "optionalDependencies": {
    "@esbuild/aix-ppc64": "0.25.5",
    "@esbuild/android-arm": "0.25.5",
    "@esbuild/android-arm64": "0.25.5",
    "@esbuild/android-x64": "0.25.5",
    "@esbuild/darwin-arm64": "0.25.5",
    "@esbuild/darwin-x64": "0.25.5",
    "@esbuild/freebsd-arm64": "0.25.5",
    "@esbuild/freebsd-x64": "0.25.5",
    "@esbuild/linux-arm": "0.25.5",
    "@esbuild/linux-arm64": "0.25.5",
    "@esbuild/linux-ia32": "0.25.5",
    "@esbuild/linux-loong64": "0.25.5",
    "@esbuild/linux-mips64el": "0.25.5",
    "@esbuild/linux-ppc64": "0.25.5",
    "@esbuild/linux-riscv64": "0.25.5",
    "@esbuild/linux-s390x": "0.25.5",
    "@esbuild/linux-x64": "0.25.5",
    "@esbuild/netbsd-arm64": "0.25.5",
    "@esbuild/netbsd-x64": "0.25.5",
    "@esbuild/openbsd-arm64": "0.25.5",
    "@esbuild/openbsd-x64": "0.25.5",
    "@esbuild/sunos-x64": "0.25.5",
    "@esbuild/win32-arm64": "0.25.5",
    "@esbuild/win32-ia32": "0.25.5",
    "@esbuild/win32-x64": "0.25.5"
  }
}
```

The esbuild binary is about 10 MiB, so there's about 250 MiB of optional
dependencies. 

Most engineering orgs only care about a few operating systems, typically Linux
and Darwin, with both x64 and arm64 architectures. Luckily, pnpm provides
hooks to prune unused optional dependencies. The `prune` hook runs after
