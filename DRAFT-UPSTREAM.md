# Draft upstream proposal: host-managed IMEX mode for GB200/GB300

This file is a working draft that lives only on this fork branch. It is not
meant to be merged. CONTRIBUTING.md asks for an acknowledged issue before a
pull request, so it holds (1) the issue to file first, (2) the security and
isolation trade-off, and (3) the pull request description to use once the
issue is acknowledged.

---

## 1. Draft issue (feature request template)

**Title:** `feat: support host-managed IMEX on GB200/GB300 (no ComputeDomain when the node's IMEX daemon is run by the host)`

**Component:** Catalog / Workloads (also touches Controller / Reconcilers and CLI (nvcrectl))

### Feature Summary

Add an opt-in "host IMEX" mode to the GB200/GB300 catalog entries. When every
target node carries the label `nvcre.nvidia.com/imex-mode=host`, the catalog
creates no ComputeDomain and no IMEX channel ResourceClaimTemplate. The workload
pods instead get the host's IMEX channels through CDI.

### Problem / Use Case

Some GB200/GB300 racks run the IMEX daemon on the host: the `nvidia-imex`
systemd service installed with the driver, managed by whoever runs the
nodes. The NVIDIA DRA driver's ComputeDomain controller is not deployed there,
or its IMEX daemons are deliberately not used, because the host already owns
IMEX on those nodes.

The NVCRE catalog always renders a ComputeDomain plus a channel
ResourceClaimTemplate for GB200/GB300 entries, including single-node entries
such as `nccl-loopback` and single-tray jobs where `numNodes` is 1. On a host
managed rack nothing ever satisfies that claim, so the pods stay Pending
forever. A certification cannot run there at all, even one that needs only a
single tray.

### Proposed Solution

- A node label, `nvcre.nvidia.com/imex-mode=host`, set by the cluster
  operator.
- `platform.DetectIMEXMode(nodes)` returns `host` only when there is at least
  one node and **every** node in the set carries the label. An unlabelled or
  mixed set keeps today's ComputeDomain path exactly as it is, so the mode is
  opt-in and nothing changes for existing users.
- `BuildConfig` / `TemplateData` / `platform.OverrideConfig` carry an
  `IMEXMode` field. The certification controller detects it on the
  arch-filtered node set that sizes the job; the WorkloadRun controller detects
  it on its discovered nodes; `nvcrectl certification render --dry-run`
  detects it on the nodes it reads, so the preview matches what the controller
  creates. Offline render has no nodes and keeps the ComputeDomain path.
- In host mode the shared GB200/GB300 dependency fragments
  (`gb200-compute-domain-and-dra-{comm,torch}.yaml`, and the AWS GB300
  `gb300-roce-{comm,torch}.yaml` that re-declare the channel claim next to the
  RoCE claim) render no ComputeDomain and no channel claim. The pods get
  `runtimeClassName: nvidia` and the pod annotation
  `cdi.k8s.io/imex: nvidia.com/imex-channel=all`, so the NVIDIA container
  runtime injects the host IMEX channel devices through CDI.
- The host `nvidia-imex` service is used as-is; NVCRE never stops or masks it.

### Evidence

A prototype of this change on the v0.9.0 release (the same catalog and
controller edits, before the rebase onto `main`) was run live on two GB300
NVL72 racks whose IMEX daemon is host-managed, 1 tray per job, 4 GPUs per
tray. Every run created no ComputeDomain, the pods started and every job ran
to completion and reported its bandwidth.

| Run | Rack / trays            | all-gather (floor >= 532 GB/s) | all-reduce (floor >= 706 GB/s) |
|-----|-------------------------|--------------------------------|--------------------------------|
| 1   | rack A, 2 trays         | 649.3 GB/s, Succeeded          | 688.0 GB/s, Failed (below floor) |
| 2   | rack A, 2 trays         | 647.7 GB/s, Succeeded          | 688.8 GB/s, Failed (below floor) |
| 3   | rack B (idle), 1 tray   | 649.74 GB/s, Succeeded         | 694.75 GB/s, Failed (below floor) |

The floors are the operator's own calibration, set in the Certification, not
catalog defaults. The all-reduce "Failed" means only that the measured
bandwidth did not reach that floor; each run itself completed normally and
produced its measurement, which is what this issue is about. Before the change,
the same certifications never started because the IMEX channel claim was never
allocated.

