  Precedent PRs for exposing KeyFunction in client-go tools/cache

  Direct prior attempts (CLOSED, NOT MERGED)

  Two prior PRs attempted essentially the same change. Understanding why they were rejected is critical for framing yours differently:

  1. kubernetes/kubernetes#114321 — "Allow to create shared index informers with custom key functions" (2022, by @vincepri / Cluster API)
    - Proposed adding custom KeyFunc to SharedIndexInformer for multi-cluster controller-runtime use cases.
    - Rejected by @deads2k, @lavalamp, @liggitt with objections: (a) k/k and k/client-go are single-cluster focused, multi-cluster is an anti-goal; (b) making the key function indeterminate breaks the assumption
  that namespace/name uniquely identify objects, creating potential security/correctness issues; (c) Replace() semantics are unclear for multi-cluster caches; (d) need a cohesive design first, not a piecemeal
  change.
  2. kubernetes/kubernetes#130002 — "client-go: enable KeyFunction configuration in SharedIndexInformerOptions" (2025, by @xigang)
    - Second attempt. Got triage/accepted and priority/awaiting-more-evidence labels but never received lgtm/approval. Closed without merge.

  Merged PRs that established the Options pattern and exposed new fields

  These are the strongest precedents — each added a new configurable field to the same option structs your PR modifies:

  3. kubernetes/kubernetes#111898 — "Reflector: support logging Unstructured type" (merged 2022-12-09)
    - Created SharedIndexInformerOptions, ReflectorOptions, NewSharedIndexInformerWithOptions(), and NewReflectorWithOptions(). Added ObjectDescription field so out-of-tree consumers could provide meaningful type
   descriptions for Unstructured informers.
    - Why it fits: Established the exact options structs your PR extends.
  4. kubernetes/kubernetes#124245 — "Allow for configuring MinWatchTimeout in reflector" (merged 2024-04-18)
    - Added MinWatchTimeout to ReflectorOptions, Config, and created InformerOptions with NewInformerWithOptions(). Motivated by kubelet needing longer watch timeouts.
    - Why it fits: Added a new field to ReflectorOptions and Config and created InformerOptions — the same structs your PR extends, for the same reason (out-of-tree configurability).
  5. kubernetes/kubernetes#94363 — "Add WatchListPageSize to cache.Config" (merged 2020-09-07)
    - Added WatchListPageSize to cache.Config for external configurability of the initial/relist page size.
    - Why it fits: Extended Config with a previously-internal setting for out-of-tree use.
  6. kubernetes/kubernetes#136824 — "Replace deprecated BackoffManager with DelayFunc in Reflector" (merged 2026-02-13)
    - Added Backoff field to ReflectorOptions for customization of reflector backoff strategy.
    - Why it fits: Extended ReflectorOptions with a new optional field, same pattern as adding KeyFunction.
  7. kubernetes/kubernetes#126387 — "client-go/tools/cache: add APIs with context parameter" (merged 2024-12-18)
    - Added Logger field to ReflectorOptions and InformerOptions for contextual logging.
    - Why it fits: Extended the same option structs with a new field for out-of-tree configurability.
  8. kubernetes/kubernetes#135782 — "Add identifier-based queue depth metrics for RealFIFO" (merged 2026-02-06)
    - Added Identifier and InformerMetricsProvider fields to SharedIndexInformerOptions and InformerOptions.
    - Why it fits: Extended SharedIndexInformerOptions with new fields, exactly as your PR does.

  Merged PRs that exposed KeyFunc as a configurable parameter in cache primitives

  These show that KeyFunc has been an explicit configuration point in lower-level cache primitives for years:

  9. kubernetes/kubernetes#86015 — "informers: Don't treat relist same as sync" (merged 2020-01-24)
    - Introduced DeltaFIFOOptions with KeyFunction as an explicit option field, deprecating the old positional NewDeltaFIFO(keyFunc, knownObjects) constructor.
    - Why it fits: The most direct precedent — DeltaFIFOOptions.KeyFunction is configurable, but the informer layer above never plumbed it through. Your PR closes this gap.
  10. kubernetes/kubernetes#100355 — "Replace deprecated NewDeltaFIFO with NewDeltaFIFOWithOptions" (merged 2021-04-09)
    - Migrated all callers to the options-based DeltaFIFOOptions{KeyFunction: ...} pattern.
    - Why it fits: Codified the pattern that KeyFunction is an option, not a hardcoded default.
  11. kubernetes/kubernetes#132243 — "Add RealFIFOOptions struct" (merged 2025-10-21)
    - Introduced RealFIFOOptions with KeyFunction as a field, following the same WithOptions pattern.
    - Why it fits: The newer RealFIFO also takes KeyFunction as an option — your PR ensures the informer layer can actually pass a non-default value down to it.

  Merged PRs that exposed internal informer capabilities for out-of-tree use

  12. kubernetes/kubernetes#107507 — "Add configuration point to SharedInformer to transform objects before storing" (merged 2022-01-24)
    - Added SetTransform to SharedInformer, allowing out-of-tree consumers (controller-runtime) to strip fields like managedFields to reduce memory.
    - Why it fits: Closest precedent in spirit. Added a new customization hook to SharedInformer for an out-of-tree use case (controller-runtime), modifying internal behavior (what gets stored). Vincepri
  explicitly cited this PR as precedent in #114321.
  13. kubernetes/kubernetes#104300 — "Create TransformingInformer" (merged 2022-01-11)
    - Added TransformingInformer to client-go with custom transform functions. Pattern originated in CoreDNS.
    - Why it fits: Exposed a new customization point in the informer for out-of-tree consumers.
  14. kubernetes/kubernetes#117046 — "Allow adding indexes after informer starts" (merged 2023-12-13)
    - Removed the restriction that indexes can only be added before informer start.
    - Why it fits: Loosened an internal informer constraint for the benefit of out-of-tree consumers.
  15. kubernetes/kubernetes#45946 — "Expose informer constructors" (merged 2017-07-25)
    - Exported informer constructors for use by out-of-tree components.
    - Why it fits: Early precedent for making client-go internals accessible to external projects.

  Historical: KeyFunc as a foundational concept

  16. kubernetes/kubernetes#3810 — "Support namespacing in cache.Store" (merged 2015-01-30)
    - Original PR introducing KeyFunc as a parameter to cache.Store. The foundational design decision.
  17. kubernetes/kubernetes#26854 — "cacher.go: remove NewCacher func" (merged 2016-07-05)
    - Moved KeyFunc construction responsibility to callers rather than having the cacher implicitly choose one. Established the principle that callers should decide their key function.

  ---
  Key arguments for your PR based on precedents

  1. The gap is already there: DeltaFIFOOptions.KeyFunction (#86015) and RealFIFOOptions.KeyFunction (#132243) already accept custom key functions. The informer/reflector layer just never plumbs them through.
  Your PR closes an existing inconsistency.
  2. The Options pattern is well-established: PRs #111898, #124245, #126387, #135782, and #136824 all added fields to the exact same structs (SharedIndexInformerOptions, ReflectorOptions, InformerOptions,
  Config). Your change follows the identical pattern.
  3. Transform was the same kind of change: #107507 (SetTransform) added a hook that changes what gets stored in the informer cache, for controller-runtime's benefit. KeyFunction changes how objects are keyed — a
   smaller surface area change.
  4. Addressing previous objections: The prior rejections (#114321, #130002) were primarily about the multi-cluster framing and lack of cohesive design. If your PR can frame the use case differently (e.g.,
  general-purpose extensibility, consistency with DeltaFIFO, single-cluster use cases that need custom keys) and address the Replace/relist semantics concern, it has a stronger path forward.

