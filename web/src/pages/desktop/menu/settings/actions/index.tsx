import { useEffect, useState } from 'react';
import { Alert, Button, Divider, Input, InputNumber, message, Select, Switch, Tag } from 'antd';
import { PlusIcon, Trash2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/actions.ts';
import type { ActionInfo } from '@/api/actions.ts';

export const Actions = () => {
  const { t } = useTranslation();

  const [actions, setActions] = useState<ActionInfo[]>([]);
  const [shortPress, setShortPress] = useState('');
  const [longPress, setLongPress] = useState('');
  const [keepDefaults, setKeepDefaults] = useState(true);
  const [range, setRange] = useState({ min: 352, max: 511 });
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    api
      .getActions()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setActions(rsp.data?.actions ?? []);
        setShortPress(rsp.data?.shortPress ?? '');
        setLongPress(rsp.data?.longPress ?? '');
        setKeepDefaults(rsp.data?.keepDefaults ?? true);
        setRange({ min: rsp.data?.minGpio ?? 352, max: rsp.data?.maxGpio ?? 511 });
      })
      .catch(() => {
        // Fields keep their defaults if this fails.
      });
  }, []);

  function update(index: number, patch: Partial<ActionInfo>) {
    setActions((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)));
  }

  function updateGpio(index: number, patch: Partial<api.ActionGPIO>) {
    setActions((prev) =>
      prev.map((item, i) =>
        i === index
          ? {
              ...item,
              gpio: {
                pin: item.gpio?.pin ?? range.min,
                mode: item.gpio?.mode ?? 'pulse',
                durationMs: item.gpio?.durationMs ?? 500,
                activeLow: item.gpio?.activeLow ?? false,
                ...patch
              }
            }
          : item
      )
    );
  }

  function add() {
    setActions((prev) => [
      ...prev,
      {
        // The server replaces an empty id; this only has to be unique in React.
        id: `new-${Date.now()}-${prev.length}`,
        name: '',
        type: 'gpio',
        gpio: { pin: range.min, mode: 'pulse', durationMs: 500, activeLow: false },
        showInMenu: true
      }
    ]);
  }

  function remove(index: number) {
    const removed = actions[index];
    if (removed) {
      // A binding left pointing at a deleted action is rejected on save.
      if (shortPress === removed.id) setShortPress('');
      if (longPress === removed.id) setLongPress('');
    }
    setActions((prev) => prev.filter((_, i) => i !== index));
  }

  function save() {
    setIsLoading(true);

    api
      .setActions({ actions, shortPress, longPress, keepDefaults })
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.actions.saveFailed'));
          return;
        }
        setActions(rsp.data?.actions ?? actions);
        message.success(t('settings.actions.saved'));
      })
      .catch(() => message.error(t('settings.actions.saveFailed')))
      .finally(() => setIsLoading(false));
  }

  function test(action: ActionInfo) {
    api
      .runAction(action.id)
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.actions.runFailed'));
          return;
        }
        message.success(t('settings.actions.ran'));
      })
      .catch(() => message.error(t('settings.actions.runFailed')));
  }

  // A binding can only point at something already saved.
  const bindable = actions.filter((action) => !action.id.startsWith('new-'));
  const options = [
    { value: '', label: t('settings.actions.none') },
    ...bindable.map((action) => ({ value: action.id, label: action.name || action.id }))
  ];

  return (
    <>
      <div className="text-base">{t('settings.actions.title')}</div>
      <Divider className="opacity-50" />

      <span className="text-xs text-neutral-500">{t('settings.actions.desc')}</span>

      <div className="mt-4 flex items-center justify-between">
        <span>{t('settings.actions.list')}</span>
        <Button icon={<PlusIcon size={14} />} onClick={add}>
          {t('settings.actions.add')}
        </Button>
      </div>

      <div className="mt-3 flex flex-col space-y-3">
        {actions.map((action, index) => (
          <div key={action.id} className="flex flex-col space-y-2 rounded bg-neutral-800/40 p-3">
            <div className="flex items-center space-x-2">
              <Input
                value={action.name}
                placeholder={t('settings.actions.namePlaceholder')}
                onChange={(e) => update(index, { name: e.target.value })}
              />
              <Select
                value={action.type}
                style={{ width: 130 }}
                onChange={(value) => update(index, { type: value })}
                options={[
                  { value: 'gpio', label: t('settings.actions.typeGpio') },
                  { value: 'command', label: t('settings.actions.typeCommand') }
                ]}
              />
              <Button danger icon={<Trash2Icon size={14} />} onClick={() => remove(index)} />
            </div>

            {action.type === 'gpio' ? (
              <div className="flex flex-col space-y-2">
                <div className="flex items-center space-x-2">
                  <InputNumber
                    addonBefore="GPIO"
                    min={range.min}
                    max={range.max}
                    value={action.gpio?.pin}
                    onChange={(value) => updateGpio(index, { pin: value ?? range.min })}
                  />
                  <Select
                    value={action.gpio?.mode ?? 'pulse'}
                    style={{ width: 120 }}
                    onChange={(value) => updateGpio(index, { mode: value })}
                    options={[
                      { value: 'pulse', label: t('settings.actions.modePulse') },
                      { value: 'high', label: t('settings.actions.modeHigh') },
                      { value: 'low', label: t('settings.actions.modeLow') }
                    ]}
                  />
                  {action.gpio?.mode === 'pulse' && (
                    <InputNumber
                      addonAfter="ms"
                      min={0}
                      max={10000}
                      step={100}
                      value={action.gpio?.durationMs}
                      onChange={(value) => updateGpio(index, { durationMs: value ?? 500 })}
                    />
                  )}
                </div>

                <div className="flex items-center space-x-2">
                  <Switch
                    size="small"
                    checked={action.gpio?.activeLow ?? false}
                    onChange={(value) => updateGpio(index, { activeLow: value })}
                  />
                  <span className="text-xs text-neutral-500">
                    {t('settings.actions.activeLow')}
                  </span>
                </div>

                {action.gpio?.reservedFor && (
                  <Alert
                    type="warning"
                    showIcon
                    message={t('settings.actions.reserved', { use: action.gpio.reservedFor })}
                  />
                )}
              </div>
            ) : (
              <Input.TextArea
                rows={2}
                value={action.command}
                spellCheck={false}
                placeholder="/etc/init.d/S95nanokvm restart"
                onChange={(e) => update(index, { command: e.target.value })}
              />
            )}

            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <Switch
                  size="small"
                  checked={action.showInMenu}
                  onChange={(value) => update(index, { showInMenu: value })}
                />
                <span className="text-xs text-neutral-500">{t('settings.actions.showInMenu')}</span>
              </div>

              {!action.id.startsWith('new-') && (
                <Button size="small" onClick={() => test(action)}>
                  {t('settings.actions.test')}
                </Button>
              )}
            </div>
          </div>
        ))}
      </div>

      <Divider className="opacity-50" style={{ margin: '28px 0' }} />

      <div className="flex flex-col space-y-1">
        <span>{t('settings.actions.button')}</span>
        <span className="text-xs text-neutral-500">{t('settings.actions.buttonDesc')}</span>
      </div>

      <div className="mt-4 flex flex-col space-y-4">
        <div className="flex items-center justify-between">
          <span className="text-sm">{t('settings.actions.shortPress')}</span>
          <Select
            value={shortPress}
            style={{ width: 220 }}
            options={options}
            onChange={setShortPress}
          />
        </div>

        <div className="flex items-center justify-between">
          <span className="text-sm">{t('settings.actions.longPress')}</span>
          <Select
            value={longPress}
            style={{ width: 220 }}
            options={options}
            onChange={setLongPress}
          />
        </div>

        <div className="flex items-center justify-between">
          <div className="flex flex-col">
            <span className="text-sm">{t('settings.actions.keepDefaults')}</span>
            <span className="text-xs text-neutral-500">
              {t('settings.actions.keepDefaultsDesc')}
            </span>
          </div>
          <Tag color="blue">{t('settings.actions.always')}</Tag>
        </div>
      </div>

      <div className="mt-6">
        <Button type="primary" loading={isLoading} onClick={save}>
          {t('settings.actions.save')}
        </Button>
      </div>
    </>
  );
};