---

## 2. Security and isolation trade-off

State this plainly in both the issue and the PR:

- `nvidia.com/imex-channel=all` exposes **all** IMEX channels on the node to
  the pod. A ComputeDomain scopes one IMEX channel to one workload. Host mode
  therefore gives up per-workload IMEX isolation: any pod admitted with that
  annotation and runtime class can reach every channel the host IMEX daemon
  serves on that node.
- Host mode is intended for single-tenant nodes, or for certification run by
  the cluster operator while the nodes are not shared with other tenants. It is
  not a multi-tenant mode.
- It must not be mixed with the DRA driver's ComputeDomain IMEX daemons on the
  same nodes. The two would contend for the same IMEX domain. The all-or-nothing
  label rule keeps a single node set on one path, but operators are
  responsible for not labelling nodes that the DRA ComputeDomain also manages.
- No `hostNetwork`, `hostIPC`, `privileged`, added capabilities or hostPath
  mounts are used. The only extra access is the IMEX channel device nodes the
  CDI spec injects.
- The mode is opt-in through a label an operator has to set; unlabelled
  clusters are unchanged.

---

## 3. Draft pull request description

**Title:** `feat(catalog): host-managed IMEX mode for GB200/GB300`

### Summary

Adds an opt-in host-managed IMEX mode. When every target node is labelled
`nvcre.nvidia.com/imex-mode=host`, GB200/GB300 catalog entries render no
ComputeDomain and no IMEX channel ResourceClaimTemplate; pods instead receive
the host IMEX channels through CDI (`cdi.k8s.io/imex:
nvidia.com/imex-channel=all`, `runtimeClassName: nvidia`). Unlabelled or mixed
node sets keep the ComputeDomain path unchanged.

- `pkg/platform/imex.go`: `IMEXModeLabel`, `IMEXModeHost`,
  `DetectIMEXMode` (all-or-nothing).
- `pkg/catalog`: `BuildConfig.IMEXMode` and `TemplateData.IMEXMode`; the
  GB200/GB300 dependency fragments switch on `.IMEXMode`.
- `pkg/platform/overrides.go`: `OverrideConfig.IMEXMode` for WorkloadRun.
- `pkg/controller`: the certification controller detects the mode on the
  arch-filtered node set; the WorkloadRun controller on its discovered nodes.
- `pkg/certification`: `nvcrectl certification render --dry-run` detects the
  mode from the nodes it reads. `renderCertification` keeps its signature and
  delegates with an empty mode, so existing render goldens are unchanged.

See "Security and isolation trade-off" in the issue: host mode exposes all IMEX
channels on the node to the pod and is meant for single-tenant or
operator-run certification.

### Related Issue

Closes #<issue>

### Type of Change

- [x] New feature

### Component(s) Affected

- [x] Controller / Reconcilers
- [x] Catalog / Workloads
- [x] CLI (nvcrectl)

### Testing

- Unit tests: `pkg/platform/imex_test.go` (label detection: none, all,
  mixed, empty), `pkg/catalog/imex_test.go` (one-node and multi-node GB300
  entries in both modes), `pkg/certification/host_imex_test.go` (GB300
  certification render with `nccl-all-reduce` on 2 nodes, `nccl-loopback` and
  `nemotron5-8b` on 1, in both modes), WorkloadRun override tests for torch and
  mpi.
- `make test` on this branch (rebased on `main` after v0.10.0): all Go
  unit-test packages pass, including the new tests. `test/releasepolicy`
  fails identically on unmodified `main` in the test environment because `jq`
  is not installed there; it does not touch this change.
- `go test ./cmd/integration/` (envtest 1.36, golden files): pass, goldens
  unchanged.
- `gofmt`, `make vet` and `make lint` clean.
- Manual: three certification runs on two GB300 NVL72 racks with
  host-managed IMEX (1 tray per job, 4 GPUs each); no ComputeDomain created,
  all jobs completed (numbers in the issue).

### Checklist

- [x] Commits are signed off for the DCO (`git commit -s`)
- [ ] Documentation updated (a short section on the label and the trade-off
      in the GB200/GB300 platform docs, if maintainers want it in this PR)
