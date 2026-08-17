# Fork changelog

Changes in this fork relative to upstream [sipeed/NanoKVM](https://github.com/sipeed/NanoKVM).
Upstream's own changelog remains in [CHANGELOG.md](CHANGELOG.md).

## Parked work

### H.265 (branch `feat/h265`, not merged)

A complete H.265 path exists on `feat/h265` and builds end to end, but is **not on `main`** and
has never run on hardware. Parked deliberately; revisit if HEVC becomes worthwhile.

What it contains:

- An H.265 encoder channel (`PT_H265`, `VENC_RC_MODE_H265CBR`) in `kvm_mmf.cpp`, which had only
  ever implemented H.264 despite a comment implying otherwise.
- A fix to `h264_stream_dump`, which hard-coded three stream packets (SPS/PPS/I). H.265 sends
  four, because it prepends a VPS, so every H.265 keyframe would have been discarded as a VENC
  error. It now copies however many packets arrive, which is correct for both codecs.
- `kvmv_set_codec`/`kvmv_get_codec`, cgo bindings, a persisted setting, and a WebRTC track that
  selects its mime type and RTP payloader from the codec actually in use.
- Two build fixes that are prerequisites for **any** addition to the C API, not just H.265. The
  Go server links against a stripped `libkvm.so` stub checked into git, so newly exported
  functions failed to link; the real library defers its OpenCV and mmf symbols to sibling
  libraries resolved at runtime, which needs `-Wl,--allow-shlib-undefined`.

Before reviving it: H.265 over WebRTC works only in Chrome 136+ and Safari 18+. Firefox does
not support it and has stated it will not.

## 2.20.0

### Fixed

- **Untranslated keys were showing on screen** — the camera panel displayed `settings.mqtt.camera`
  and friends instead of its labels, and the Wake-on-LAN name field showed a raw key. Both were
  edits that silently failed to match the locale file and were invisible until someone opened
  that particular panel.

  A build now fails if any `t('...')` key used in the source is missing from the locale, so this
  cannot ship again. The check found one more missing key beyond the reported ones.

### Changed

- **The MQTT settings tab is now "Smart Home"**, which is what it is actually for. It covers
  Home Assistant discovery and the camera stream, only one of which involves MQTT.

- **The camera section says what it really does.** It was filed under MQTT, which implied the
  video goes through the broker. It does not: the stream is served directly over HTTPS and
  authenticated by the token in its URL, nothing is published to the broker, and no frames pass
  through it. go2rtc reads that MJPEG stream and re-serves it as WebRTC, so the transcoding
  happens on the Home Assistant machine rather than here.

## 2.19.0

### Added

- **Actions are now step sequences**, so one button or menu entry can do something useful
  rather than a single operation: pulse a relay, wait, then run a command, or drive several
  pins in order. Steps run in order and stop at the first failure, and the reason names the
  step that failed instead of reporting a bare "action failed".

  New step types alongside GPIO and command:

  - **Toggle**, which reads a pin and writes the opposite, which is what a light switch wants.
  - **PWM**, for brightness or fan speed, using the four chips the SoC exposes. Duty is lowered
    before the period changes and raised afterwards, because sysfs rejects a duty larger than
    the period it is being written against.
  - **Wait**, for hardware that needs a gap between operations.
  - Commands can now run in the background or wait with a configurable timeout.

- **The button understands four events**, not two: short, double, long and very long. The
  thresholds match the C++ daemon's own (1500 ms and 9000 ms), so a press means the same thing
  whichever handler reacts. A short press is held back briefly to see whether a second follows,
  which is the only way to tell a single press from the first half of a double.

  A very long press still resets the password: the daemon's handling is never suppressed, and a
  custom action runs alongside it.

- **GPIO state can be read**, so a toggle can show whether it is currently on. A pin that has
  never been driven reports as unknown rather than claiming to be off, since reading it would
  otherwise export it as a side effect of merely looking.

Actions saved in the previous single-operation form are converted when read, so nothing already
configured is lost.

## 2.18.0

### Fixed

- **PicoClaw could not be installed**, stalling at a few percent and then dropping back to the
  Install button with no explanation.

  The cause is not on the device. The stock URLs point at `cdn.sipeed.com`, which resolves to a
  CDN whose addresses time out at the TCP level — reproduced from the device and from a machine
  on an unrelated network, while GitHub over HTTPS from that same device works. Nothing in the
  firmware can make an unreachable host reachable.

  What is fixable: the download and checksum URLs can now be pointed at a mirror, under the
  PicoClaw settings. Both must be set together, since a mirror's archive checked against the
  stock checksum fails verification in a way that looks like corruption.

## 2.17.0

### Added

- **Custom GPIO and command actions**, configured in Settings > Actions and triggered from a new
  menu bar item or from the NanoKVM's own button.

  The device exposes five gpiochips covering 352-511, of which the firmware uses only a handful;
  the rest were unreachable without a shell. An action drives a pin — high, low, or a pulse of a
  given length, with an active-low option for the relay boards that need it — or runs a command.

  The pin is left exported after use, because unexporting resets the line and would drop a relay
  that had just been switched on. Pins the firmware already drives are allowed but labelled:
  repurposing the ATX power and reset lines is a reasonable thing to want, and refusing it
  outright would be worse than saying what it does.

  **The NanoKVM has one button, not two.** It is read as `gpio-keys` and told apart by how long
  it is held, so a short and a long press can run different actions. evdev delivers events to
  every reader, so watching it here does not take it away from the C++ daemon: OLED paging,
  Wi-Fi setup and hold-to-reset-password keep working, and a custom action runs alongside them.
  Suppressing that behaviour would mean changing the daemon, and losing the password reset is
  not a good trade for a remapped button.

  Commands run under a timeout, so one that never returns cannot pin the single core.

### Not yet included

- **Addressable RGB (WS2812) is not in this release.** It needs precise bit timing that
  userspace GPIO cannot hold, but the device does expose `/dev/spidev0.0` through `3.0`, and
  clocking the LED protocol over SPI is the reliable way to do it. That, along with LED count
  and animations, is worth its own release rather than being rushed in here.

## 2.16.0

### Added

- **Active sessions are visible and can be revoked**, in Settings > Account. The token is a JWT
  and says nothing about whether it is still wanted, so a stolen one stayed valid until it
  expired and there was no way to see it had been taken.

  Each token now carries a session id and a token whose session is gone is refused. The list
  shows every signed-in browser with its address and last-seen time, and either one or all of
  the others can be revoked. Revoking the current session is refused, since that is what logout
  is for.

  **Everyone signs in once more after this update.** A token issued earlier predates session
  tracking, and accepting those would leave a way to sidestep revocation entirely.

  The check runs on every authenticated request but only takes a read lock on an in-memory map,
  and `lastSeen` is written at most every five minutes so a busy session does not mean constant
  writes to an SD card. A store that cannot be read starts empty, which signs everyone out
  rather than failing open.

## 2.15.0

### Added

- **Configurable NTP and STUN servers**, in Settings > Network. Both reach out to third
  parties, and neither could be changed without editing files on the device.

  ntpd ships in the base image pointed at `pool.ntp.org`, so a device talks to a public pool
  whether or not anyone wanted it to. Servers are now set from the UI, and time sync can be
  switched off entirely. Only the `server` lines of `ntp.conf` are rewritten; the `restrict`
  lines the image ships are access control and are left alone. The setting is reasserted at
  startup, since `ntp.conf` lives on the root filesystem and an image update can replace it.

  STUN already defaulted to disabled but could only be changed in YAML. The stored value now
  wins over the default, and an empty field means disabled rather than being a third state.

### Fixed

- `resolv.conf` is deduplicated during repair. udhcpc appends rather than replaces, so a device
  collects a second copy of every line on each lease renewal, and musl reads only the first
  three nameservers it finds.

## 2.14.0

### Fixed

- **DNS was completely broken on any device using DHCP.** BusyBox's udhcpc script writes
  `resolv.conf` as `nameserver 1.1.1.1 # eth0`, and musl's resolver — unlike glibc — treats the
  trailing comment as part of the address, so every lookup failed with "bad address". This is
  what made PicoClaw's download stall at a few percent and then give up.

  The hook that writes clean entries already existed but was only installed when someone set
  DNS manually, so a device left on DHCP was broken by the stock script and never received the
  fix. It is now installed unconditionally at startup, and `resolv.conf` is repaired on every
  start, so an already-broken device fixes itself rather than needing the file edited by hand.

  udhcpc also overwrites a manual configuration on lease renewal, so a manual setting is now
  reasserted rather than merely cleaned up.

### Deliberately not implemented

- **Signed updates.** Signing would mean only builds made with the maintainer's key install,
  which works against a fork intended to be extended and rebuilt by others; it would make the
  project harder to contribute to for a threat that is already largely covered. Updates are
  fetched over HTTPS and verified against the SHA-512 recorded in the release manifest. This is
  a deliberate trade in favour of the project staying open to the community, and is not an
  oversight.

## 2.13.0

### Changed

- **Nothing contacts a third party unless asked.** An audit of every outbound endpoint found
  three that reached out on their own:

  - **STUN was set to `stun.l.google.com` and contacted on every WebRTC session.** On a LAN, or
    over a VPN, the host candidates are sufficient, so it now defaults to disabled. Set a
    server in the config if this device has to traverse NAT.
  - **The legacy `update-nanokvm.py` still fetched Sipeed's CDN.** `new_app_init()` copies that
    script to `/etc/kvm/`, so it stays on the device and would have pulled a stock build over
    this firmware.
  - **The update settings page carried its own idea of the official server**, which no longer
    matched the firmware's, so the built-in default was displayed as a custom server.

  Tailscale and PicoClaw download URLs are unchanged: nothing is fetched unless those
  components are installed, and upstream is where they come from. The Sipeed community links on
  the About page are ordinary links and issue no request unless clicked.

  Still outstanding: `ntpd` runs from the base image rather than from anything in this
  repository, so it uses whatever pool that image ships.

## 2.12.0

### Changed

- **Updates now come from this fork rather than Sipeed's CDN.** With no custom update server
  configured, the device fell back to upstream, which would offer a stock build as an "update"
  and quietly replace everything this fork adds. Both the stable and preview URLs now resolve
  here; there is no separate preview channel.

  Note that GitHub's `releases/latest` skips prereleases, so a release must be published or
  promoted as stable before a device will see it.

- The repository was renamed to `NanoKVM-Extended`. GitHub redirects the old URLs, so an
  existing custom update server keeps working, but it is worth pointing at the new address.

## 2.11.1

### Added

- **A Wake-on-LAN device can be named while it is added.** Saving stored only the address, so
  naming meant saving first and then editing the entry, which is not obvious from a list of raw
  MAC addresses. An optional name field now sits beside the address.

## 2.11.0

### Fixed

- **Saving a Wake-on-LAN device always failed with "invalid arguments"**, whatever the MAC
  format. Adding reused the request type used for renaming, whose name field is marked
  required, while the Save button sends only the address. Adding now has its own request type.

### Added

- **Process list**, in Settings > Resources: what is running, sorted by memory, with stop and
  force-kill. It reads `/proc` directly rather than shelling out to busybox's cut-down `ps`,
  and takes a process name from between the first `(` and the last `)` in `stat`, because a
  name can itself contain spaces and parentheses.

  `init`, `kvm_system` and `NanoKVM-Server` are not killable from here: killing the first
  panics the kernel, and the other two would stop video capture or take the web UI down along
  with the button that was just pressed. Kills are audited.

## 2.10.0

### Added

- **Wake-on-LAN devices can be added without waking them.** An address previously only joined
  the list as a side effect of being woken, so a machine had to be woken by hand once before it
  could be clicked. The list is now something that can be set up in advance. Naming and
  click-to-wake already existed.

- **KVM switch targets can be reordered**, which sets the order the buttons appear in both the
  menu and Home Assistant, since both read the same list.

- **The menu bar icons can be rearranged**, in Settings > Appearance, alongside the existing
  show/hide toggles. Screen, keyboard, mouse and settings stay put: they are the controls the
  device exists for. An order stored by an earlier version is reconciled on load rather than
  trusted outright, so an icon added in a later release still appears for someone who had
  already rearranged their bar.

- **The Tailscale version is shown, with an update button.** Install always fetched the latest
  archive, so updating was already a reinstall; what was missing was any way to see which
  version is running or whether a newer one exists. The available version comes from the
  redirect the "latest" URL lands on, and an update is only offered when both versions are
  known and differ, so a failed lookup cannot nag about an update that may not exist.

  Updating stops the daemon before replacing the binaries, since it holds them open, and
  restarts it on the old ones if the download fails rather than leaving it stopped. Login state
  lives outside the binaries, so the node stays connected.

## 2.9.0

### Added

- **Choose what the OLED shows**, under Settings > Device. Each row (IP, resolution, stream
  type, frame rate, quality) can be hidden, and an optional label can be set.

  The label draws on the IP row when the IP is hidden, which is enough to tell otherwise
  identical devices apart in a rack.

  Settings live in `/etc/kvm/oled.conf`, outside `/kvmapp`, so they survive an application
  update — the same reason the HID mode marker moved there. The `kvm_system` daemon re-reads
  the file when its timestamp changes, so a save applies within seconds without restarting
  anything, and a missing file means show everything, which is the stock behaviour. The file
  is replaced atomically, since the daemon watches the timestamp and could otherwise read a
  half-written file.

  A hidden row is skipped before rendering and its label painted as spaces, so no text is
  left stranded on screen by a later state update.

## 2.8.0

### Added

- **Camera access for Home Assistant**, under Settings > MQTT. Home Assistant needs to pull
  video from something that cannot hold a browser session, so the existing MJPEG stream is
  exposed to a caller holding a token. go2rtc ingests `multipart/x-mixed-replace` directly and
  re-serves it as WebRTC, which means the transcoding happens on the Home Assistant host and
  this device serves one stream it already produces. Publishing frames over MQTT was rejected
  for the opposite reason: it would make a 256 MB device encode and ship every frame through
  the broker.

  The token is the on/off control. Both endpoints return 404 until one is generated, disabling
  clears it, and enabling mints a fresh one so toggling off and on invalidates any URL handed
  out before. The two routes sit outside the session-authenticated group and are read-only:
  neither can send input to the target. The token is compared in constant time and accepted
  from the query string, because go2rtc takes a plain URL.

  A snapshot endpoint is included for a still-image or polling setup.

## 2.7.1

> **2.7.0 was tagged from a commit that did not contain any of this.** The work below was
> committed to a side branch while `git push origin main` pushed the unchanged `main` ref, so
> the release built from `main` shipped 2.6.1's contents under a 2.7.0 tag. Use 2.7.1.

### Added

- **Resource usage tab**, under Settings > Resources: CPU, memory, SD card, temperature, load
  average and uptime, refreshed every few seconds. All of it is read from `/proc` and `/sys`,
  so polling costs almost nothing and never touches the video or HID paths.

  CPU percentage is a delta between two samples of cumulative jiffies, so a poll arriving
  sooner than the sample gap reuses the previous answer instead of dividing by a tiny delta
  and reporting noise. Memory reports `MemAvailable`, not `MemFree`: on this device `MemFree`
  sits at a few MB because the kernel holds most of the remainder as reclaimable cache, which
  reads as alarming while nothing is wrong.

  The same figures are published as Home Assistant sensors, so they can be graphed and
  alerted on there.

- **Home Assistant integration over MQTT discovery.** Enable it under Settings > MQTT. This
  device announces itself to Home Assistant, which creates the entities on its own; every
  configured KVM Switch target is mirrored as a Home Assistant button, so selecting a machine
  can be automated.

  This uses a persistent connection rather than the one-shot publishing behind user-defined
  commands, because discovery needs a live session: a retained last-will so entities go
  unavailable when the device drops off, a subscription so buttons can be pressed, and
  periodic state. Discovery documents are retained, so Home Assistant restores the entities
  across restarts without this device being online.

- **Server-side hotkey replay.** KVM targets can now be triggered with no browser present,
  via `POST /api/switcher/press` or the Home Assistant buttons. The browser still plays
  targets over the WebSocket to keep the UI responsive. Rather than duplicate the keymap in
  Go, the browser resolves each key's HID usage code when the hotkey is recorded and stores
  it alongside the target.

## 2.6.1

### Fixed

- **The Content-Security-Policy added in 2.6.0 blocked the app's own inline scripts**, breaking
  parts of the UI. `script-src` and `style-src` are removed rather than loosened to
  `'unsafe-inline'`: a policy permissive enough for inlined bundle output allows exactly what
  `script-src` exists to prevent, while still breaking on the next dependency that inlines
  something. Making it strict would require per-build hashes or a nonce threaded through the
  bundler.

  What remains is what actually protects this device and cannot be tripped by how the bundle
  is emitted: frame denial and `frame-ancestors` (the realistic threat for something that
  controls a machine's keyboard and mouse), `nosniff`, a referrer policy, `base-uri` and
  `form-action`.

## 2.6.0

### Added

- **Named KVM switch buttons.**
  Label each machine behind the switch and record the hotkey that selects it, under
  Settings > KVM Switch. Configured targets appear as buttons in the KVM Switch menu, so
  switching is a click instead of remembering port numbers. Only configured targets show.

  Playback is stepped rather than a chord: switch hotkeys are usually sequential taps such
  as ScrollLock, ScrollLock, 2, and the built-in shortcut feature holds every key down at
  once, which those switches do not recognise. Each step is released before the next
  begins, and the inter-step delay is configurable because switch firmware often drops taps
  that arrive faster than it polls. Keys held together are still captured as one step, so
  chords such as Ctrl+Alt+1 work.

- **Optional TOTP two-factor authentication.** Opt-in and never required. Enrolment only
  takes effect once a code confirms the authenticator is in sync, so a botched setup cannot
  lock anyone out. Ten single-use backup codes are issued at enrolment and only their
  hashes are stored. Disabling requires both the password and a code.

  Recovery, since this device exposes no shell by default: power down, read the SD card
  elsewhere and delete `/etc/kvm/totp.json`.

- **Security headers**: frame denial, nosniff, a referrer policy, and a CSP scoped to allow
  the app's own WebSocket and blob-based video. HSTS is deliberately omitted: the device
  serves a self-signed certificate on a private address, and a pinned HSTS entry would lock
  the UI out if it ever served plain HTTP again.

- **Audit logging** for authentication and access-affecting configuration changes, tagged
  with an `audit` field. Deliberately limited to cold endpoints; the HID path is never
  logged, since that would record whatever is typed on the target.

### Fixed

- The account file holds the bcrypt hash of the admin password but was written world-
  readable at `0o644`; it is now `0o600`, with an explicit chmod so files created by earlier
  builds are tightened too. Its parent directory was created `0o644`, which leaves a
  directory non-traversable, and is now `0o755`.

## 2.5.5

### Fixed

- **The selected HID mode now survives application updates.**
  `new_app_init()` in `kvm_system` runs an unconditional
  `cp -f /kvmapp/system/init.d/S03usbdev /etc/init.d/` on every application update, so a
  device switched to HID-only mode was reverted to the composite gadget by the next
  update. Anything depending on the HID-only layout broke silently, with no indication
  that an update had changed it.

  This matters for KVM switches whose dedicated keyboard port only accepts a
  single-function HID device: the composite gadget (keyboard + mouse + touchpad +
  network + mass storage) is rejected outright by that port, so keyboard hotkey
  switching stops working.

  `S03usbdev` now selects the layout at boot from `/etc/kvm/hid-only`, which lives
  outside `/kvmapp` and therefore survives updates, rather than depending on which copy
  of the script happens to be installed. `SetHidMode` maintains that marker.

## 2.5.4

### Fixed

- **The MQTT settings tab crashed with "can't access property map, commands is null".**
  Go marshals a nil slice as `null` rather than `[]`, and the client spread that null over
  its defaults, so the render mapped over null. `Commands` is now kept non-nil in the
  default config, after loading a file that stored null, and when a request omits it. The
  client also coerces it, so a device on an older build cannot reproduce the crash.

- **The virtual keyboard and keyboard shortcuts crashed with React error #130.**
  `react-simple-keyboard` 3.x is CJS-only and its module export is an object whose
  `.default` holds the component. Under the Vite 8 rolldown bundler adopted upstream in
  761b02e, the default import resolves to that namespace object instead of the component,
  which React rejects as an invalid element type. The interop default is now unwrapped.
  A scan of every bare default import found this to be the only affected package.

## 2.5.3

### Fixed

- **The SD image now actually installs the HID fix.**
  `/etc/init.d/S03usbdev` is a regular file baked into the base image at build time, not a
  symlink into `/kvmapp`, and it is a different (older) file than the copy shipped in the
  application package. Replacing `/kvmapp` therefore left the stock USB gadget script running
  at boot, so the 2.5.1 and 2.5.2 images did not change any USB descriptor.

  The image build now installs `kvmapp/system/init.d/S03usbdev` over `/etc/init.d/S03usbdev`
  — the same copy the HID-mode switch performs at runtime — and fails the build if the
  resulting script still gates the subclass on `/boot/BIOS`.

  **This does not apply to OTA updates**, which only replace `/kvmapp`. After updating over
  OTA, toggle Menu > Mouse > HID Only Mode off and on to copy the new script into
  `/etc/init.d/`, or flash the image instead.

## 2.5.2

### Added

- **MQTT command publishing, with a KVM Switch menu.**
  Provides a switching path that does not depend on HID at all: NanoKVM publishes a
  command to an MQTT broker, and an ESPHome IR blaster subscribed to that topic
  transmits the switch's IR code.

  Commands are user-defined name/topic/payload triples, so payloads can be matched to
  whatever the ESPHome automation expects without rebuilding the firmware. Each becomes
  a button in the new KVM Switch menu. Configure under Settings > MQTT.

  The client connects and disconnects per publish rather than holding an idle
  connection, keeping the feature entirely off the video and HID paths. Broker
  credentials are stored in `/etc/kvm/mqtt.json`, written atomically at mode 0600; the
  stored password is never returned to the browser. TLS connections require TLS 1.2 or
  newer.

## 2.5.1

### Fixed

- **USB HID keyboard and mouse are now always Boot Interface devices.**
  `kvmapp/system/init.d/S03usbdev` previously set `bInterfaceSubClass = 1` only when the
  marker file `/boot/BIOS` was present, which it is not by default. Stock devices therefore
  enumerated the emulated keyboard with `bInterfaceSubClass 0` while a real keyboard reports
  `1` (Boot Interface Subclass).

  BIOS/UEFI USB stacks and the hotkey scanners in inexpensive KVM switches implement only the
  minimal HID *boot protocol* rather than full report-descriptor parsing, so they did not
  recognise the emulated keyboard. A full OS was unaffected because it parses the report
  descriptor. This is the cause of KVM-switch hotkey combinations working from a physical
  keyboard but not through NanoKVM.

  The `/boot/BIOS` conditional is removed; keyboard (`hid.GS0`) and mouse (`hid.GS1`) are now
  unconditionally boot-compliant in both normal and HID-only mode. Unlike switching the device
  to HID-only mode, this keeps the RNDIS/NCM network and mass-storage gadget functions intact.

  Upstream references: [#544](https://github.com/sipeed/NanoKVM/issues/544),
  [#25](https://github.com/sipeed/NanoKVM/issues/25),
  [PR #814](https://github.com/sipeed/NanoKVM/pull/814) (unmerged; used as a design reference).

- **The touchpad HID function is no longer advertised as a boot device.**
  `hid.GS2` reports absolute coordinates, which a boot-protocol host misinterprets as relative
  deltas. `S03usbhid` set `subclass = 1` on it unconditionally and `S03usbdev` set it whenever
  `/boot/BIOS` existed. Both now leave it non-boot.
