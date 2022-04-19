package job

import (
	"context"
	"fmt"
	"time"

	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

var (
	jobOwnerKey             = ".metadata.controller"
	apiGVStr                = batchv1.GroupVersion.String()
	scheduledTimeAnnotation = "batch.tutorial.kubebuilder.io/scheduled-at"
)

// SetupJobIndexWithManager creates an in-memory field index for Jobs. In order to allow our
// reconciler to quickly look up Jobs by their owner, we'll need an index. We declare an index key
// that we can later use with the client as a pseudo-field name, and then describe how to extract
// the indexed value from the Job object.  The indexer will automatically take care of namespaces
// for us, so we just have to extract the owner name if the Job has a CronJob owner.
func SetupJobIndexWithManager(mgr ctrl.Manager) error {
	return mgr.GetFieldIndexer().IndexField(context.Background(), &kbatch.Job{}, jobOwnerKey, indexJob)
}

func indexJob(rawObj client.Object) []string {
	// grab the job object, extract the owner...
	job := rawObj.(*kbatch.Job)
	owner := metav1.GetControllerOf(job)
	if owner == nil {
		return nil
	}
	// ...make sure it's a CronJob...
	if owner.APIVersion != apiGVStr || owner.Kind != "CronJob" {
		return nil
	}

	// ...and if so, return it
	return []string{owner.Name}
}

// ListJobsForCronJob lists jobs owned by the CronJob. It takes advantage of the in-memory index.
func ListJobsForCronJob(ctx context.Context, c client.Client, namespace string, cronJob string) (*kbatch.JobList, error) {
	/*
		### 2: List all active jobs, and update the status

		To fully update our status, we'll need to list all child jobs in this namespace that belong to this CronJob.
		Similarly to Get, we can use the List method to list the child jobs.  Notice that we use variadic options to
		set the namespace and field match (which is actually an index lookup that we set up below).
	*/
	var childJobs kbatch.JobList
	if err := c.List(ctx, &childJobs, client.InNamespace(namespace), client.MatchingFields{jobOwnerKey: cronJob}); err != nil {
		return nil, err
	}
	return &childJobs, nil
}

// IsJobFinished indicates if a job has reached a terminated state.
func IsJobFinished(job *kbatch.Job) (bool, kbatch.JobConditionType) {
	/*
		We consider a job "finished" if it has a "Complete" or "Failed" condition marked as true.
		Status conditions allow us to add extensible status information to our objects that other
		humans and controllers can examine to check things like completion and health.
	*/
	for _, c := range job.Status.Conditions {
		if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
			return true, c.Type
		}
	}
	return false, ""
}

func GetScheduledTimeForJob(job *kbatch.Job) (*time.Time, error) {
	/*
		We'll use a helper to extract the scheduled time from the annotation that
		we added during job creation.
	*/
	timeRaw := job.Annotations[scheduledTimeAnnotation]
	if len(timeRaw) == 0 {
		return nil, nil
	}

	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
	if err != nil {
		return nil, err
	}
	return &timeParsed, nil
}

func ConstructJobForCronJob(cronJob *batchv1.CronJob, scheduledTime time.Time, scheme *runtime.Scheme) (*kbatch.Job, error) {
	/*
		We need to construct a job based on our CronJob's template.  We'll copy over the spec
		from the template and copy some basic object meta.

		Then, we'll set the "scheduled time" annotation so that we can reconstitute our
		`LastScheduleTime` field each reconcile.

		Finally, we'll need to set an owner reference.  This allows the Kubernetes garbage collector
		to clean up jobs when we delete the CronJob, and allows controller-runtime to figure out
		which cronjob needs to be reconciled when a given job changes (is added, deleted, completes, etc).
	*/

	// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
	name := fmt.Sprintf("%s-%d", cronJob.Name, scheduledTime.Unix())

	job := &kbatch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   cronJob.Namespace,
		},
		Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
	}
	for k, v := range cronJob.Spec.JobTemplate.Annotations {
		job.Annotations[k] = v
	}
	job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
	for k, v := range cronJob.Spec.JobTemplate.Labels {
		job.Labels[k] = v
	}
	if err := ctrl.SetControllerReference(cronJob, job, scheme); err != nil {
		return nil, err
	}

	return job, nil
}
