import { useState } from 'react';
import { Button, Empty } from 'antd';
import { ZapIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { getActions, runAction } from '@/api/actions.ts';
import type { ActionInfo } from '@/api/actions.ts';
import { MenuItem } from '@/components/menu-item.tsx';

export const Actions = () => {
  const { t } = useTranslation();

  const [actions, setActions] = useState<ActionInfo[]>([]);
  const [busy, setBusy] = useState('');
  const [log, setLog] = useState('');

  function handleOpenChange(open: boolean) {
    if (!open) {
      setLog('');
      return;
    }

    getActions()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        // Only the ones asked for in the menu; an action can exist purely for
        // the physical button.
        setActions((rsp.data?.actions ?? []).filter((item: ActionInfo) => item.showInMenu));
      })
      .catch(() => setLog(t('actions.loadFailed')));
  }

  function run(action: ActionInfo) {
    if (busy) return;

    setBusy(action.id);
    setLog('');

    runAction(action.id)
      .then((rsp: any) => {
        setLog(rsp.code === 0 ? t('actions.ran', { name: action.name }) : rsp.msg);
      })
      .catch(() => setLog(t('actions.runFailed')))
      .finally(() => setBusy(''));
  }

  const content = (
    <div className="flex w-[240px] flex-col space-y-2 p-1">
      <div className="text-sm text-neutral-300">{t('actions.title')}</div>

      {actions.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={<span className="text-xs text-neutral-500">{t('actions.empty')}</span>}
        />
      ) : (
        <div className="flex flex-col space-y-1">
          {actions.map((action) => (
            <Button key={action.id} block loading={busy === action.id} onClick={() => run(action)}>
              {action.name}
            </Button>
          ))}
        </div>
      )}

      {log && <div className="text-xs text-neutral-400">{log}</div>}
    </div>
  );

  return (
    <MenuItem
      title={t('actions.title')}
      icon={<ZapIcon size={18} />}
      content={content}
      onOpenChange={handleOpenChange}
    />
  );
};
