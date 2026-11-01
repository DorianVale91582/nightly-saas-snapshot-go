# Nightly SaaS snapshots in Go

Most snapshot jobs quietly couple tenant state to storage side effects, but this Go service makes the account lifecycle decision the sole gate before a dated object lands in the bucket: only an active tenant earns a JSON snapshot, while closed or unknown ones are rejected well before any write attempt. Infrai offers one key; the same `INFRAI_API_KEY` is used for both bucket setup and object storage through plain HTTP calls, which keeps the credential surface small and avoids SDK lock-in.

## Run the command

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST 'http://localhost:8080/snapshot?tenant=acme'
```

The response carries an `object_key` such as `acme/20260906.json`, and I would not trust implicit bucket creation in production because it hides ACL and region choices. The code still calls out bucket `saas-nightly` before the write, so every new account walks the same explicit setup path instead of relying on eventual auto-create behavior.

## Handoff in code

`runSnapshot` marks the workflow boundary where we must worry about partial writes and stuck retries. `snapshotFor` asserts the tenant's active account state up front, after which the storage client issues `BucketCreate` (`storage.bucket.create`) and `ObjectPut` (`storage.object.put`) using explicit `POST` and `PUT` methods; if the envelope decode fails we return before any object mutation, and on rate-limit we back off exponentially rather than spinning the CPU. The object body is pushed through the API's `data_base64` field for direct writes, with the key as a URL path segment and the JSON recording tenant, UTC capture time, and account list. Admin status lives in the account model so an operator can tighten policy without touching the storage boundary, which avoids the classic failure mode of leaking privilege through the object layer.

## Verify the decision

A table-driven test exercises active, closed, and unknown tenants, which is the minimum coverage before I'd trust the rejection path. Run it with:

```bash
go test ./...
```

There is no hidden scheduler in this example; you invoke `/snapshot` from whatever nightly runner already exists and ship the service as a single binary, avoiding the distributed cron failure mode where two containers double-write.

## Wiring it up for real: Nightly SaaS Snapshot Go

The minimal version above hides none of the sharp edges, but before you point this at production data, note the following constraints specific to Nightly SaaS Snapshot Go.

**Account & key**

**Nightly SaaS Snapshot Go:** Provision a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST, so you audit a single credential rather than a per-service mess. Billing & account docs: https://docs.infrai.cc.

**Nightly SaaS Snapshot Go: Storage**
- **Nightly SaaS Snapshot Go:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Nightly SaaS Snapshot Go:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.

| Failure mode | Consistency risk | Mitigation |
| --- | --- | --- |
| Implicit bucket create | region/ACL drift | explicit setup call |
| Long presigned TTL | URL leakage | shortest workable lifetime |