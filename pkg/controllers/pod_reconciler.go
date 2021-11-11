package controllers

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/SunSince90/go-naivecoin/pkg/peers"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type PodReconciler struct {
	client.Client
	myself     string
	peerEvents chan *peers.PeerEvent
	lock       sync.Mutex
}

func NewPodReconciler(mgr manager.Manager, peerEvents chan *peers.PeerEvent) (*PodReconciler, error) {
	myself := os.Getenv("NAME")
	if myself == "" {
		return nil, fmt.Errorf("could not retrieve pod name")
	}

	if mgr == nil {
		return nil, fmt.Errorf("nil manager provided")
	}

	pr := &PodReconciler{
		Client:     mgr.GetClient(),
		myself:     myself,
		peerEvents: peerEvents,
		lock:       sync.Mutex{},
	}

	c, err := controller.New("pod-controller", mgr, controller.Options{
		Reconciler: pr,
	})
	if err != nil {
		return nil, err
	}

	// Watch for Pod create / update / delete events and call Reconcile
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}, pr)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

func (p *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var pod corev1.Pod

	err := p.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			// We remove peers when they are in deletion phase anyways,
			// so we don't really need to do anything here.
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		return ctrl.Result{}, err
	}

	peer := &peers.Peer{
		Name: pod.Name,
		IP:   pod.Status.PodIP,
	}
	eventType := peers.EventNewPeer

	if pod.DeletionTimestamp != nil || pod.Status.Phase != corev1.PodRunning {
		log.Info().Str("peer-name", peer.Name).Str("peer-ip", peer.IP).Msg("removing peer...")
		eventType = peers.EventDeadPeer
	} else {
		log.Info().Str("peer-name", peer.Name).Str("peer-ip", peer.IP).Msg("found new peer")
	}

	p.peerEvents <- &peers.PeerEvent{
		EventType: eventType,
		Peer:      peer,
	}

	return ctrl.Result{}, nil
}

// Create handles pod Create events.
func (p *PodReconciler) Create(ev event.CreateEvent) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	pod, ok := ev.Object.(*corev1.Pod)
	if !ok {
		log.Error().Str("event", "Create").Msg("skipping: could not successfully parse event")
		return false
	}

	if !p.commonPredicates(pod) {
		return false
	}

	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	return true
}

// Update handes Update events.
func (p *PodReconciler) Update(ev event.UpdateEvent) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	currPod, currOk := ev.ObjectNew.(*corev1.Pod)
	prevPod, prevOk := ev.ObjectOld.(*corev1.Pod)

	if !currOk || !prevOk {
		log.Error().Str("event", "Update").Msg("skipping: could not parse the events")
		return false
	}

	if !p.commonPredicates(currPod) {
		return false
	}

	if currPod.DeletionTimestamp != nil {
		// This is the way to know when a resource is being deleted.
		// We're not calling p.Delete because we're holding a lock and
		// we're relasing it with defer. So Delete wouln't be able to get
		// it.
		return prevPod.DeletionTimestamp == nil
	}

	if currPod.Status.Phase == prevPod.Status.Phase {
		return false
	}

	return true
}

// Delete handles pod Delete events.
func (p *PodReconciler) Delete(ev event.DeleteEvent) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	pod, ok := ev.Object.(*corev1.Pod)
	if !ok {
		log.Error().Str("event", "Delete").Msg("skipping: could not successfully parse event")
		return false
	}

	if !p.commonPredicates(pod) {
		return false
	}

	return true
}

func (p *PodReconciler) commonPredicates(pod *corev1.Pod) bool {
	if pod.Name == p.myself {
		return false
	}

	if val, exists := pod.Labels["app"]; !exists || val != "go-naivecoin" {
		return false
	}

	if len(pod.Status.PodIP) == 0 {
		return false
	}

	return true
}

// Generic handles pod events that are neither Create, Update or Delete.
func (p *PodReconciler) Generic(ge event.GenericEvent) bool {
	log.Info().Msg("skipping unknown event")
	return false
}
