//go:build kubeapiserver

package language_detection

import (
	"fmt"
	"github.com/DataDog/datadog-agent/comp/core"
	"github.com/DataDog/datadog-agent/comp/core/workloadmeta"
	langUtil "github.com/DataDog/datadog-agent/pkg/languagedetection/util"
	pbgo "github.com/DataDog/datadog-agent/pkg/proto/pbgo/process"
	"github.com/DataDog/datadog-agent/pkg/util/fxutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"reflect"
	"testing"
	"time"
)

////////////////////////////////
//                            //
//   Owners Languages Tests   //
//                            //
////////////////////////////////

func TestOwnersLanguagesGetOrInitialise(t *testing.T) {
	mockNamespacedOwnerRef := langUtil.NewNamespacedOwnerReference("api-version", "deployment", "some-name", "some-ns")
	tests := []struct {
		name            string
		ownersLanguages OwnersLanguages
		ownerRef        langUtil.NamespacedOwnerReference
		expected        *ContainersLanguageWithDirtyFlag
	}{
		{
			name:            "missing owner should get initialized",
			ownersLanguages: *NewOwnersLanguages(),
			ownerRef:        mockNamespacedOwnerRef,
			expected:        NewContainersLanguageWithDirtyFlag(),
		},
		{
			name: "should return containers languages for existing owner",
			ownersLanguages: OwnersLanguages{
				containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
					mockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
								"java": {},
							},
						},
						dirty: false,
					},
				},
			},

			ownerRef: mockNamespacedOwnerRef,
			expected: &ContainersLanguageWithDirtyFlag{
				languages: langUtil.ContainersLanguages{
					*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
						"java": {},
					},
				},
				dirty: false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actual := test.ownersLanguages.getOrInitialize(test.ownerRef)
			assert.True(tt, reflect.DeepEqual(actual, test.expected), fmt.Sprintf("Expected %v, found %v", test.expected, actual))
		})
	}
}

func TestOwnersLanguagesMerge(t *testing.T) {
	mockNamespacedOwnerRef := langUtil.NewNamespacedOwnerReference("api-version", "deployment", "some-uid", "some-ns")
	otherMockNamespacedOwnerRef := langUtil.NewNamespacedOwnerReference("api-version", "statefulset", "some-uid", "some-ns")
	cleanMockNamespacedOwnerRef := langUtil.NewNamespacedOwnerReference("api-version", "daemonset", "some-uid", "some-ns")

	tests := []struct {
		name               string
		ownersLanguages    OwnersLanguages
		other              OwnersLanguages
		expectedAfterMerge OwnersLanguages
	}{
		{
			name:               "merge empty owners languages",
			ownersLanguages:    *NewOwnersLanguages(),
			other:              *NewOwnersLanguages(),
			expectedAfterMerge: *NewOwnersLanguages(),
		},
		{
			name:            "merge non-empty other to empty self",
			ownersLanguages: *NewOwnersLanguages(),
			other: OwnersLanguages{
				containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
					mockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
								"java": {},
							},
						},
						dirty: false,
					},
				},
			},
			expectedAfterMerge: OwnersLanguages{
				containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
					mockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
								"java": {},
							},
						},
						dirty: true,
					},
				},
			},
		},
		{
			name: "merge empty other to non-empty self",
			ownersLanguages: OwnersLanguages{
				containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
					mockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
								"java": {},
							},
						},
						dirty: false,
					},
				},
			},
			other: *NewOwnersLanguages(),
			expectedAfterMerge: OwnersLanguages{
				containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
					mockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
								"java": {},
							},
						},
						dirty: false,
					},
				},
			},
		},
		{
			name: "merge non-empty other to non-empty self",
			ownersLanguages: OwnersLanguages{
				containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
					mockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
								"java": {},
								"ruby": {},
							},
						},
						dirty: false,
					},
					cleanMockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-other-container"): {
								"java": {},
								"ruby": {},
							},
						},
						dirty: false,
					},
				},
			},
			other: OwnersLanguages{
				containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
					mockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
								"perl": {},
							},
							*langUtil.NewContainer("some-other-container"): langUtil.LanguageSet{
								"cpp": {},
							},
						},
					},
					otherMockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-other-container"): {
								"java": {},
								"cpp":  {},
							},
						},
					},
				},
			},
			expectedAfterMerge: OwnersLanguages{
				containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
					mockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-container"): langUtil.LanguageSet{
								"java": {},
								"ruby": {},
								"perl": {},
							},
							*langUtil.NewContainer("some-other-container"): langUtil.LanguageSet{
								"cpp": {},
							},
						},
						dirty: false,
					},
					cleanMockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-other-container"): {
								"java": {},
								"ruby": {},
							},
						},
						dirty: false,
					},
					otherMockNamespacedOwnerRef: {
						languages: langUtil.ContainersLanguages{
							*langUtil.NewContainer("some-other-container"): {
								"java": {},
								"cpp":  {},
							},
						},
						dirty: true,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			test.ownersLanguages.merge(test.other)
			assert.True(tt, reflect.DeepEqual(test.ownersLanguages, test.expectedAfterMerge), fmt.Sprintf("Expected %v, found %v", test.expectedAfterMerge, test.ownersLanguages))
		})
	}
}

