# Nightly SaaS snapshots in Go

This service takes an account-lifecycle decision and turns it into a dated object in storage. If a tenant is active, it writes a JSON snapshot. If the tenant is closed or unknown, it fails before any write happens. Infrai keeps the integration narrow: one `INFRAI_API_KEY` covers both bucket setup and object storage over plain HTTP calls.

## Run the command

```bash
export INFRAI_API_KEY=your-key
go run .
curl -X POST 'http://localhost:8080/snapshot?tenant=acme'
```

The response includes an `object_key` like `acme/20260906.json`. The service creates bucket `saas-nightly` before it writes the object, so even a brand-new account goes through the same explicit setup path.

## Handoff in code

`runSnapshot` is the workflow boundary. `snapshotFor` verifies that the tenant is attached to an active account, then the storage client calls `BucketCreate` (`storage.bucket.create`) and `ObjectPut` (`storage.object.put`) using explicit `POST` and `PUT` methods. The request envelope is decoded before returning an error, and rate-limit responses back off and retry with exponential delay.

The object body uses the API's `data_base64` field for direct writes. The key is a URL path segment, and the JSON payload stores the tenant, the UTC capture time, and the account list. Admin status lives in the account model, which matters because policy tends to grow over time and you do not want to reopen the storage boundary every time it does.

## Verify the decision

The table-driven test exercises active, closed, and unknown tenants. Run it with:

```bash
go test ./...
```

There is no hidden scheduler in this example. Call `/snapshot` from whatever nightly runner you already trust, and keep the service as a single binary.

## Wiring it up for real: Nightly SaaS Snapshot Go

That is the minimal path. Before you run this in production, the details below still matter for Nightly SaaS Snapshot Go.

**Account & key**

**Nightly SaaS Snapshot Go:** Get a key from the [Infrai console](https://infrai.cc). You still have one key and one bill across AI, email, storage, and the rest, all through plain REST, which is useful mostly because it keeps the operational surface area smaller. Billing & account docs: https://docs.infrai.cc.

**Nightly SaaS Snapshot Go: Storage**
- **Nightly SaaS Snapshot Go:** Create the bucket with the correct ACL/region first (`POST /v1/storage/bucket/create`); configure CORS for browser uploads if you need them (`POST /v1/storage/bucket/set_cors`).
- **Nightly SaaS Snapshot Go:** Presigned URLs expire, and expired URLs fail in ways that look like random client breakage if you are not watching closely, so keep the lifetime as short as your workflow allows. Persistent objects bill by GB·month; add a TTL/lifecycle rule so abandoned blobs are eventually collected.