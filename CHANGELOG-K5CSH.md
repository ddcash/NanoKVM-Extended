# Fork changelog

Changes in this fork relative to upstream [sipeed/NanoKVM](https://github.com/sipeed/NanoKVM).
Upstream's own changelog remains in [CHANGELOG.md](CHANGELOG.md).

## Unreleased

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
