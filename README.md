# NanoKVM-Extended

A fork of [sipeed/NanoKVM](https://github.com/sipeed/NanoKVM) that started with one problem —
keyboard hotkeys would not pass through to a KVM switch — and grew into a set of changes around
switching, monitoring, home automation and hardening.

Licensed under GPL-3.0, same as upstream. All upstream copyright and licence terms are
retained; see [LICENSE](LICENSE). Changes made here are documented in
[CHANGELOG-EXTENDED.md](CHANGELOG-EXTENDED.md), and upstream's own changelog remains in
[CHANGELOG.md](CHANGELOG.md).

> Built for one person's hardware and shared in case it is useful. It is not affiliated with
> Sipeed, and nothing here is tested across the whole NanoKVM range.

## The problem this started with

A NanoKVM feeding an AIMOS 8-in-1 HDMI KVM switch. The switch selects inputs by keyboard
hotkey, which worked from a physical keyboard but never from the NanoKVM.

Two separate causes:

1. **The emulated keyboard was not a USB HID boot device.** `S03usbdev` only set
   `bInterfaceSubClass = 1` when a `/boot/BIOS` marker file existed, which it does not by
   default. A real keyboard reports `1`; the NanoKVM reported `0`. BIOS/UEFI stacks and the
   hotkey scanners in inexpensive KVM switches implement only the HID *boot protocol*, so they
   never recognised it. Full operating systems were unaffected because they parse the report
   descriptor. See upstream [#544](https://github.com/sipeed/NanoKVM/issues/544) and
   [#25](https://github.com/sipeed/NanoKVM/issues/25).

2. **The switch's dedicated keyboard port rejects composite devices.** It accepts only a
   single-function HID device, so the composite gadget (keyboard + mouse + touchpad + network +
   mass storage) did not enumerate there at all. The switch's HUB ports do enumerate, but the
   hotkey scanner does not listen on those. **HID Only Mode** is what fixes this; USB 1.1 was
   not the deciding factor, dropping the composite functions was.

Both are addressed here, and the selected HID mode now survives updates — `new_app_init()`
copies `S03usbdev` over `/etc/init.d/` on *every* application update, which silently reverted
the mode until the choice moved to `/etc/kvm/hid-only`.

## What this fork adds

**KVM switching**

- Named targets with recorded hotkeys, shown as buttons in the menu and reorderable.
  Playback is stepped rather than a chord: switch hotkeys are usually sequential taps such as
  `ScrollLock, ScrollLock, 2`, and holding the keys together does not trigger them. The delay
  between steps is configurable, because switch firmware often drops taps that arrive faster
  than it polls.
- Hotkeys can also be replayed server-side, so a target can be selected with no browser open.
- MQTT publishing, for driving an ESPHome IR blaster as an alternative switching path.

**Home Assistant**

- MQTT discovery: the device announces itself and Home Assistant creates the entities. Every
  KVM target becomes a button, alongside HDMI state and resource sensors.
- Token-gated read-only video access, intended for go2rtc to ingest and re-serve as WebRTC.
  Transcoding happens on the Home Assistant host; this device serves one stream it already
  produces. The token is the on/off control — the endpoints 404 without one.

**Monitoring**

- CPU, memory, SD card, temperature, load average and uptime, read from `/proc` and `/sys`.
  Memory reports `MemAvailable`, not `MemFree`: the kernel keeps most of the remainder as
  reclaimable cache, so `MemFree` looks alarming while nothing is wrong.
- A process list with stop and force-kill. `init`, `kvm_system` and `NanoKVM-Server` are not
  killable: the first panics the kernel, the others stop video capture or take the web UI down.

**Display and interface**

- Choose which rows the OLED shows, plus an optional label for telling identical devices apart.
- Rearrange the menu bar icons.
- Wake-on-LAN devices can be added without waking them first.
- Tailscale version display and one-click update.

**Hardening** — optional TOTP (never mandatory), security headers, audit logging for
authentication and configuration changes, and a fix for the account file, which held the
password hash while being world-readable.

## Installing

Once this firmware is running, it checks **this repository** for updates rather than Sipeed's
CDN. That is deliberate: upstream's build would otherwise be offered as an "update" and would
quietly replace everything here. No configuration is needed — but note that GitHub's
`releases/latest` skips prereleases, so only releases published or promoted as stable are
offered.

Each release publishes both an over-the-air package and a full SD card image:

| Asset | Use |
|---|---|
| `nanokvm_<version>.tar.gz` | Settings > Update > offline upload |
| `nanokvm_<version>.img.xz` | Flash to an SD card with Etcher or `dd` |

The OTA package is the safer path. Flash the image when starting from scratch, or to recover.
Note that reflashing replaces the root filesystem, which resets the web password to
`admin`/`admin`.

Anything touching USB descriptors or the OLED daemon is worth trying on a spare SD card first:
a bad build there can leave you without input or without a display while everything else looks
healthy.

## Building

CI does the work; the toolchain is awkward to reproduce by hand.

- `builder-image.yml` publishes the RISC-V builder image to GHCR.
- `package.yml` cross-compiles the Go server and the C++ vision and system components, builds
  the frontend, and assembles the OTA package.
- `sd-image.yml` grafts that package onto Sipeed's published base image and emits `.img.xz`.
  Note Sipeed uses two release tracks: numeric tags are OTA packages, `v`-prefixed tags are
  full images.
- `go-mod-tidy.yml` exists because `go mod tidy` needs a real toolchain; hand-written
  `go.mod`/`go.sum` entries pick versions the module graph rejects.

Releases are cut with `create-tag.yml` followed by `release.yml`.

## Notes for anyone digging in

- **WebRTC fails in Firefox**, not on the device: Firefox replaces its host ICE candidate with
  an mDNS `.local` name the NanoKVM cannot resolve. Chrome works. Setting
  `media.peerconnection.ice.obfuscate_host_addresses=false` is the workaround. ICE also fails
  over plain HTTP in any browser, since host candidates are suppressed on an insecure origin.
- **`server/dl_lib/libkvm.so` is a stripped link stub.** Newly exported C functions fail to
  link without `-Wl,--allow-shlib-undefined`, because the real library defers its OpenCV and
  mmf symbols to sibling libraries resolved only at runtime.
- The OLED is drawn by `kvm_system`, a standalone daemon, so display work needs no new exported
  symbols.
- **H.265 is parked on `feat/h265`**: complete and building, never run on hardware. The encoder
  did *not* already support it — only H.264 was implemented, despite a comment suggesting
  otherwise — and `h264_stream_dump` hard-coded three stream packets while H.265 sends four,
  because it prepends a VPS. Browser support is the real limit: only Chrome 136+ and Safari 18+
  negotiate HEVC over WebRTC.

## Credit

All the hard parts — the RISC-V platform, the video pipeline, the web application — are
[Sipeed's](https://github.com/sipeed/NanoKVM). This fork is a thin layer on top.
