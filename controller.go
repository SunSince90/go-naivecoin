package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func handleCreateEvent(pod *corev1.Pod, myself string) {
	l := log.With().Str("name", pod.Name).Str("ip", pod.Status.PodIP).Logger()
	if pod.Name == myself {
		l.Info().Msg("ignoring because it is me")
		return
	}

	if val, exists := pod.Labels["app"]; !exists || val != "go-naivecoin" {
		l.Info().Msg("ignoring because it is not related to go-naivecoin")
		return
	}

	if pod.Status.Phase == corev1.PodRunning {
		// TODO: ip or something else?
		l.Info().Msg("found a new peer")

		// TODO: peers should be a struct, so that we can establish a
		// connection with this peer when it is added.
		peers[pod.Name] = pod.Status.PodIP
	}
}

func handleUpdateFunc(curr, prev *corev1.Pod, myself string) {
	l := log.With().Str("name", curr.Name).Str("ip", curr.Status.PodIP).Logger()
	if curr.Name == myself {
		l.Info().Msg("ignoring because it is me")
		return
	}

	if val, exists := curr.Labels["app"]; !exists || val != "go-naivecoin" {
		l.Info().Msg("ignoring because it is not related to go-naivecoin")
		return
	}

	if len(curr.Status.PodIP) == 0 {
		l.Info().Msg("skipping: pod does not have an IP")
		return
	}

	if curr.Status.Phase == prev.Status.Phase {
		log.Info().Msg("same status as before")
		return
	}

	if curr.Status.Phase == corev1.PodRunning {
		l.Info().Msg("found a new running peer")
		peers[curr.Name] = curr.Status.PodIP
	} else {
		l.Info().Msg("found a not running peer")
		delete(peers, curr.Name)
	}
}

func handleDeleteFunc(pod *corev1.Pod) {
	delete(peers, pod.Name)
}

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

// SetPodController sets the pod controller to the provided manager.
//
// The second parameter is the name of the running pod, so that the pod
// controller can correctly avoid parsing events about itself.
func SetPodController(mgr manager.Manager, myself string) (controller.Controller, error) {
	c, err := controller.New("pod-controller", mgr, controller.Options{
		Reconciler: reconcile.Func(func(context.Context, reconcile.Request) (reconcile.Result, error) {
			return reconcile.Result{}, nil
		}),
	})
	if err != nil {
		return nil, err
	}

	// Watch for Pod create / update / delete events and call Reconcile
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.Funcs{
		CreateFunc: func(ce event.CreateEvent, rli workqueue.RateLimitingInterface) {
			if pod, ok := ce.Object.(*corev1.Pod); ok {
				handleCreateEvent(pod, myself)
			}
		},
		UpdateFunc: func(ue event.UpdateEvent, rli workqueue.RateLimitingInterface) {
			newPod, newOK := ue.ObjectNew.(*corev1.Pod)
			oldPod, oldOK := ue.ObjectOld.(*corev1.Pod)

			if newOK && oldOK {
				handleUpdateFunc(newPod, oldPod, myself)
			}
		},
		DeleteFunc: func(de event.DeleteEvent, rli workqueue.RateLimitingInterface) {
			if pod, ok := de.Object.(*corev1.Pod); ok {
				handleDeleteFunc(pod)
			}
		},
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}
