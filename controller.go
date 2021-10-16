package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// GetControllerManager creates the pod controller and returns its manager so
// that it could be started.
func GetControllerManager(namespace string) (manager.Manager, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return nil, err
	}

	return mgr, nil
}

// PodEventHandler is a structure that handles event related to pods.
type PodEventHandler struct {
	myself string
}

// NewPodEventHandler returns a new PodEventHandler.
func NewPodEventHandler(myself string) *PodEventHandler {
	return &PodEventHandler{
		myself: myself,
	}
}

// Create handles pod Create events.
func (p *PodEventHandler) Create(ce event.CreateEvent, _ workqueue.RateLimitingInterface) {
	pod, ok := ce.Object.(*corev1.Pod)
	if !ok {
		// TODO: log this
		log.Error().Str("event", "Create").Msg("skipping: could not successfully parse event")
		return
	}

	l := log.With().Str("name", pod.Name).Str("ip", pod.Status.PodIP).Logger()
	if pod.Name == p.myself {
		l.Info().Msg("ignoring because it is me")
		return
	}

	if val, exists := pod.Labels["app"]; !exists || val != "go-naivecoin" {
		l.Info().Msg("ignoring because it is not related to go-naivecoin")
		return
	}

	if pod.Status.Phase != corev1.PodRunning {
		// TODO: only verbose
		l.Info().Msg("ignoring because it is not related to go-naivecoin")
		return
	}

	l.Info().Msg("found a new peer")

	// TODO: send event
}

// Update handes Update events.
func (p *PodEventHandler) Update(ue event.UpdateEvent, _ workqueue.RateLimitingInterface) {
	currPod, currOk := ue.ObjectNew.(*corev1.Pod)
	prevPod, prevOk := ue.ObjectOld.(*corev1.Pod)

	if !currOk || !prevOk {
		log.Error().Str("event", "Update").Msg("skipping: could not parse the events")
		return
	}

	l := log.With().Str("name", currPod.Name).Str("ip", currPod.Status.PodIP).Logger()
	if currPod.Name == p.myself {
		l.Info().Msg("ignoring because it is me")
		return
	}

	if val, exists := currPod.Labels["app"]; !exists || val != "go-naivecoin" {
		l.Info().Msg("ignoring because it is not related to go-naivecoin")
		return
	}

	if len(currPod.Status.PodIP) == 0 {
		l.Info().Msg("skipping: pod does not have an IP")
		return
	}

	if currPod.DeletionTimestamp != nil {
		// This is the way to know when a resource is being deleted.
		// TODO: delete the pod
	}

	if currPod.Status.Phase == prevPod.Status.Phase {
		log.Info().Msg("same status as before")
		return
	}

	if currPod.Status.Phase == corev1.PodRunning {
		l.Info().Msg("found a new running peer")
	} else {
		l.Info().Msg("found a not running peer")
	}

	// TODO: add event
}

// Delete handles pod Delete events.
func (p *PodEventHandler) Delete(de event.DeleteEvent, _ workqueue.RateLimitingInterface) {
	pod, ok := de.Object.(*corev1.Pod)
	if !ok {
		// TODO: log this
		log.Error().Str("event", "Delete").Msg("skipping: could not successfully parse event")
		return
	}

	log.Info().Msg("found dead peer")

	// TODO: add event
	_ = pod
}

// Generic handles pod events that are neither Create, Update or Delete.
func (p *PodEventHandler) Generic(ge event.GenericEvent, _ workqueue.RateLimitingInterface) {
	log.Info().Msg("skipping unknown event")
}

// SetPodController sets the pod controller to the provided manager.
//
// The second parameter is the name of the running pod, so that the pod
// controller can correctly avoid parsing events about itself.
func SetPodController(mgr manager.Manager, podEH *PodEventHandler) (controller.Controller, error) {
	// podController := NewPod
	c, err := controller.New("pod-controller", mgr, controller.Options{
		Reconciler: reconcile.Func(func(context.Context, reconcile.Request) (reconcile.Result, error) {
			return reconcile.Result{}, nil
		}),
	})
	if err != nil {
		return nil, err
	}

	// Watch for Pod create / update / delete events and call Reconcile
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, podEH)
	if err != nil {
		return nil, err
	}

	return c, nil
}
