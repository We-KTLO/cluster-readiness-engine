// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package certification

import (
	"encoding/json"
	"strings"
	"testing"

	nvcrev1alpha1 "github.com/NVIDIA/cluster-readiness-engine/api/v1alpha1"
	"github.com/NVIDIA/cluster-readiness-engine/pkg/catalog"
	"github.com/NVIDIA/cluster-readiness-engine/pkg/gpu"
	"github.com/NVIDIA/cluster-readiness-engine/pkg/platform"
)

// TestRenderHostIMEX: the IMEX mode the dry-run detects from the target nodes
// reaches every GB300 category, one-node and multi-node alike; offline render
// (mode "") keeps the ComputeDomain path.
func TestRenderHostIMEX(t *testing.T) {
	for _, mode := range []string{"", platform.IMEXModeHost} {
		cert := &nvcrev1alpha1.Certification{
			Name: "gb300",
			Spec: nvcrev1alpha1.CertificationSpec{
				Target: nvcrev1alpha1.TargetSpec{
					NodeSelector: map[string]string{gpu.ProductLabel: "NVIDIA-GB300"},
				},
				Categories: []nvcrev1alpha1.CertificateCategory{
					{Domain: testDomainCommunication, Variant: testVariantNCCLAllReduce,
						Options: &nvcrev1alpha1.CategoryOptions{NodesPerJob: new(int32(2))}},
					{Domain: testDomainCommunication, Variant: "nccl-loopback",
						Options: &nvcrev1alpha1.CategoryOptions{NodesPerJob: new(int32(1))}},
					{Domain: testDomainTraining, Variant: hostIMEXTrainingVariant,
						Options: &nvcrev1alpha1.CategoryOptions{NodesPerJob: new(int32(1))}},
				},
			},
		}
		workflows, err := renderCertification(cert, "", "", nil, "", catalog.DefaultOCISecondaryNetwork, mode)
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

// hostIMEXTrainingVariant is the one-node training category the test renders.
const hostIMEXTrainingVariant = "nemotron5-8b"
