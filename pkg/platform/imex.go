// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package platform

import corev1 "k8s.io/api/core/v1"

// IMEXModeLabel marks a node whose IMEX channel is served by the host's own IMEX
// service (the provider-managed nvidia-imex daemon) and injected into pods through
// a CDI spec, rather than by the DRA driver's ComputeDomain. Value: IMEXModeHost.
const IMEXModeLabel = "nvcre.nvidia.com/imex-mode"

// IMEXModeHost is the IMEXModeLabel value for host-managed IMEX.
const IMEXModeHost = "host"

// IMEXChannelCDIAnnotation requests every IMEX channel from the host CDI spec.
const IMEXChannelCDIAnnotation = "cdi.k8s.io/imex"

// DetectIMEXMode returns IMEXModeHost when there is at least one node and every
// node carries IMEXModeLabel=host, and "" otherwise. It never guesses: a mixed or
// unlabelled set keeps the ComputeDomain path.
func DetectIMEXMode(nodes []corev1.Node) string {
	if len(nodes) == 0 {
		return ""
	}
	for i := range nodes {
		if nodes[i].Labels[IMEXModeLabel] != IMEXModeHost {
			return ""
		}
	}
	return IMEXModeHost
}
