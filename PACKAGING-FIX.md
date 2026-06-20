# Packaging fix — CRDs missing from the published package

## Symptom

Installing the published provider (`ghcr.io/rossigee/provider-harbor:v0.16.1`)
on a Crossplane v2 cluster: the `ProviderRevision` goes **Healthy/Installed=True**
but registers **0 managed-resource CRDs**. Applying any MR fails:

```
no matches for kind "Project" in version "project.harbor.m.crossplane.io/v1beta1"
ensure CRDs are installed first
```

Confirmed by extracting the published OCI artifact: the `package.yaml` Crossplane
reads contains **only** the `meta.pkg.crossplane.io/v1 Provider` doc — **zero**
`CustomResourceDefinition` docs. The 15 CRDs are present in the image only as
loose files under `package/crds/`, which the Crossplane package runtime does not
read.

## Root cause

The stock `.github/workflows/release.yml` runs `make publish` with both:

- `REGISTRY_ORGS=ghcr.io/<owner>` → the **runtime controller Docker image**, and
- `XPKG_REG_ORGS=ghcr.io/<owner>` → the **Crossplane package (xpkg)**

…pointing at the **same** ref `ghcr.io/<owner>/provider-harbor:<tag>`. The two
artifacts collide on that single tag and the plain runtime image (built by
`cluster/images/.../Dockerfile`, which does `COPY package/ /package/` — meta-only
`package.yaml` + loose `crds/`) overwrites the real xpkg. Consumers pull the
runtime image, so Crossplane sees a package with no CRDs.

(`crossplane xpkg build` itself is fine — it recurses `package/` and merges the
CRDs into `package.yaml`. The bug is the publish step clobbering it.)

## Fix

`.github/workflows/release-xpkg-fixed.yml` packages the right way. For **each
architecture** (`linux/amd64` and `linux/arm64`; the set is the `PLATFORMS` env
var):

1. build the provider binary,
2. build the runtime controller image **locally** (loaded into the Docker
   daemon, **not** pushed to the provider tag; QEMU enables the cross-arch
   build),
3. `crossplane xpkg build --package-root=package --embed-runtime-image=<local>`
   — embeds the runtime and merges all CRDs into `package.yaml`,
4. **verify** `package.yaml` contains the Provider meta **and** all
   `package/crds/*.yaml` CRDs (hard gate — fails the run otherwise),

then push all per-arch xpkgs as a **single multi-platform package index**
(`crossplane xpkg push -f amd64.xpkg,arm64.xpkg <ref>`) to
`ghcr.io/<owner>/provider-harbor:<tag>` — Crossplane selects the matching arch
per node.

No second artifact is pushed to the provider tag, so nothing can clobber the
package. This is the **canonical** release path: it triggers on any `v*` tag
(or `workflow_dispatch`), and the stock `release.yml` has been disabled so it
cannot race this workflow on the same tag.

Scope: this fixes **packaging only**.
The stubbed `Robot`/`Member` controllers (client type-switch omits them; `Project`
ID is mocked) are a separate, larger code fix.
