import type { ReactNode } from 'react';
import { Button, Switch } from 'antd';
import { useAtom } from 'jotai';
import {
  ChevronDownIcon,
  ChevronUpIcon,
  DiscIcon,
  DownloadIcon,
  FileJsonIcon,
  MaximizeIcon,
  MonitorIcon,
  NetworkIcon,
  PowerIcon,
  TerminalSquareIcon,
  XIcon,
  ZapIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as ls from '@/lib/localstorage.ts';
import { menuDisabledItemsAtom, menuOrderAtom } from '@/jotai/settings.ts';
import { Robot } from '@/components/icons/robot.tsx';

export const MenuIcons = () => {
  const { t } = useTranslation();

  const [menuDisabledItems, setMenuDisabledItems] = useAtom(menuDisabledItemsAtom);
  const [menuOrder, setMenuOrder] = useAtom(menuOrderAtom);

  const icons: Record<string, ReactNode> = {
    image: <DiscIcon size={16} />,
    download: <DownloadIcon size={16} />,
    terminal: <TerminalSquareIcon size={16} />,
    script: <FileJsonIcon size={16} />,
    wol: <NetworkIcon size={16} />,
    kvmSwitch: <MonitorIcon size={16} />,
    actions: <ZapIcon size={16} />,
    picoclaw: <Robot size={16} />,
    power: <PowerIcon size={16} />
  };

  // Reorderable icons come first, in their configured order, followed by the
  // two that always sit at the end of the bar.
  const items = [
    ...menuOrder
      .filter((key) => icons[key])
      .map((key) => ({ key, icon: icons[key], label: undefined as string | undefined })),
    { key: 'fullscreen', icon: <MaximizeIcon size={16} />, label: 'fullscreen.toggle' },
    { key: 'collapse', icon: <XIcon size={16} />, label: 'menu.collapse' }
  ];

  function move(key: string, delta: number) {
    const index = menuOrder.indexOf(key);
    const next = index + delta;
    if (index < 0 || next < 0 || next >= menuOrder.length) return;

    const reordered = [...menuOrder];
    [reordered[index], reordered[next]] = [reordered[next], reordered[index]];

    setMenuOrder(reordered);
    ls.setMenuOrder(reordered);
  }

  function updateItems(key: string) {
    const exist = menuDisabledItems.includes(key);

    const newItems = exist
      ? menuDisabledItems.filter((item) => item !== key)
      : [...menuDisabledItems, key];

    setMenuDisabledItems(newItems);
    ls.setMenuDisabledItems(newItems);
  }

  return (
    <div className="mt-8 flex flex-col space-y-5">
      <div className="flex flex-col">
        <span className="text-neutral-400">{t('settings.appearance.menuBar.icons')}</span>
        <span className="text-xs text-neutral-500">
          {t('settings.appearance.menuBar.iconsDesc')}
        </span>
      </div>

      <div className="mt-5 flex flex-col space-y-5">
        {items.map((item) => (
          <div key={item.key} className="flex items-center justify-between">
            <div className="flex items-center space-x-2 text-neutral-400">
              {item.icon}
              <span className="text-neutral-300">
                {item.label ? t(item.label) : t(`${item.key}.title`)}
              </span>
            </div>

            <div className="flex items-center space-x-1">
              {icons[item.key] && (
                <>
                  <Button
                    size="small"
                    icon={<ChevronUpIcon size={12} />}
                    disabled={menuOrder.indexOf(item.key) === 0}
                    title={t('settings.appearance.menuBar.moveUp')}
                    onClick={() => move(item.key, -1)}
                  />
                  <Button
                    size="small"
                    icon={<ChevronDownIcon size={12} />}
                    disabled={menuOrder.indexOf(item.key) === menuOrder.length - 1}
                    title={t('settings.appearance.menuBar.moveDown')}
                    onClick={() => move(item.key, 1)}
                  />
                </>
              )}

              <Switch
                value={!menuDisabledItems.includes(item.key)}
                onChange={() => updateItems(item.key)}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
