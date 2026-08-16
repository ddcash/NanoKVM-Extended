import { useState } from 'react';
import { Button, Empty } from 'antd';
import { MonitorIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { getMqttConfig, publishMqttCommand } from '@/api/mqtt.ts';
import type { MqttCommand } from '@/api/mqtt.ts';
import { MenuItem } from '@/components/menu-item.tsx';

export const KvmSwitch = () => {
  const { t } = useTranslation();

  const [commands, setCommands] = useState<MqttCommand[]>([]);
  const [enabled, setEnabled] = useState(false);
  const [sending, setSending] = useState('');
  const [log, setLog] = useState('');

  function handleOpenChange(open: boolean) {
    if (!open) {
      setLog('');
      return;
    }

    getMqttConfig()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setEnabled(!!rsp.data?.enabled);
        setCommands(rsp.data?.commands || []);
      })
      .catch((error) => {
        console.error('Failed to load MQTT config:', error);
        setLog('Failed to load configuration');
      });
  }

  function send(name: string) {
    if (sending) return;

    setSending(name);
    setLog('');

    publishMqttCommand(name)
      .then((rsp: any) => {
        setLog(rsp.code === 0 ? t('kvmSwitch.sent', { name }) : rsp.msg);
      })
      .catch((error) => {
        console.error('Failed to publish command:', error);
        setLog('Failed to send command');
      })
      .finally(() => setSending(''));
  }

  const content = (
    <div className="flex w-[240px] flex-col space-y-2 p-1">
      <div className="text-sm text-neutral-300">{t('kvmSwitch.title')}</div>

      {!enabled || commands.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <span className="text-xs text-neutral-500">
              {enabled ? t('kvmSwitch.noCommands') : t('kvmSwitch.disabled')}
            </span>
          }
        />
      ) : (
        <div className="flex flex-col space-y-1">
          {commands.map((cmd) => (
            <Button
              key={cmd.name}
              block
              loading={sending === cmd.name}
              onClick={() => send(cmd.name)}
            >
              {cmd.name}
            </Button>
          ))}
        </div>
      )}

      {log && <div className="text-xs text-neutral-400">{log}</div>}
    </div>
  );

  return (
    <MenuItem
      title={t('kvmSwitch.title')}
      icon={<MonitorIcon size={18} />}
      content={content}
      onOpenChange={handleOpenChange}
    />
  );
};
