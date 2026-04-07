# Proto and API versioning (v3 vs v4)

## Where the protos live

- **v3:** Canonical file is `src/proto/openclaude.proto` in the [openclaude](https://github.com/Gitlawb/openclaude) repository, with **`package openclaude.v1`** and service **`AgentService`** (bidi `Chat`).
- **v4:** Service definition and a v3→v4 **conceptual mapping table** live in this repo under [`internal/grpc/proto/openclaude.proto`](../internal/grpc/proto/openclaude.proto) and [`internal/grpc/README.md`](../internal/grpc/README.md). The v4 package is **`openclaude.v4`** so generated clients and service names do not collide with v3.

## Compatibility policy

- **Major package version** (`openclaude.v1` vs `openclaude.v2`, etc.) is the primary **compatibility boundary** for wire and generated stubs.
- v4 does **not** guarantee that v3 gRPC clients work against a v4 server without a documented adapter or compatibility gateway.
- Prefer **additive** changes (new optional fields, new RPCs) within a major version; document **breaking** changes in release notes when bumping the package or removing fields.

## Practical guidance for client authors

- Generate stubs from the **same** `.proto` (and plugin versions) as the server you target.
- For migrating from v3 to v4, start from the mapping table in [`internal/grpc/README.md`](../internal/grpc/README.md) and [MIGRATION_V3.md](./MIGRATION_V3.md).
