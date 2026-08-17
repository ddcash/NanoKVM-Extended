import { atom } from 'jotai';

// menu bar disabled items
export const menuDisabledItemsAtom = atom<string[]>([]);

// The reorderable menu icons, in their default order. Screen, keyboard, mouse
// and settings are fixed and deliberately absent.
export const DEFAULT_MENU_ORDER = [
  'image',
  'download',
  'terminal',
  'script',
  'wol',
  'kvmSwitch',
  'actions',
  'picoclaw',
  'power'
];

// menu bar icon order
export const menuOrderAtom = atom<string[]>(DEFAULT_MENU_ORDER);

// track how many submenus are currently open
export const submenuOpenCountAtom = atom(0);

// web title
export const webTitleAtom = atom('');

// menu display mode: 'off' | 'auto' | 'always'
export const menuDisplayModeAtom = atom<string>('auto');

// show the remote keyboard lock-status indicator beside the menu bar
export const keyboardLedStatusVisibleAtom = atom(true);
