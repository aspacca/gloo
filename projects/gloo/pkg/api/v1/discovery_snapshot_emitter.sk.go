// Code generated by solo-kit. DO NOT EDIT.

package v1

import (
	"sync"
	"time"

	github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes "github.com/solo-io/solo-kit/pkg/api/v1/resources/common/kubernetes"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/solo-io/go-utils/errutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/errors"
)

var (
	mDiscoverySnapshotIn  = stats.Int64("discovery.gloo.solo.io/snap_emitter/snap_in", "The number of snapshots in", "1")
	mDiscoverySnapshotOut = stats.Int64("discovery.gloo.solo.io/snap_emitter/snap_out", "The number of snapshots out", "1")

	discoverysnapshotInView = &view.View{
		Name:        "discovery.gloo.solo.io_snap_emitter/snap_in",
		Measure:     mDiscoverySnapshotIn,
		Description: "The number of snapshots updates coming in",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{},
	}
	discoverysnapshotOutView = &view.View{
		Name:        "discovery.gloo.solo.io/snap_emitter/snap_out",
		Measure:     mDiscoverySnapshotOut,
		Description: "The number of snapshots updates going out",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{},
	}
)

func init() {
	view.Register(discoverysnapshotInView, discoverysnapshotOutView)
}

type DiscoveryEmitter interface {
	Register() error
	Upstream() UpstreamClient
	KubeNamespace() github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes.KubeNamespaceClient
	Secret() SecretClient
	Snapshots(watchNamespaces []string, opts clients.WatchOpts) (<-chan *DiscoverySnapshot, <-chan error, error)
}

func NewDiscoveryEmitter(upstreamClient UpstreamClient, kubeNamespaceClient github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes.KubeNamespaceClient, secretClient SecretClient) DiscoveryEmitter {
	return NewDiscoveryEmitterWithEmit(upstreamClient, kubeNamespaceClient, secretClient, make(chan struct{}))
}

func NewDiscoveryEmitterWithEmit(upstreamClient UpstreamClient, kubeNamespaceClient github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes.KubeNamespaceClient, secretClient SecretClient, emit <-chan struct{}) DiscoveryEmitter {
	return &discoveryEmitter{
		upstream:      upstreamClient,
		kubeNamespace: kubeNamespaceClient,
		secret:        secretClient,
		forceEmit:     emit,
	}
}

type discoveryEmitter struct {
	forceEmit     <-chan struct{}
	upstream      UpstreamClient
	kubeNamespace github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes.KubeNamespaceClient
	secret        SecretClient
}

func (c *discoveryEmitter) Register() error {
	if err := c.upstream.Register(); err != nil {
		return err
	}
	if err := c.kubeNamespace.Register(); err != nil {
		return err
	}
	if err := c.secret.Register(); err != nil {
		return err
	}
	return nil
}

func (c *discoveryEmitter) Upstream() UpstreamClient {
	return c.upstream
}

func (c *discoveryEmitter) KubeNamespace() github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes.KubeNamespaceClient {
	return c.kubeNamespace
}

func (c *discoveryEmitter) Secret() SecretClient {
	return c.secret
}

