import type { SwitcherStep } from '@/api/switcher.ts';

import { KeyboardReport } from './keyboard.ts';
import { client, MessageEvent } from './websocket.ts';

function send(report: Uint8Array) {
  client.send(new Uint8Array([MessageEvent.Keyboard, ...report]));
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Play a switch hotkey.
 *
 * Each step is pressed and then fully released before the next begins. KVM
 * switch hotkeys are usually a sequence of taps rather than one chord
 * (ScrollLock, ScrollLock, 2), and holding the keys down together does not
 * trigger them. The delay between steps matters too: switch firmware commonly
 * ignores taps that arrive faster than it polls.
 */
export async function playSwitcherSteps(steps: SwitcherStep[], stepDelayMs: number) {
  for (let i = 0; i < steps.length; i++) {
    const keyboard = new KeyboardReport();

    for (const key of steps[i].keys) {
      send(keyboard.keyDown(key.code));
    }

    // Release everything before the next step so a repeated key registers as a
    // second press rather than being swallowed as a held key.
    send(keyboard.reset());

    if (i < steps.length - 1 && stepDelayMs > 0) {
      await sleep(stepDelayMs);
    }
  }
}
