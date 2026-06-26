---
title: Response bodies
weight: 50
description: |
  Describe a concrete response payload without a dedicated swagger:response
  struct — declare the body inline on the route, or shadow a generic envelope's
  payload with a doc-only struct.
---

When a handler's actual Go return type doesn't map cleanly to the payload you
want documented, these knobs let you pin the response body the spec describes —
inline on the route, or via a doc-only struct that stands in for a generic
envelope.

{{< children type="card" description="true" >}}