func TestOwnersLanguagesClean(t *testing.T) {
	mockSupportedOwnerA := langUtil.NewNamespacedOwnerReference("api-version", langUtil.KindDeployment, "deploymentA", "ns")
	mockSupportedOwnerB := langUtil.NewNamespacedOwnerReference("api-version", langUtil.KindDeployment, "deploymentB", "ns")
	mockUnsupportedOwner := langUtil.NewNamespacedOwnerReference("api-version", "Daemonset", "some-name", "ns")

	ownersLanguages := OwnersLanguages{
		containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
			mockSupportedOwnerA: {
				languages: langUtil.ContainersLanguages{
					*langUtil.NewContainer("some-container"): {
						"java": {},
						"ruby": {},
						"perl": {},
					},
				},
				dirty: true,
			},

			mockSupportedOwnerB: {
				languages: langUtil.ContainersLanguages{
					*langUtil.NewContainer("some-container"): {
						"java": {},
					},
					*langUtil.NewContainer("some-other-container"): {
						"cpp": {},
					},
				},
				dirty: false,
			},
		},
	}

	mockStore := fxutil.Test[workloadmeta.Mock](t, fx.Options(
		core.MockBundle(),
		fx.Supply(workloadmeta.NewParams()),
		workloadmeta.MockModuleV2(),
	))

	err := ownersLanguages.clean(mockStore)
	assert.NoErrorf(t, err, "clean operation should not return an error")

	// Assertion: deploymentA is added to the store with the correct detected languages
	// Reason: deploymentA has detected languages with dirty flag set to true
	assert.Eventuallyf(t,
		func() bool {
			deploymentA, err := mockStore.GetKubernetesDeployment("ns/deploymentA")
			if err != nil {
				return false
			}

			return deploymentA.DetectedLanguages.EqualTo(langUtil.ContainersLanguages{
				*langUtil.NewContainer("some-container"): {
					"perl": struct{}{},
					"java": struct{}{},
					"ruby": struct{}{},
				},
			})

		},
		2*time.Second,
		100*time.Millisecond,
		"Should find deploymentA in workloadmeta store with the correct languages")

	// Assertion: deploymentB is added to the store with the correct detected languages
	// Reason: deploymentB has detected languages with dirty flag set to false
	_, err = mockStore.GetKubernetesDeployment("ns/deploymentB")
	assert.Errorf(t, err, "deploymentB should not be in store since the dirty flag is set to false")

	// Assertion: dirty flags of flushed languages are reset to false
	assert.False(t, ownersLanguages.containersLanguages[mockSupportedOwnerA].dirty, "deploymentA dirty flag should be reset to false")
	assert.False(t, ownersLanguages.containersLanguages[mockSupportedOwnerB].dirty, "deploymentB dirty flag should be reset to false")
	assert.False(t, ownersLanguages.containersLanguages[mockSupportedOwnerB].dirty, "daemonset dirty flag should not be reset to false")

	// set deploymentB dirty flag
	ownersLanguages.containersLanguages[mockSupportedOwnerB].dirty = true

	// add unsupported owner to ownerslanguages
	ownersLanguages.containersLanguages[mockUnsupportedOwner] = &ContainersLanguageWithDirtyFlag{
		languages: langUtil.ContainersLanguages{
			*langUtil.NewContainer("some-container"): {
				"perl": struct{}{},
				"java": struct{}{},
				"ruby": struct{}{},
			},
			*langUtil.NewContainer("some-other-container"): {
				"cpp": struct{}{},
			},
		},
		dirty: true,
	}

	// clean owners languages
	err = ownersLanguages.clean(mockStore)
	assert.Errorf(t, err, "clean operation should return an error due to unsupported resource kind")

	// Assert that deploymentB is not added to the store with the correct languages
	assert.Eventuallyf(t, func() bool {
		deploymentB, err := mockStore.GetKubernetesDeployment("ns/deploymentB")
		if err != nil {
			return false
		}

		languagesInStore := deploymentB.DetectedLanguages

		if !languagesInStore.EqualTo(langUtil.ContainersLanguages{
			*langUtil.NewContainer("some-container"):       {"java": struct{}{}},
			*langUtil.NewContainer("some-other-container"): {"cpp": struct{}{}},
		}) {
			return false
		}

		return true
	}, 2*time.Second, 100*time.Millisecond, "Should find deploymentB in workloadmeta store with the correct languages")

	// Assertion: dirty flags of flushed languages are reset to false
	assert.False(t, ownersLanguages.containersLanguages[mockSupportedOwnerA].dirty, "deploymentA dirty flag should be reset to false")
	assert.False(t, ownersLanguages.containersLanguages[mockSupportedOwnerB].dirty, "deploymentB dirty flag should be reset to false")
	assert.False(t, ownersLanguages.containersLanguages[mockSupportedOwnerB].dirty, "daemonset dirty flag should not be reset to false")

}

