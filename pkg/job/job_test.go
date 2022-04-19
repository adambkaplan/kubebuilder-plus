package job

import (
	"testing"

	"github.com/onsi/gomega"
	kbatch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

func TestIndexJob(t *testing.T) {
	boolPtr := true
	cases := []struct {
		name      string
		ownerRefs []metav1.OwnerReference
		expected  []string
	}{
		{
			name: "no owner",
		},
		{
			name: "owned by something else",
			ownerRefs: []metav1.OwnerReference{
				{
					APIVersion: "test.object/v1",
					Kind:       "thing",
					Name:       "hello",
				},
			},
		},
		{
			name: "owned by CronJob",
			ownerRefs: []metav1.OwnerReference{
				{
					APIVersion: batchv1.GroupVersion.String(),
					Kind:       "CronJob",
					Name:       "my-cron",
					Controller: &boolPtr,
				},
			},
			expected: []string{"my-cron"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := &kbatch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
					Name:      "job",
				},
			}
			job.SetOwnerReferences(tc.ownerRefs)
			o := gomega.NewWithT(t)
			actual := indexJob(job)
			o.Expect(actual).To(gomega.Equal(tc.expected))
		})
	}
}
