# Host PoC

This is a minimal backend proof-of-concept for the planned `Go + Wails` host.

Current scope:

- embed `frp` as a Go library
- keep a clean manager API for `start/stop/status`
- load `EasyTier` through its Rust FFI DLL on Windows
- provide a tiny CLI so the backend can be exercised before any GUI exists

## Build EasyTier FFI

Build the Rust FFI layer before exercising the real EasyTier adapter:

```powershell
./scripts/build-easytier-ffi.ps1
```

The script looks for:

- `LIBCLANG_PATH`
- `PROTOC`

and falls back to the local machine paths used during this PoC.

After a successful build it also stages the Windows runtime files EasyTier expects:

- `Packet.dll`
- `wintun.dll`
- `WinDivert64.sys`

## Commands

Validate an frp client config:

```powershell
go run ./cmd/poc frp validate --config ../onani/frp/conf/frpc.toml
```

Run a same-process frp smoke test for a few seconds:

```powershell
go run ./cmd/poc frp smoke --config ../onani/frp/conf/frpc.toml --run-for 3s
```

Check the combined backend snapshot:

```powershell
go run ./cmd/poc snapshot
```

Run EasyTier through the DLL-backed adapter:

```powershell
go run ./cmd/poc easytier smoke --config ./testdata/easytier/basic.toml --run-for 3s
```

Run both `frp` and `EasyTier` inside the same host process:

```powershell
go run ./cmd/poc stack smoke --frp-config ../onani/frp/conf/frpc.toml --easytier-config ./testdata/easytier/basic.toml --run-for 3s
```

If the DLL cannot be found, the host falls back to a stub adapter and reports the reason in the snapshot output.