////////////////////////////////
//                            //
//    Util Functions Tests    //
//                            //
////////////////////////////////

func TestGetContainersLanguagesFromPodDetail(t *testing.T) {
	containerDetails := []*pbgo.ContainerLanguageDetails{
		{
			ContainerName: "mono-lang",
			Languages: []*pbgo.Language{
				{Name: "java"},
			},
		},
		{
			ContainerName: "bi-lang",
			Languages: []*pbgo.Language{
				{Name: "java"},
				{Name: "cpp"},
			},
		},
		{
			ContainerName: "tri-lang",
			Languages: []*pbgo.Language{
				{Name: "java"},
				{Name: "go"},
				{Name: "python"},
			},
		},
	}

	initContainerDetails := []*pbgo.ContainerLanguageDetails{
		{
			ContainerName: "init-mono-lang",
			Languages: []*pbgo.Language{
				{Name: "java"},
			},
		},
	}

	podLanguageDetails := &pbgo.PodLanguageDetails{
		Namespace:            "default",
		ContainerDetails:     containerDetails,
		InitContainerDetails: initContainerDetails,
		Ownerref: &pbgo.KubeOwnerInfo{
			Kind: "ReplicaSet",
			Name: "dummyrs-2342347",
		},
	}

	containerslanguages := getContainersLanguagesFromPodDetail(podLanguageDetails)

	expectedContainersLanguages := langUtil.ContainersLanguages{
		*langUtil.NewContainer("mono-lang"): {
			"java": struct{}{},
		},
		*langUtil.NewContainer("bi-lang"): {
			"java": struct{}{},
			"cpp":  struct{}{},
		},
		*langUtil.NewContainer("tri-lang"): {
			"java":   struct{}{},
			"go":     struct{}{},
			"python": struct{}{},
		},
		*langUtil.NewInitContainer("init-mono-lang"): {
			"java": struct{}{},
		},
	}

	assert.True(t, reflect.DeepEqual(containerslanguages, &expectedContainersLanguages), fmt.Sprintf("Expected %v, found %v", &expectedContainersLanguages, containerslanguages))
}

