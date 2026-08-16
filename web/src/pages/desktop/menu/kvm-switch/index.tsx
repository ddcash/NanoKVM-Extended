import { useState } from 'react';
import { Button, Divider, Empty } from 'antd';
import { MonitorIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { getMqttConfig, publishMqttCommand } from '@/api/mqtt.ts';
import type { MqttCommand } from '@/api/mqtt.ts';
import { getSwitcher } from '@/api/switcher.ts';
import type { SwitcherTarget } from '@/api/switcher.ts';
import { playSwitcherSteps } from '@/lib/switcher.ts';
import { MenuItem } from '@/components/menu-item.tsx';

export const KvmSwitch = () => {
  const { t } = useTranslation();

  const [targets, setTargets] = useState<SwitcherTarget[]>([]);
  const [stepDelayMs, setStepDelayMs] = useState(120);

  const [commands, setCommands] = useState<MqttCommand[]>([]);
  const [mqttEnabled, setMqttEnabled] = useState(false);

  const [busy, setBusy] = useState('');
  const [log, setLog] = useState('');

  function handleOpenChange(open: boolean) {
    if (!open) {
      setLog('');
      return;
    }

    getSwitcher()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setTargets(rsp.data?.targets ?? []);
        setStepDelayMs(rsp.data?.stepDelayMs ?? 120);
      })
      .catch(() => setLog(t('kvmSwitch.loadFailed')));

    // MQTT is optional; a failure here must not hide the key-based targets.
    getMqttConfig()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setMqttEnabled(!!rsp.data?.enabled);
        setCommands(rsp.data?.commands ?? []);
      })
      .catch(() => {
        setMqttEnabled(false);
        setCommands([]);
      });
  }

  async function switchTo(target: SwitcherTarget) {
    if (busy) return;

    setBusy(target.id);
    setLog('');

    try {
      await playSwitcherSteps(target.steps, stepDelayMs);
      setLog(t('kvmSwitch.switched', { name: target.name }));
    } catch (error) {
      console.error('Failed to send switch keys:', error);
      setLog(t('kvmSwitch.sendFailed'));
    } finally {
      setBusy('');
    }
  }

  function sendMqtt(name: string) {
    if (busy) return;

    setBusy(name);
    setLog('');

    publishMqttCommand(name)
      .then((rsp: any) => {
        setLog(rsp.code === 0 ? t('kvmSwitch.sent', { name }) : rsp.msg);
      })
      .catch(() => setLog(t('kvmSwitch.sendFailed')))
      .finally(() => setBusy(''));
  }

  const hasMqtt = mqttEnabled && commands.length > 0;

  const content = (
    <div className="flex w-[240px] flex-col space-y-2 p-1">
      <div className="text-sm text-neutral-300">{t('kvmSwitch.title')}</div>

      {targets.length === 0 && !hasMqtt ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={<span className="text-xs text-neutral-500">{t('kvmSwitch.noTargets')}</span>}
        />
      ) : (
        <div className="flex flex-col space-y-1">
          {targets.map((target) => (
            <Button
              key={target.id}
              block
              loading={busy === target.id}
              onClick={() => switchTo(target)}
            >
              {target.name}
            </Button>
          ))}

          {targets.length > 0 && hasMqtt && <Divider className="my-1 opacity-40" />}

          {hasMqtt &&
            commands.map((cmd) => (
              <Button
                key={cmd.name}
                block
                loading={busy === cmd.name}
                onClick={() => sendMqtt(cmd.name)}
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
