import { useEffect, useState } from 'react';
import { Button, Divider, Input, message, Select, Switch } from 'antd';
import { PlusIcon, Trash2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/actions.ts';
import type { ActionInfo, ActionStep } from '@/api/actions.ts';

import { StepEditor } from './step.tsx';

type Bindings = {
  shortPress: string;
  doublePress: string;
  longPress: string;
  veryLongPress: string;
};

export const Actions = () => {
  const { t } = useTranslation();

  const [actions, setActions] = useState<ActionInfo[]>([]);
  const [bindings, setBindings] = useState<Bindings>({
    shortPress: '',
    doublePress: '',
    longPress: '',
    veryLongPress: ''
  });
  const [range, setRange] = useState({ min: 352, max: 511 });
  const [thresholds, setThresholds] = useState({ long: 1500, veryLong: 9000 });
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    api
      .getActions()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setActions(rsp.data?.actions ?? []);
        setBindings({
          shortPress: rsp.data?.shortPress ?? '',
          doublePress: rsp.data?.doublePress ?? '',
          longPress: rsp.data?.longPress ?? '',
          veryLongPress: rsp.data?.veryLongPress ?? ''
        });
        setRange({ min: rsp.data?.minGpio ?? 352, max: rsp.data?.maxGpio ?? 511 });
        setThresholds({
          long: rsp.data?.longPressMs ?? 1500,
          veryLong: rsp.data?.veryLongPressMs ?? 9000
        });
      })
      .catch(() => {
        // Fields keep their defaults if this fails.
      });
  }, []);

  function update(index: number, patch: Partial<ActionInfo>) {
    setActions((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)));
  }

  function updateStep(actionIndex: number, stepIndex: number, patch: Partial<ActionStep>) {
    setActions((prev) =>
      prev.map((action, i) =>
        i === actionIndex
          ? {
              ...action,
              steps: action.steps.map((step, j) => (j === stepIndex ? { ...step, ...patch } : step))
            }
          : action
      )
    );
  }

  function addStep(actionIndex: number) {
    setActions((prev) =>
      prev.map((action, i) =>
        i === actionIndex
          ? {
              ...action,
              steps: [
                ...action.steps,
                {
                  type: 'gpio',
                  gpio: { pin: range.min, mode: 'pulse', durationMs: 500, activeLow: false }
                }
              ]
            }
          : action
      )
    );
  }

  function moveStep(actionIndex: number, stepIndex: number, delta: number) {
    const target = stepIndex + delta;
    setActions((prev) =>
      prev.map((action, i) => {
        if (i !== actionIndex) return action;
        if (target < 0 || target >= action.steps.length) return action;
        const steps = [...action.steps];
        [steps[stepIndex], steps[target]] = [steps[target], steps[stepIndex]];
        return { ...action, steps };
      })
    );
  }

  function removeStep(actionIndex: number, stepIndex: number) {
    setActions((prev) =>
      prev.map((action, i) =>
        i === actionIndex
          ? { ...action, steps: action.steps.filter((_, j) => j !== stepIndex) }
          : action
      )
    );
  }

  function addAction() {
    setActions((prev) => [
      ...prev,
      {
        // The server replaces a "new-" id; this only has to be unique in React.
        id: `new-${Date.now()}-${prev.length}`,
        name: '',
        showInMenu: true,
        steps: [
          {
            type: 'gpio',
            gpio: { pin: range.min, mode: 'pulse', durationMs: 500, activeLow: false }
          }
        ]
      }
    ]);
  }

  function removeAction(index: number) {
    const removed = actions[index];
    if (removed) {
      // A binding left pointing at a deleted action is rejected on save.
      setBindings(
        (prev) =>
          Object.fromEntries(
            Object.entries(prev).map(([key, value]) => [key, value === removed.id ? '' : value])
          ) as Bindings
      );
    }
    setActions((prev) => prev.filter((_, i) => i !== index));
  }

  function save() {
    setIsLoading(true);

    api
      .setActions({ actions, ...bindings, keepDefaults: true })
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
          // The server reports which step failed and why.
          message.error(rsp.msg || t('settings.actions.runFailed'));
          return;
        }
        message.success(t('settings.actions.ran'));
      })
      .catch(() => message.error(t('settings.actions.runFailed')));
  }

  // A binding can only point at something already saved.
  const options = [
    { value: '', label: t('settings.actions.none') },
    ...actions
      .filter((action) => !action.id.startsWith('new-'))
      .map((action) => ({ value: action.id, label: action.name || action.id }))
  ];

  const bindingRows: { key: keyof Bindings; label: string; hint: string }[] = [
    { key: 'shortPress', label: t('settings.actions.shortPress'), hint: `< ${thresholds.long}ms` },
    { key: 'doublePress', label: t('settings.actions.doublePress'), hint: '' },
    {
      key: 'longPress',
      label: t('settings.actions.longPress'),
      hint: `${thresholds.long}–${thresholds.veryLong}ms`
    },
    {
      key: 'veryLongPress',
      label: t('settings.actions.veryLongPress'),
      hint: `> ${thresholds.veryLong}ms`
    }
  ];

  return (
    <>
      <div className="text-base">{t('settings.actions.title')}</div>
      <Divider className="opacity-50" />

      <span className="text-xs text-neutral-500">{t('settings.actions.desc')}</span>

      <div className="mt-4 flex items-center justify-between">
        <span>{t('settings.actions.list')}</span>
        <Button icon={<PlusIcon size={14} />} onClick={addAction}>
          {t('settings.actions.add')}
        </Button>
      </div>

      <div className="mt-3 flex flex-col space-y-4">
        {actions.map((action, index) => (
          <div key={action.id} className="flex flex-col space-y-3 rounded bg-neutral-800/40 p-3">
            <div className="flex items-center space-x-2">
              <Input
                value={action.name}
                placeholder={t('settings.actions.namePlaceholder')}
                onChange={(e) => update(index, { name: e.target.value })}
              />
              <Button danger icon={<Trash2Icon size={14} />} onClick={() => removeAction(index)} />
            </div>

            <div className="flex flex-col space-y-2">
              {action.steps.map((step, stepIndex) => (
                <StepEditor
                  key={stepIndex}
                  step={step}
                  index={stepIndex}
                  total={action.steps.length}
                  range={range}
                  onChange={(patch) => updateStep(index, stepIndex, patch)}
                  onMove={(delta) => moveStep(index, stepIndex, delta)}
                  onRemove={() => removeStep(index, stepIndex)}
                />
              ))}

              <Button size="small" icon={<PlusIcon size={12} />} onClick={() => addStep(index)}>
                {t('settings.actions.addStep')}
              </Button>
            </div>

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
        {bindingRows.map((row) => (
          <div key={row.key} className="flex items-center justify-between">
            <div className="flex flex-col">
              <span className="text-sm">{row.label}</span>
              {row.hint && <span className="text-xs text-neutral-500">{row.hint}</span>}
            </div>
            <Select
              value={bindings[row.key]}
              style={{ width: 220 }}
              options={options}
              onChange={(value) => setBindings((prev) => ({ ...prev, [row.key]: value }))}
            />
          </div>
        ))}
      </div>

      <div className="mt-4">
        <span className="text-xs text-neutral-500">{t('settings.actions.keepDefaultsDesc')}</span>
      </div>

      <div className="mt-6">
        <Button type="primary" loading={isLoading} onClick={save}>
          {t('settings.actions.save')}
        </Button>
      </div>
    </>
  );
};