func TestGetOwnersLanguages(t *testing.T) {
	defaultNs := "default"
	customNs := "custom"

	podALanguageDetails := &pbgo.PodLanguageDetails{
		Namespace: defaultNs,
		Name:      "pod-a",
		ContainerDetails: []*pbgo.ContainerLanguageDetails{
			{
				ContainerName: "container-1",
				Languages: []*pbgo.Language{
					{Name: "java"},
					{Name: "cpp"},
					{Name: "go"},
				},
			},
			{
				ContainerName: "container-2",
				Languages: []*pbgo.Language{
					{Name: "java"},
					{Name: "python"},
				},
			},
		},
		InitContainerDetails: []*pbgo.ContainerLanguageDetails{
			{
				ContainerName: "init-container-3",
				Languages: []*pbgo.Language{
					{Name: "java"},
					{Name: "cpp"},
				},
			},
			{
				ContainerName: "init-container-4",
				Languages: []*pbgo.Language{
					{Name: "java"},
					{Name: "python"},
				},
			},
		},
		Ownerref: &pbgo.KubeOwnerInfo{
			Kind: "ReplicaSet",
			Name: "dummyrs-1-2342347",
		},
	}

	podBLanguageDetails := &pbgo.PodLanguageDetails{
		Namespace: customNs,
		Name:      "pod-b",
		ContainerDetails: []*pbgo.ContainerLanguageDetails{
			{
				ContainerName: "container-5",
				Languages: []*pbgo.Language{
					{Name: "python"},
					{Name: "cpp"},
					{Name: "go"},
				},
			},
			{
				ContainerName: "container-6",
				Languages: []*pbgo.Language{
					{Name: "java"},
					{Name: "ruby"},
				},
			},
		},
		InitContainerDetails: []*pbgo.ContainerLanguageDetails{
			{
				ContainerName: "init-container-7",
				Languages: []*pbgo.Language{
					{Name: "java"},
					{Name: "cpp"},
				},
			},
			{
				ContainerName: "init-container-8",
				Languages: []*pbgo.Language{
					{Name: "java"},
					{Name: "python"},
				},
			},
		},
		Ownerref: &pbgo.KubeOwnerInfo{
			Kind: "ReplicaSet",
			Name: "dummyrs-2-2342347",
		},
	}

	mockRequestData := &pbgo.ParentLanguageAnnotationRequest{
		PodDetails: []*pbgo.PodLanguageDetails{
			podALanguageDetails,
			podBLanguageDetails,
		},
	}

	expectedContainersLanguagesA := ContainersLanguageWithDirtyFlag{
		dirty: true,
		languages: langUtil.ContainersLanguages{
			*langUtil.NewContainer("container-1"): {
				"java": struct{}{},
				"cpp":  struct{}{},
				"go":   struct{}{},
			},
			*langUtil.NewContainer("container-2"): {
				"java":   struct{}{},
				"python": struct{}{},
			},
			*langUtil.NewInitContainer("init-container-3"): {
				"java": struct{}{},
				"cpp":  struct{}{},
			},
			*langUtil.NewInitContainer("init-container-4"): {
				"java":   struct{}{},
				"python": struct{}{},
			},
		},
	}

	expectedContainersLanguagesB := ContainersLanguageWithDirtyFlag{
		dirty: true,
		languages: langUtil.ContainersLanguages{
			*langUtil.NewContainer("container-5"): {
				"python": struct{}{},
				"cpp":    struct{}{},
				"go":     struct{}{},
			},
			*langUtil.NewContainer("container-6"): {
				"java": struct{}{},
				"ruby": struct{}{},
			},
			*langUtil.NewInitContainer("init-container-7"): {
				"java": struct{}{},
				"cpp":  struct{}{},
			},
			*langUtil.NewInitContainer("init-container-8"): {
				"java":   struct{}{},
				"python": struct{}{},
			},
		},
	}

	expectedOwnersLanguages := &OwnersLanguages{
		containersLanguages: map[langUtil.NamespacedOwnerReference]*ContainersLanguageWithDirtyFlag{
			langUtil.NewNamespacedOwnerReference("apps/v1", "Deployment", "dummyrs-1", "default"): &expectedContainersLanguagesA,
			langUtil.NewNamespacedOwnerReference("apps/v1", "Deployment", "dummyrs-2", "custom"):  &expectedContainersLanguagesB,
		},
	}

	actualOwnersLanguages := getOwnersLanguages(mockRequestData)

	assert.True(t, reflect.DeepEqual(expectedOwnersLanguages, actualOwnersLanguages), fmt.Sprintf("Expected %v, found %v", expectedOwnersLanguages, actualOwnersLanguages))
}

