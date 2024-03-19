# package `tagger`

The **Tagger** component is the central source of truth for client-side entity tagging.
It connects to **WorkloadMeta** that detect entities and collect their tags.
Tags are then stored in memory (by the **TagStore**) and can be queried by the `tagger.Tag()` method.
Calling once `tagger.Init()` after the **config** package is ready is needed to enable collection.

The package methods use a common `defaultTagger` object, but we can create a custom **Tagger** object for testing.

The package will implement an IPC mechanism (a server and a client) to allow
other agents to query the **DefaultTagger** and avoid duplicating the information
in their process. Switch between local and client mode will be done via a build flag.

The tagger is also available to python checks via the `tagger` module exporting
the `get_tags()` function. This function accepts the same arguments as the Go `Tag()`
function, and returns an empty list on errors.

## Workloadmeta

The **Tagger** subscribe to Workloadmeta as its single information source and store **TagInfo** in the **TagStore**.
The **Tagger** will stream data from Workloadmeta and update the **TagStore** incrementally.

## TagStore

The **TagStore** reads **TagInfo** structs and stores them in a in-memory
cache. Cache invalidation is triggered by a TTL mechanism, and by the **Tagger**
when it receives a **TagInfo** with **DeleteEntity** set.
# TODO Mot sure about the all DeleteEntity thing.

* sending new tags for the same `Entity`, all the tags from this `Source`
  will be removed and replaced by the new tags
* sending a **TagInfo** with **DeleteEntity** set, all the tags collected for
  this entity by the specified source (but not others) will be deleted when
  **prune()** is called.

## TagCardinality

**TagInfo** accepts and store tags that have different cardinality. **TagCardinality** can be:

# TODO What about standard cardinality?
* **LowCardinality**: in the host count order of magnitude.
* **OrchestratorCardinality**: tags that change value for each pod or task.
* **HighCardinality**: typically tags that change value for each web request, user agent, container, etc.

## Entity IDs

Tagger entities are identified by a string-typed ID, with one of the following forms:

<!-- NOTE: a similar table appears in comp/core/autodiscovery/README.md; please keep both in sync -->
| *Service*                               | *Tagger Entity*                                                    |
|-----------------------------------------|--------------------------------------------------------------------|
| workloadmeta.KindContainer              | `container_id://<sha>`                                             |
| workloadmeta.KindContainerImageMetadata | `container_image_metadata://<sha>`                                 |
| workloadmeta.KindGardenContainer        | `container_id://<sha>`                                             |
| workloadmeta.KindKubernetesPod          | `kubernetes_pod_uid://<uid>`                                       |
| workloadmeta.KindECSTask                | `ecs_task://<task-id>`                                             |
| CloudFoundry LRP                        | `<processGuid>/<svcName>/<instanceGuid>`  or `<appGuid>/<svcName>` |
| Container runtime or orchestrator       | (none)                                                             |
| Kubernetes Endpoint                     | `kube_endpoint_uid://<namespace>/<name>/<ip>`                      |
| Kubernetes Service                      | `kube_service://<namespace>/<name>`                                |
| SNMP Config                             | config hash                                                        |

## Tagger

The Tagger handles the glue between **Workloadmeta**, the **TagStore** and the
cache miss logic. If the tags from the **TagStore** are missing some sources,
they will be manually queried in a block way, and the cache will be updated.

For convenience, the package creates a **defaultTagger** object that is used
when calling the `tagger.Tag()` method.
# TODO What is globalTagger vs defaultTagger?

                   +--------------+
                   | Workloadmeta |
                   +-----+--------+
                         |
                         |
    +--------+      +----+-----+       +-------------+
    |  User  <------+  Tagger  +-------> IPC handler |
    |packages|      +--+-----^-+       +-------------+
    +--------+         |     |
                       |     |
                    +--v-----+-+
                    | TagStore |
                    +----------+
