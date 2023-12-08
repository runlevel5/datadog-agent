package mutate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestInjectAgentSidecar(t *testing.T) {
	tests := []struct {
		name             string
		pod              *corev1.Pod
		wantErr          bool
		wantSidecar      bool
		wantSidecarEnvs  func([]corev1.EnvVar)
		wantSidecarProps func([]corev1.Container)
	}{
		{
			name:    "nominal case: java",
			pod:     fakePod("java-pod"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := injectAgentSidecar(tt.pod, "", nil)
			assert.Nil(t, err)
			containers := tt.pod.Spec.Containers
			assert.Equal(t, "datadog-agent-sidecar", containers[len(containers)-1].Name)
			verifySidecarEnvs(&containers[len(containers)-1].Env)
			verifySidecarProps(&containers[len(containers)-1])
		})
	}
}

func verifySidecarEnvs(envVars *[]corev1.EnvVar) {

}

func verifySidecarProps(container *corev1.Container) {

}