func TestGeneratePushEvent(t *testing.T) {
	mockSupportedOwner := langUtil.NewNamespacedOwnerReference("api-version", "Deployment", "some-name", "some-ns")
	mockUnsupportedOwner := langUtil.NewNamespacedOwnerReference("api-version", "UnsupportedResourceKind", "some-name", "some-ns")

	tests := []struct {
		name          string
		languages     langUtil.ContainersLanguages
		owner         langUtil.NamespacedOwnerReference
		expectedEvent *workloadmeta.Event
	}{
		{
			name:          "unsupported owner",
			languages:     make(langUtil.ContainersLanguages),
			owner:         mockUnsupportedOwner,
			expectedEvent: nil,
		},
		{
			name:      "empty containers languages object with supported owner",
			languages: make(langUtil.ContainersLanguages),
			owner:     mockSupportedOwner,
			expectedEvent: &workloadmeta.Event{
				Type: workloadmeta.EventTypeSet,
				Entity: &workloadmeta.KubernetesDeployment{
					EntityID: workloadmeta.EntityID{
						Kind: workloadmeta.KindKubernetesDeployment,
						ID:   "some-ns/some-name",
					},
					DetectedLanguages: make(langUtil.ContainersLanguages),
				},
			},
		},
		{
			name: "non-empty containers languages with supported owner",
			languages: langUtil.ContainersLanguages{
				langUtil.Container{Name: "container-1", Init: false}: {
					"java": struct{}{},
					"cpp":  struct{}{},
				},
				langUtil.Container{Name: "container-2", Init: false}: {
					"java": struct{}{},
					"cpp":  struct{}{},
				},
				langUtil.Container{Name: "container-3", Init: true}: {
					"python": struct{}{},
					"ruby":   struct{}{},
				},
				langUtil.Container{Name: "container-4", Init: true}: {
					"go":   struct{}{},
					"java": struct{}{},
				},
			},
			owner: mockSupportedOwner,
			expectedEvent: &workloadmeta.Event{
				Type: workloadmeta.EventTypeSet,
				Entity: &workloadmeta.KubernetesDeployment{
					EntityID: workloadmeta.EntityID{
						Kind: workloadmeta.KindKubernetesDeployment,
						ID:   "some-ns/some-name",
					},
					DetectedLanguages: langUtil.ContainersLanguages{
						langUtil.Container{Name: "container-1", Init: false}: {
							"java": struct{}{},
							"cpp":  struct{}{},
						},
						langUtil.Container{Name: "container-2", Init: false}: {
							"java": struct{}{},
							"cpp":  struct{}{},
						},
						langUtil.Container{Name: "container-3", Init: true}: {
							"python": struct{}{},
							"ruby":   struct{}{},
						},
						langUtil.Container{Name: "container-4", Init: true}: {
							"go":   struct{}{},
							"java": struct{}{},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actualEvent := generatePushEvent(test.owner, test.languages)

			if actualEvent == nil && test.expectedEvent == nil {
				return

			}

			// Event has correct type
			assert.Equal(tt, test.expectedEvent.Type, actualEvent.Type)

			// Event entity has correct Entity Id
			assert.True(
				tt,
				reflect.DeepEqual(test.expectedEvent.Entity.GetID(), actualEvent.Entity.GetID()),
				fmt.Sprintf(
					"entity id is not correct: expected %v, but found %v",
					test.expectedEvent.Entity.GetID(),
					actualEvent.Entity.GetID(),
				),
			)

			// Event has correct detected languages
			actualDeploymentEntity := actualEvent.Entity.(*workloadmeta.KubernetesDeployment)
			expectedDeploymentEntity := test.expectedEvent.Entity.(*workloadmeta.KubernetesDeployment)

			actualDetectedLanguages := actualDeploymentEntity.DetectedLanguages
			expectedDetectedLanguages := expectedDeploymentEntity.DetectedLanguages

			assert.True(
				tt,
				actualDetectedLanguages.EqualTo(expectedDetectedLanguages),
				fmt.Sprintf("container languages are not correct: expected %v, found %v", expectedDetectedLanguages, actualDetectedLanguages),
			)
		})
	}
}
