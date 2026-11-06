// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	nvcrev1alpha1 "github.com/NVIDIA/cluster-readiness-engine/api/v1alpha1"
)

// Literals shared by the host IMEX cases.
const (
	imexTestDomainCommunication = "communication"
	imexTestDomainTraining      = "training"
	imexTestArch                = "gb300"
)

// TestHostIMEXMode: with IMEXMode "host" the GB200/GB300 entries take the IMEX channel from the
// host's CDI spec and create no ComputeDomain and no channel claim; without it they render the DRA
// ComputeDomain path. The single-node loopback cases pin that the per-job ComputeDomain on a
// one-node job is removed too. The AWS GB300 RoCE fragments in the overrides re-declare the channel
// claim, so "compute-domain" must not appear anywhere in host mode.
func TestHostIMEXMode(t *testing.T) {
	for _, tc := range []struct {
		domain, variant string
		nodes           int32
	}{
		{imexTestDomainCommunication, "nccl-all-reduce", 2},
		{imexTestDomainCommunication, "nccl-all-gather", 2},
		{imexTestDomainCommunication, "nccl-alltoall", 2},
		{imexTestDomainCommunication, "nccl-loopback", 1},
		{imexTestDomainCommunication, "nccl-loopback-nvswitch", 1},
		{imexTestDomainTraining, "nemotron5-8b", 2},
		{imexTestDomainTraining, "nemotron5-8b", 1},
		{imexTestDomainTraining, "nemotron5-56b", 8},
	} {
		for _, mode := range []string{"", "host"} {
			t.Run(fmt.Sprintf("%s/nodes=%d/%s", tc.variant, tc.nodes, mode), func(t *testing.T) {
				entry := Lookup(tc.domain, tc.variant)
				if entry == nil {
					t.Fatalf("no catalog entry %s/%s", tc.domain, tc.variant)
				}
				spec, err := entry.Build(nvcrev1alpha1.TargetSpec{}, BuildConfig{
					NodesPerJob: tc.nodes, GpusPerNode: 4, GPUArchitecture: imexTestArch, EnableMNNVL: true, IMEXMode: mode,
				})
				if err != nil {
					t.Fatalf("build: %v", err)
				}
				b, err := json.Marshal(spec)
				if err != nil {
					t.Fatal(err)
				}
				out := string(b)
				hasCD := strings.Contains(out, `"kind":"ComputeDomain"`)
				hasCDI := strings.Contains(out, `"runtimeClassName":"nvidia"`) && strings.Contains(out, `"cdi.k8s.io/imex":"nvidia.com/imex-channel=all"`)
				hasClaim := strings.Contains(out, "compute-domain")
				if mode == "host" && (hasCD || hasClaim || !hasCDI) {
					t.Errorf("host mode: ComputeDomain=%v compute-domain claim=%v CDI=%v, want false/false/true", hasCD, hasClaim, hasCDI)
				}
				if mode == "" && (!hasCD || !hasClaim || hasCDI) {
					t.Errorf("default: ComputeDomain=%v compute-domain claim=%v CDI=%v, want true/true/false", hasCD, hasClaim, hasCDI)
				}
			})
		}
	}
}
