/*
Copyright 2022 The KCP Authors.

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

package resourcequota

func (rq *Controller) Sync(ctx context.Context, discoveryFunc NamespacedResourcesFunc, period time.Duration) {
	// Something has changed, so track the new state and perform a sync.
	oldResources := make(map[schema.GroupVersionResource]struct{})
	wait.UntilWithContext(ctx, func(ctx context.Context) {
		// Get the current resource list from discovery.
		newResources, err := GetQuotableResources(discoveryFunc)
		if err != nil {
			utilruntime.HandleError(err)

			if groupLookupFailures, isLookupFailure := discovery.GroupDiscoveryFailedErrorGroups(err); isLookupFailure && len(newResources) > 0 {
				// In partial discovery cases, preserve existing informers for resources in the failed groups, so resyncMonitors will only add informers for newly seen resources
				for k, v := range oldResources {
					if _, failed := groupLookupFailures[k.GroupVersion()]; failed {
						newResources[k] = v
					}
				}
			} else {
				// short circuit in non-discovery error cases or if discovery returned zero resources
				return
			}
		}

		logger := klog.FromContext(ctx)

		// Decide whether discovery has reported a change.
		if reflect.DeepEqual(oldResources, newResources) {
			logger.V(4).Info("no resource updates from discovery, skipping resource quota sync")
			return
		}

		// Ensure workers are paused to avoid processing events before informers
		// have resynced.
		rq.workerLock.Lock()
		defer rq.workerLock.Unlock()

		// Something has changed, so track the new state and perform a sync.
		if loggerV := logger.V(2); loggerV.Enabled() {
			loggerV.Info("syncing resource quota controller with updated resources from discovery", "diff", printDiff(oldResources, newResources))
		}

		// Perform the monitor resync and wait for controllers to report cache sync.
		if err := rq.resyncMonitors(ctx, newResources); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to sync resource monitors: %v", err))
			return
		}

		// at this point, we've synced the new resources to our monitors, so record that fact.
		oldResources = newResources

		// wait for caches to fill for a while (our sync period).
		// this protects us from deadlocks where available resources changed and one of our informer caches will never fill.
		// informers keep attempting to sync in the background, so retrying doesn't interrupt them.
		// the call to resyncMonitors on the reattempt will no-op for resources that still exist.
		if rq.quotaMonitor != nil &&
			!cache.WaitForNamedCacheSync(
				"resource quota",
				waitForStopOrTimeout(ctx.Done(), period),
				func() bool { return rq.quotaMonitor.IsSynced(ctx) },
			) {
			utilruntime.HandleError(fmt.Errorf("timed out waiting for quota monitor sync"))
			return
		}

		logger.V(2).Info("synced quota controller")
	}, period)
}
