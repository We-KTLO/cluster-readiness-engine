// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package certification

import (
	"encoding/json"
	"strings"
	"testing"

	nvcrev1alpha1 "github.com/NVIDIA/cluster-readiness-engine/api/v1alpha1"
	_ "github.com/NVIDIA/cluster-readiness-engine/pkg/catalog"
	"github.com/NVIDIA/cluster-readiness-engine/pkg/platform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestRenderHostIMEX: the IMEX mode the dry-run detects from the target nodes
// reaches every GB300 category, one-node and multi-node alike; offline render
// (renderCertification) keeps the ComputeDomain path.
func TestRenderHostIMEX(t *testing.T) {
	for _, mode := range []string{"", platform.IMEXModeHost} {
		cert := &nvcrev1alpha1.Certification{
			ObjectMeta: metav1.ObjectMeta{Name: "gb300"},
			Spec: nvcrev1alpha1.CertificationSpec{
				Target: nvcrev1alpha1.TargetSpec{
					NodeSelector: map[string]string{"nvidia.com/gpu.product": "NVIDIA-GB300"},
				},
				Categories: []nvcrev1alpha1.CertificateCategory{
					{Domain: "communication", Variant: "nccl-all-reduce",
						Options: &nvcrev1alpha1.CategoryOptions{NodesPerJob: hostIMEXInt32(2)}},
					{Domain: "communication", Variant: "nccl-loopback",
						Options: &nvcrev1alpha1.CategoryOptions{NodesPerJob: hostIMEXInt32(1)}},
					{Domain: "training", Variant: "nemotron5-8b",
						Options: &nvcrev1alpha1.CategoryOptions{NodesPerJob: hostIMEXInt32(1)}},
				},
			},
		}
		workflows, err := renderCertificationWithIMEX(cert, "", "", nil, "", mode)
		if err != nil {
			t.Fatalf("mode %q: render: %v", mode, err)
		}
		for _, wf := range workflows {
			b, err := json.Marshal(wf.Spec)
			if err != nil {
				t.Fatal(err)
			}
			out := string(b)
			host := mode == platform.IMEXModeHost
			hasCD := strings.Contains(out, `"kind":"ComputeDomain"`)
			hasClaim := strings.Contains(out, "compute-domain")
			hasCDI := strings.Contains(out, `"runtimeClassName":"nvidia"`) &&
				strings.Contains(out, `"`+platform.IMEXChannelCDIAnnotation+`":"nvidia.com/imex-channel=all"`)
			if hasCD == host || hasClaim == host || hasCDI != host {
				t.Errorf("%s mode %q: ComputeDomain=%v compute-domain claim=%v CDI=%v",
					wf.Name, mode, hasCD, hasClaim, hasCDI)
			}
		}
	}
}

func hostIMEXInt32(v int32) *int32 { return &v }