func (c *discoveryEmitter) Snapshots(watchNamespaces []string, opts clients.WatchOpts) (<-chan *DiscoverySnapshot, <-chan error, error) {

	if len(watchNamespaces) == 0 {
		watchNamespaces = []string{""}
	}

	for _, ns := range watchNamespaces {
		if ns == "" && len(watchNamespaces) > 1 {
			return nil, nil, errors.Errorf("the \"\" namespace is used to watch all namespaces. Snapshots can either be tracked for " +
				"specific namespaces or \"\" AllNamespaces, but not both.")
		}
	}

	errs := make(chan error)
	var done sync.WaitGroup
	ctx := opts.Ctx
	/* Create channel for Upstream */
	type upstreamListWithNamespace struct {
		list      UpstreamList
		namespace string
	}
	upstreamChan := make(chan upstreamListWithNamespace)
	/* Create channel for KubeNamespace */
	type kubeNamespaceListWithNamespace struct {
		list      github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes.KubeNamespaceList
		namespace string
	}
	kubeNamespaceChan := make(chan kubeNamespaceListWithNamespace)
	/* Create channel for Secret */
	type secretListWithNamespace struct {
		list      SecretList
		namespace string
	}
	secretChan := make(chan secretListWithNamespace)

	for _, namespace := range watchNamespaces {
		/* Setup namespaced watch for Upstream */
		upstreamNamespacesChan, upstreamErrs, err := c.upstream.Watch(namespace, opts)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "starting Upstream watch")
		}

		done.Add(1)
		go func(namespace string) {
			defer done.Done()
			errutils.AggregateErrs(ctx, errs, upstreamErrs, namespace+"-upstreams")
		}(namespace)
		/* Setup namespaced watch for KubeNamespace */
		kubeNamespaceNamespacesChan, kubeNamespaceErrs, err := c.kubeNamespace.Watch(namespace, opts)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "starting KubeNamespace watch")
		}

		done.Add(1)
		go func(namespace string) {
			defer done.Done()
			errutils.AggregateErrs(ctx, errs, kubeNamespaceErrs, namespace+"-kubenamespaces")
		}(namespace)
		/* Setup namespaced watch for Secret */
		secretNamespacesChan, secretErrs, err := c.secret.Watch(namespace, opts)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "starting Secret watch")
		}

		done.Add(1)
		go func(namespace string) {
			defer done.Done()
			errutils.AggregateErrs(ctx, errs, secretErrs, namespace+"-secrets")
		}(namespace)

		/* Watch for changes and update snapshot */
		go func(namespace string) {
			for {
				select {
				case <-ctx.Done():
					return
				case upstreamList := <-upstreamNamespacesChan:
					select {
					case <-ctx.Done():
						return
					case upstreamChan <- upstreamListWithNamespace{list: upstreamList, namespace: namespace}:
					}
				case kubeNamespaceList := <-kubeNamespaceNamespacesChan:
					select {
					case <-ctx.Done():
						return
					case kubeNamespaceChan <- kubeNamespaceListWithNamespace{list: kubeNamespaceList, namespace: namespace}:
					}
				case secretList := <-secretNamespacesChan:
					select {
					case <-ctx.Done():
						return
					case secretChan <- secretListWithNamespace{list: secretList, namespace: namespace}:
					}
				}
			}
		}(namespace)
	}

	snapshots := make(chan *DiscoverySnapshot)
	go func() {
		originalSnapshot := DiscoverySnapshot{}
		currentSnapshot := originalSnapshot.Clone()
		timer := time.NewTicker(time.Second * 1)
		sync := func() {
			if originalSnapshot.Hash() == currentSnapshot.Hash() {
				return
			}

			stats.Record(ctx, mDiscoverySnapshotOut.M(1))
			originalSnapshot = currentSnapshot.Clone()
			sentSnapshot := currentSnapshot.Clone()
			snapshots <- &sentSnapshot
		}
		upstreamsByNamespace := make(map[string]UpstreamList)
		kubenamespacesByNamespace := make(map[string]github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes.KubeNamespaceList)
		secretsByNamespace := make(map[string]SecretList)

		for {
			record := func() { stats.Record(ctx, mDiscoverySnapshotIn.M(1)) }

			select {
			case <-timer.C:
				sync()
			case <-ctx.Done():
				close(snapshots)
				done.Wait()
				close(errs)
				return
			case <-c.forceEmit:
				sentSnapshot := currentSnapshot.Clone()
				snapshots <- &sentSnapshot
			case upstreamNamespacedList := <-upstreamChan:
				record()

				namespace := upstreamNamespacedList.namespace

				// merge lists by namespace
				upstreamsByNamespace[namespace] = upstreamNamespacedList.list
				var upstreamList UpstreamList
				for _, upstreams := range upstreamsByNamespace {
					upstreamList = append(upstreamList, upstreams...)
				}
				currentSnapshot.Upstreams = upstreamList.Sort()
			case kubeNamespaceNamespacedList := <-kubeNamespaceChan:
				record()

				namespace := kubeNamespaceNamespacedList.namespace

				// merge lists by namespace
				kubenamespacesByNamespace[namespace] = kubeNamespaceNamespacedList.list
				var kubeNamespaceList github_com_solo_io_solo_kit_pkg_api_v1_resources_common_kubernetes.KubeNamespaceList
				for _, kubenamespaces := range kubenamespacesByNamespace {
					kubeNamespaceList = append(kubeNamespaceList, kubenamespaces...)
				}
				currentSnapshot.Kubenamespaces = kubeNamespaceList.Sort()
			case secretNamespacedList := <-secretChan:
				record()

				namespace := secretNamespacedList.namespace

				// merge lists by namespace
				secretsByNamespace[namespace] = secretNamespacedList.list
				var secretList SecretList
				for _, secrets := range secretsByNamespace {
					secretList = append(secretList, secrets...)
				}
				currentSnapshot.Secrets = secretList.Sort()
			}
		}
	}()
	return snapshots, errs, nil
}
