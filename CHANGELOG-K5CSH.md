# Fork changelog

Changes in this fork relative to upstream [sipeed/NanoKVM](https://github.com/sipeed/NanoKVM).
Upstream's own changelog remains in [CHANGELOG.md](CHANGELOG.md).

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
