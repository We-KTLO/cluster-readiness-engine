// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package platform

import (
	"encoding/json"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func node(labels map[string]string) corev1.Node {
	return corev1.Node{Labels: labels}
}

func TestDetectIMEXMode(t *testing.T) {
	host := map[string]string{IMEXModeLabel: IMEXModeHost}
	for _, tc := range []struct {
		name  string
		nodes []corev1.Node
		want  string
	}{
		{"no nodes", nil, ""},
		{"all labelled", []corev1.Node{node(host), node(host)}, IMEXModeHost},
		{"one unlabelled", []corev1.Node{node(host), node(nil)}, ""},
		{"other value", []corev1.Node{node(map[string]string{IMEXModeLabel: "dra"})}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectIMEXMode(tc.nodes); got != tc.want {
				t.Errorf("DetectIMEXMode = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestWorkloadRunHostIMEX: the WorkloadRun GB200/GB300 override follows the same switch, for one
// node and several, and host mode leaves no compute-domain claim behind (the AWS GB300 RoCE
// override re-declares it).
func TestWorkloadRunHostIMEX(t *testing.T) {
	for _, nodes := range []int32{1, 2} {
		for _, fw := range []string{"torch", "mpi"} {
			for _, mode := range []string{"", IMEXModeHost} {
				b, err := json.Marshal(BuildOverrides(OverrideConfig{EntryName: "wr", NodesPerJob: nodes, GpusPerNode: 4, EnableMNNVL: true, FrameworkType: fw, IMEXMode: mode}))
				if err != nil {
					t.Fatal(err)
				}
				out := string(b)
				host := mode == IMEXModeHost
				hasCD := strings.Contains(out, `"kind":"ComputeDomain"`)
				hasClaim := strings.Contains(out, "compute-domain")
				hasCDI := strings.Contains(out, `"runtimeClassName":"nvidia"`) && strings.Contains(out, `"`+IMEXChannelCDIAnnotation+`":"nvidia.com/imex-channel=all"`)
				if hasCD == host || hasClaim == host || hasCDI != host {
					t.Errorf("nodes=%d %s mode %q: ComputeDomain=%v compute-domain claim=%v CDI=%v", nodes, fw, mode, hasCD, hasClaim, hasCDI)
				}
			}
		}
	}
}
