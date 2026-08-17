import { useRef } from 'react';
import type { ReactNode } from 'react';
import { Divider } from 'antd';
import clsx from 'clsx';
import { useAtomValue } from 'jotai';
import { GripVerticalIcon } from 'lucide-react';
import Draggable, { DraggableData, DraggableEvent } from 'react-draggable';

import {
  keyboardLedStatusVisibleAtom,
  menuDisabledItemsAtom,
  menuOrderAtom
} from '@/jotai/settings.ts';
import { useMenuBounds } from '@/hooks/useMenuBounds.ts';
import { useMenuVisibility } from '@/hooks/useMenuVisibility.ts';

import { KeyboardLedStatus } from '../keyboard-led-status';
import { Actions } from './actions';
import { DownloadImage } from './download.tsx';
import { Fullscreen } from './fullscreen';
import { Image } from './image';
import { Keyboard } from './keyboard';
import { KvmSwitch } from './kvm-switch';
import { Mouse } from './mouse';
import { Collapse, Expand } from './operations';
import { Picoclaw } from './picoclaw';
import { Power } from './power';
import { Screen } from './screen';
import { Script } from './script';
import { Settings } from './settings';
import { Terminal } from './terminal';
import { Wol } from './wol';

export const Menu = () => {
  const nodeRef = useRef<HTMLDivElement | null>(null);

  const menuDisabledItems = useAtomValue(menuDisabledItemsAtom);
  const menuOrder = useAtomValue(menuOrderAtom);
  const isKeyboardLedStatusVisible = useAtomValue(keyboardLedStatusVisibleAtom);

  const {
    isInitialized,
    isMenuExpanded,
    isMenuHidden,
    handleHovered,
    handleMoved,
    setIsMenuExpanded
  } = useMenuVisibility();

  const menuBounds = useMenuBounds(nodeRef, isMenuExpanded);

  function onDragStop(_e: DraggableEvent, data: DraggableData) {
    if (data.x === 0 && data.y === 0) return;
    handleMoved();
  }

  function isEnabled(item: string) {
    return !menuDisabledItems.includes(item);
  }

  // Rendered from the configured order rather than fixed JSX, so the bar can be
  // rearranged. Screen, keyboard, mouse and settings stay put: they are the
  // controls the device exists for.
  const components: Record<string, ReactNode> = {
    image: <Image />,
    download: <DownloadImage />,
    terminal: <Terminal />,
    script: <Script />,
    wol: <Wol />,
    kvmSwitch: <KvmSwitch />,
    actions: <Actions />,
    picoclaw: <Picoclaw />,
    power: <Power />
  };

  const orderedItems = menuOrder.filter((key) => components[key] && isEnabled(key));

  return (
    <Draggable
      nodeRef={nodeRef}
      bounds={menuBounds}
      handle="strong"
      positionOffset={{ x: '-50%', y: '0%' }}
      onStop={onDragStop}
    >
      <div
        ref={nodeRef}
        className={clsx(
          'fixed left-1/2 top-[10px] z-[1000] -translate-x-1/2 transition-opacity duration-300',
          isInitialized ? 'opacity-100' : 'opacity-0'
        )}
        onMouseEnter={() => handleHovered(true)}
        onMouseLeave={() => handleHovered(false)}
        onBlur={() => handleHovered(false)}
      >
        {/* Trigger area for auto-show when hidden */}
        {isMenuExpanded && (
          <div className="absolute -top-[10px] left-0 right-0 h-[46px] w-full bg-transparent" />
        )}

        {/* Menubar */}
        <div className="sticky top-[10px] flex w-full justify-center">
          <div
            className={clsx(
              'relative h-[36px] items-center rounded bg-neutral-800/80 pl-1 pr-2 transition-all duration-300',
              isMenuExpanded ? 'flex' : 'hidden',
              isMenuHidden ? '-translate-y-[110%] opacity-80' : 'translate-y-0 opacity-100'
            )}
          >
            {isMenuExpanded && isKeyboardLedStatusVisible && (
              <div
                className={clsx(
                  'absolute inset-y-0 right-full mr-1 transition-all duration-300',
                  isMenuHidden ? 'pointer-events-none opacity-0' : 'opacity-100'
                )}
              >
                <KeyboardLedStatus />
              </div>
            )}
            <strong>
              <div className="flex h-[30px] cursor-move select-none items-center justify-center pl-1 text-neutral-500">
                <GripVerticalIcon size={18} />
              </div>
            </strong>
            <Divider type="vertical" />

            <Screen />
            <Keyboard />
            <Mouse />
            <Divider type="vertical" />

            {orderedItems.map((key) => (
              <span key={key} className="contents">
                {components[key]}
              </span>
            ))}

            {orderedItems.length > 0 && <Divider type="vertical" />}

            <Settings />
            {isEnabled('fullscreen') && <Fullscreen />}
            {isEnabled('collapse') && <Collapse toggleMenu={setIsMenuExpanded} />}
          </div>
        </div>

        {/* Menubar expand button */}
        {!isMenuExpanded && <Expand toggleMenu={setIsMenuExpanded} />}
      </div>
    </Draggable>
  );
};
