import { useEffect, useState } from 'react';
import { Button, Divider, Input, InputNumber, message, Switch } from 'antd';
import { PlusIcon, Trash2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/mqtt.ts';
import type { MqttCommand } from '@/api/mqtt.ts';

const emptyConfig = {
  enabled: false,
  broker: '',
  port: 1883,
  tls: false,
  username: '',
  topic: '',
  commands: [] as MqttCommand[],
  hasPassword: false
};

export const Mqtt = () => {
  const { t } = useTranslation();

  const [config, setConfig] = useState(emptyConfig);
  const [password, setPassword] = useState('');
  const [isPasswordDirty, setIsPasswordDirty] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    api.getMqttConfig().then((rsp: any) => {
      if (rsp.code !== 0) return;
      setConfig({ ...emptyConfig, ...rsp.data });
    }).catch((error) => {
      console.error('Failed to load MQTT config:', error);
      message.error('Failed to load MQTT configuration');
    });
  }, []);

  function update(patch: Partial<typeof emptyConfig>) {
    setConfig((prev) => ({ ...prev, ...patch }));
  }

  function updateCommand(index: number, patch: Partial<MqttCommand>) {
    setConfig((prev) => ({
      ...prev,
      commands: prev.commands.map((cmd, i) => (i === index ? { ...cmd, ...patch } : cmd))
    }));
  }

  function addCommand() {
    update({ commands: [...config.commands, { name: '', topic: '', payload: '' }] });
  }

  function removeCommand(index: number) {
    update({ commands: config.commands.filter((_, i) => i !== index) });
  }

  function save() {
    setIsLoading(true);

    api
      .setMqttConfig({
        enabled: config.enabled,
        broker: config.broker,
        port: config.port,
        tls: config.tls,
        username: config.username,
        topic: config.topic,
        commands: config.commands,
        // Omitted unless edited, so the stored password survives a save.
        ...(isPasswordDirty ? { password } : {})
      })
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.mqtt.saveFailed'));
          return;
        }

        message.success(t('settings.mqtt.saved'));
        setIsPasswordDirty(false);
        setPassword('');
        update({ hasPassword: isPasswordDirty ? password !== '' : config.hasPassword });
      })
      .finally(() => setIsLoading(false));
  }

  return (
    <>
      <div className="text-base">{t('settings.mqtt.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-5">
        <div className="flex items-center justify-between">
          <div className="flex flex-col">
            <span>{t('settings.mqtt.enable')}</span>
            <span className="text-xs text-neutral-500">{t('settings.mqtt.enableDesc')}</span>
          </div>
          <Switch checked={config.enabled} onChange={(v) => update({ enabled: v })} />
        </div>

        <div className="flex flex-col space-y-1">
          <span className="text-sm">{t('settings.mqtt.broker')}</span>
          <div className="flex space-x-2">
            <Input
              value={config.broker}
              placeholder="192.168.2.10"
              onChange={(e) => update({ broker: e.target.value })}
            />
            <InputNumber
              value={config.port}
              min={1}
              max={65535}
              onChange={(v) => update({ port: v || 1883 })}
            />
          </div>
        </div>

        <div className="flex items-center justify-between">
          <span>{t('settings.mqtt.tls')}</span>
          <Switch checked={config.tls} onChange={(v) => update({ tls: v })} />
        </div>

        <div className="flex flex-col space-y-1">
          <span className="text-sm">{t('settings.mqtt.username')}</span>
          <Input value={config.username} onChange={(e) => update({ username: e.target.value })} />
        </div>

        <div className="flex flex-col space-y-1">
          <span className="text-sm">{t('settings.mqtt.password')}</span>
          <Input.Password
            value={password}
            placeholder={
              config.hasPassword && !isPasswordDirty ? t('settings.mqtt.passwordStored') : undefined
            }
            onChange={(e) => {
              setPassword(e.target.value);
              setIsPasswordDirty(true);
            }}
          />
        </div>

        <div className="flex flex-col space-y-1">
          <span className="text-sm">{t('settings.mqtt.defaultTopic')}</span>
          <Input
            value={config.topic}
            placeholder="nanokvm/kvm-switch"
            onChange={(e) => update({ topic: e.target.value })}
          />
          <span className="text-xs text-neutral-500">{t('settings.mqtt.defaultTopicDesc')}</span>
        </div>
      </div>

      <Divider className="opacity-50" style={{ margin: '32px 0' }} />

      <div className="flex items-center justify-between">
        <div className="flex flex-col">
          <span>{t('settings.mqtt.commands')}</span>
          <span className="text-xs text-neutral-500">{t('settings.mqtt.commandsDesc')}</span>
        </div>
        <Button icon={<PlusIcon size={14} />} onClick={addCommand}>
          {t('settings.mqtt.addCommand')}
        </Button>
      </div>

      <div className="mt-4 flex flex-col space-y-3">
        {config.commands.map((cmd, index) => (
          <div key={index} className="flex items-center space-x-2">
            <Input
              value={cmd.name}
              placeholder={t('settings.mqtt.commandName')}
              onChange={(e) => updateCommand(index, { name: e.target.value })}
            />
            <Input
              value={cmd.topic}
              placeholder={t('settings.mqtt.commandTopic')}
              onChange={(e) => updateCommand(index, { topic: e.target.value })}
            />
            <Input
              value={cmd.payload}
              placeholder={t('settings.mqtt.commandPayload')}
              onChange={(e) => updateCommand(index, { payload: e.target.value })}
            />
            <Button danger icon={<Trash2Icon size={14} />} onClick={() => removeCommand(index)} />
          </div>
        ))}
      </div>

      <div className="mt-6">
        <Button type="primary" loading={isLoading} onClick={save}>
          {t('settings.mqtt.save')}
        </Button>
      </div>
    </>
  );
};
