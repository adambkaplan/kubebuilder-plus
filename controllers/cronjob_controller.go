/*
Copyright 2022 The Kubernetes authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// +kubebuilder:docs-gen:collapse=Apache License

/*
We'll start out with some imports.  You'll see below that we'll need a few more imports
than those scaffolded for us.  We'll talk about each one when we use it.
*/
package controllers

import (
	"context"
	"sort"
	"time"

	kbatch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
	"tutorial.kubebuilder.io/project/pkg/cron"
	"tutorial.kubebuilder.io/project/pkg/job"
)

/*
Next, we'll need a Clock, which will allow us to fake timing in our tests.
*/

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Clock
}

/*
We'll mock out the clock to make it easier to jump around in time while testing,
the "real" clock just calls `time.Now`.
*/
type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

// clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	Now() time.Time
}

// +kubebuilder:docs-gen:collapse=Clock

/*
Notice that we need a few more RBAC permissions -- since we're creating and
managing jobs now, we'll need permissions for those, which means adding
a couple more [markers](/reference/markers/rbac.md).
*/

//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

/*
Now, we get to the heart of the controller -- the reconciler logic.
*/
var (
	scheduledTimeAnnotation = "batch.tutorial.kubebuilder.io/scheduled-at"
)

func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	/*
		### 1: Load the CronJob by name

		We'll fetch the CronJob using our client.  All client methods take a
		context (to allow for cancellation) as their first argument, and the object
		in question as their last.  Get is a bit special, in that it takes a
		[`NamespacedName`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client?tab=doc#ObjectKey)
		as the middle argument (most don't have a middle argument, as we'll see
		below).

		Many client methods also take variadic options at the end.
	*/
	var cronJob batchv1.CronJob
	if err := r.Get(ctx, req.NamespacedName, &cronJob); err != nil {
		log.Error(err, "unable to fetch CronJob")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	childJobs, err := job.ListJobsForCronJob(ctx, r.Client, req.Namespace, req.Name)
	if err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	/*

		<aside class="note">

		<h1>What is this index about?</h1>

		<p>The reconciler fetches all jobs owned by the cronjob for the status. As our number of cronjobs increases,
		looking these up can become quite slow as we have to filter through all of them. For a more efficient lookup,
		these jobs will be indexed locally on the controller's name. A jobOwnerKey field is added to the
		cached job objects. This key references the owning controller and functions as the index. Later in this
		document we will configure the manager to actually index this field.</p>

		</aside>

		Once we have all the jobs we own, we'll split them into active, successful,
		and failed jobs, keeping track of the most recent run so that we can record it
		in status.  Remember, status should be able to be reconstituted from the state
		of the world, so it's generally not a good idea to read from the status of the
		root object.  Instead, you should reconstruct it every run.  That's what we'll
		do here.

		We can check if a job is "finished" and whether it succeeded or failed using status
		conditions.  We'll put that logic in a helper to make our code cleaner.
	*/

	// find the active list of jobs
	var activeJobs []*kbatch.Job
	var successfulJobs []*kbatch.Job
	var failedJobs []*kbatch.Job
	var mostRecentTime *time.Time // find the last run so we can update the status

	// +kubebuilder:docs-gen:collapse=isJobFinished

	// +kubebuilder:docs-gen:collapse=getScheduledTimeForJob

	for i, childJob := range childJobs.Items {
		_, finishedType := job.IsJobFinished(&childJob)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}

		// We'll store the launch time in an annotation, so we'll reconstitute that from
		// the active jobs themselves.
		scheduledTimeForJob, err := job.GetScheduledTimeForJob(&childJob)
		if err != nil {
			log.Error(err, "unable to parse schedule time for child job", "job", &childJob)
			continue
		}
		if scheduledTimeForJob != nil {
			if mostRecentTime == nil {
				mostRecentTime = scheduledTimeForJob
			} else if mostRecentTime.Before(*scheduledTimeForJob) {
				mostRecentTime = scheduledTimeForJob
			}
		}
	}

	if mostRecentTime != nil {
		cronJob.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		cronJob.Status.LastScheduleTime = nil
	}
	cronJob.Status.Active = nil
	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		cronJob.Status.Active = append(cronJob.Status.Active, *jobRef)
	}

	/*
		Here, we'll log how many jobs we observed at a slightly higher logging level,
		for debugging.  Notice how instead of using a format string, we use a fixed message,
		and attach key-value pairs with the extra information.  This makes it easier to
		filter and query log lines.
	*/
	log.V(1).Info("job count", "active jobs", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	/*
		Using the date we've gathered, we'll update the status of our CRD.
		Just like before, we use our client.  To specifically update the status
		subresource, we'll use the `Status` part of the client, with the `Update`
		method.

		The status subresource ignores changes to spec, so it's less likely to conflict
		with any other updates, and can have separate permissions.
	*/
	if err := r.Status().Update(ctx, &cronJob); err != nil {
		log.Error(err, "unable to update CronJob status")
		return ctrl.Result{}, err
	}

	/*
		Once we've updated our status, we can move on to ensuring that the status of
		the world matches what we want in our spec.

		### 3: Clean up old jobs according to the history limit

		First, we'll try to clean up old jobs, so that we don't leave too many lying
		around.
	*/

	// NB: deleting these is "best effort" -- if we fail on a particular one,
	// we won't requeue just to finish the deleting.
	if cronJob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i].Status.StartTime == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if int32(i) >= int32(len(failedJobs))-*cronJob.Spec.FailedJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete old failed job", "job", job)
			} else {
				log.V(0).Info("deleted old failed job", "job", job)
			}
		}
	}

	if cronJob.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulJobs, func(i, j int) bool {
			if successfulJobs[i].Status.StartTime == nil {
				return successfulJobs[j].Status.StartTime != nil
			}
			return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
		})
		for i, job := range successfulJobs {
			if int32(i) >= int32(len(successfulJobs))-*cronJob.Spec.SuccessfulJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); (err) != nil {
				log.Error(err, "unable to delete old successful job", "job", job)
			} else {
				log.V(0).Info("deleted old successful job", "job", job)
			}
		}
	}

	/* ### 4: Check if we're suspended

	If this object is suspended, we don't want to run any jobs, so we'll stop now.
	This is useful if something's broken with the job we're running and we want to
	pause runs to investigate or putz with the cluster, without deleting the object.
	*/

	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		log.V(1).Info("cronjob suspended, skipping")
		return ctrl.Result{}, nil
	}

	/*
		### 5: Get the next scheduled run

		If we're not paused, we'll need to calculate the next scheduled run, and whether
		or not we've got a run that we haven't processed yet.
	*/
	// +kubebuilder:docs-gen:collapse=getNextSchedule

	// figure out the next times that we need to create
	// jobs at (or anything we missed).
	missedRun, nextRun, err := cron.GetNextSchedule(&cronJob, r.Now())
	if err != nil {
		log.Error(err, "unable to figure out CronJob schedule")
		// we don't really care about requeuing until we get an update that
		// fixes the schedule, so don't return an error
		return ctrl.Result{}, nil
	}

	/*
		We'll prep our eventual request to requeue until the next job, and then figure
		out if we actually need to run.
	*/
	scheduledResult := ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())} // save this so we can re-use it elsewhere
	log = log.WithValues("now", r.Now(), "next run", nextRun)

	/*
		### 6: Run a new job if it's on schedule, not past the deadline, and not blocked by our concurrency policy

		If we've missed a run, and we're still within the deadline to start it, we'll need to run a job.
	*/
	if missedRun.IsZero() {
		log.V(1).Info("no upcoming scheduled times, sleeping until next")
		return scheduledResult, nil
	}

	// make sure we're not too late to start the run
	log = log.WithValues("current run", missedRun)
	tooLate := false
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*cronJob.Spec.StartingDeadlineSeconds) * time.Second).Before(r.Now())
	}
	if tooLate {
		log.V(1).Info("missed starting deadline for last run, sleeping till next")
		// TODO(directxman12): events
		return scheduledResult, nil
	}

	/*
		If we actually have to run a job, we'll need to either wait till existing ones finish,
		replace the existing ones, or just add new ones.  If our information is out of date due
		to cache delay, we'll get a requeue when we get up-to-date information.
	*/
	// figure out how to run this job -- concurrency policy might forbid us from running
	// multiple at the same time...
	if cronJob.Spec.ConcurrencyPolicy == batchv1.ForbidConcurrent && len(activeJobs) > 0 {
		log.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activeJobs))
		return scheduledResult, nil
	}

	// ...or instruct us to replace existing ones...
	if cronJob.Spec.ConcurrencyPolicy == batchv1.ReplaceConcurrent {
		for _, activeJob := range activeJobs {
			// we don't care if the job was already deleted
			if err := r.Delete(ctx, activeJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete active job", "job", activeJob)
				return ctrl.Result{}, err
			}
		}
	}

	/*
		Once we've figured out what to do with existing jobs, we'll actually create our desired job
	*/

	// +kubebuilder:docs-gen:collapse=constructJobForCronJob

	// actually make the job...
	job, err := job.ConstructJobForCronJob(&cronJob, missedRun, r.Scheme)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// don't bother requeuing until we get a change to the spec
		return scheduledResult, nil
	}

	// ...and create it on the cluster
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for CronJob", "job", job)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created Job for CronJob run", "job", job)

	/*
		### 7: Requeue when we either see a running job or it's time for the next scheduled run

		Finally, we'll return the result that we prepped above, that says we want to requeue
		when our next run would need to occur.  This is taken as a maximum deadline -- if something
		else changes in between, like our job starts or finishes, we get modified, etc, we might
		reconcile again sooner.
	*/
	// we'll requeue once we see the running job, and update our status
	return scheduledResult, nil
}

/*
### Setup

Finally, we'll update our setup.

Additionally, we'll inform the manager that this controller owns some Jobs, so that it
will automatically call Reconcile on the underlying CronJob when a Job changes, is
deleted, etc.
*/

func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// set up a real clock, since we're not in a test
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	if err := job.SetupJobIndexWithManager(mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Owns(&kbatch.Job{}).
		Complete(r)
}
