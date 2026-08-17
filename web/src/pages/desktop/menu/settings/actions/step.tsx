import { Alert, Button, Input, InputNumber, Select, Switch } from 'antd';
import { ChevronDownIcon, ChevronUpIcon, Trash2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { ActionStep } from '@/api/actions.ts';

type StepEditorProps = {
  step: ActionStep;
  index: number;
  total: number;
  range: { min: number; max: number };
  onChange: (patch: Partial<ActionStep>) => void;
  onMove: (delta: number) => void;
  onRemove: () => void;
};

export const StepEditor = ({
  step,
  index,
  total,
  range,
  onChange,
  onMove,
  onRemove
}: StepEditorProps) => {
  const { t } = useTranslation();

  // Each branch fills in its own defaults, so switching type never leaves the
  // step half-configured and failing validation on save.
  function changeType(type: string) {
    const patch: Partial<ActionStep> = { type };

    if (type === 'gpio' && !step.gpio) {
      patch.gpio = { pin: range.min, mode: 'pulse', durationMs: 500, activeLow: false };
    }
    if (type === 'pwm' && !step.pwm) {
      patch.pwm = { chip: 0, channel: 0, periodNs: 1000000, dutyPercent: 50, enable: true };
    }
    if (type === 'command' && !step.command) {
      patch.command = { command: '', background: false, timeoutSec: 30 };
    }
    if (type === 'delay' && !step.delayMs) {
      patch.delayMs = 500;
    }

    onChange(patch);
  }

  return (
    <div className="flex flex-col space-y-2 rounded border border-neutral-700/60 p-2">
      <div className="flex items-center space-x-1">
        <span className="w-5 text-xs text-neutral-500">{index + 1}</span>

        <Select
          value={step.type}
          style={{ width: 120 }}
          onChange={changeType}
          options={[
            { value: 'gpio', label: t('settings.actions.typeGpio') },
            { value: 'pwm', label: t('settings.actions.typePwm') },
            { value: 'command', label: t('settings.actions.typeCommand') },
            { value: 'delay', label: t('settings.actions.typeDelay') }
          ]}
        />

        <div className="flex-1" />

        <Button
          size="small"
          icon={<ChevronUpIcon size={12} />}
          disabled={index === 0}
          onClick={() => onMove(-1)}
        />
        <Button
          size="small"
          icon={<ChevronDownIcon size={12} />}
          disabled={index === total - 1}
          onClick={() => onMove(1)}
        />
        <Button size="small" danger icon={<Trash2Icon size={12} />} onClick={onRemove} />
      </div>

      {step.type === 'gpio' && step.gpio && (
        <div className="flex flex-col space-y-2 pl-6">
          <div className="flex flex-wrap items-center gap-2">
            <InputNumber
              addonBefore="GPIO"
              min={range.min}
              max={range.max}
              value={step.gpio.pin}
              onChange={(value) => onChange({ gpio: { ...step.gpio!, pin: value ?? range.min } })}
            />
            <Select
              value={step.gpio.mode}
              style={{ width: 120 }}
              onChange={(value) => onChange({ gpio: { ...step.gpio!, mode: value } })}
              options={[
                { value: 'pulse', label: t('settings.actions.modePulse') },
                { value: 'high', label: t('settings.actions.modeHigh') },
                { value: 'low', label: t('settings.actions.modeLow') },
                { value: 'toggle', label: t('settings.actions.modeToggle') }
              ]}
            />
            {step.gpio.mode === 'pulse' && (
              <InputNumber
                addonAfter="ms"
                min={0}
                max={10000}
                step={100}
                value={step.gpio.durationMs}
                onChange={(value) =>
                  onChange({ gpio: { ...step.gpio!, durationMs: value ?? 500 } })
                }
              />
            )}
          </div>

          <div className="flex items-center space-x-2">
            <Switch
              size="small"
              checked={step.gpio.activeLow}
              onChange={(value) => onChange({ gpio: { ...step.gpio!, activeLow: value } })}
            />
            <span className="text-xs text-neutral-500">{t('settings.actions.activeLow')}</span>
          </div>

          {step.gpio.reservedFor && (
            <Alert
              type="warning"
              showIcon
              message={t('settings.actions.reserved', { use: step.gpio.reservedFor })}
            />
          )}
        </div>
      )}

      {step.type === 'pwm' && step.pwm && (
        <div className="flex flex-col space-y-2 pl-6">
          <div className="flex flex-wrap items-center gap-2">
            <InputNumber
              addonBefore="chip"
              min={0}
              value={step.pwm.chip}
              onChange={(value) => onChange({ pwm: { ...step.pwm!, chip: value ?? 0 } })}
            />
            <InputNumber
              addonBefore="ch"
              min={0}
              value={step.pwm.channel}
              onChange={(value) => onChange({ pwm: { ...step.pwm!, channel: value ?? 0 } })}
            />
            <InputNumber
              addonBefore="%"
              min={0}
              max={100}
              value={step.pwm.dutyPercent}
              onChange={(value) => onChange({ pwm: { ...step.pwm!, dutyPercent: value ?? 0 } })}
            />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <InputNumber
              addonBefore="period"
              addonAfter="ns"
              min={0}
              step={100000}
              style={{ width: 220 }}
              value={step.pwm.periodNs}
              onChange={(value) => onChange({ pwm: { ...step.pwm!, periodNs: value ?? 1000000 } })}
            />
            <Switch
              size="small"
              checked={step.pwm.enable}
              onChange={(value) => onChange({ pwm: { ...step.pwm!, enable: value } })}
            />
            <span className="text-xs text-neutral-500">{t('settings.actions.pwmEnable')}</span>
          </div>

          <span className="text-xs text-neutral-500">{t('settings.actions.pwmHint')}</span>
        </div>
      )}

      {step.type === 'command' && step.command && (
        <div className="flex flex-col space-y-2 pl-6">
          <Input.TextArea
            rows={2}
            spellCheck={false}
            value={step.command.command}
            placeholder="/etc/init.d/S95nanokvm restart"
            onChange={(e) => onChange({ command: { ...step.command!, command: e.target.value } })}
          />

          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center space-x-2">
              <Switch
                size="small"
                checked={step.command.background}
                onChange={(value) => onChange({ command: { ...step.command!, background: value } })}
              />
              <span className="text-xs text-neutral-500">
                {t('settings.actions.commandBackground')}
              </span>
            </div>

            {!step.command.background && (
              <InputNumber
                addonBefore={t('settings.actions.timeout')}
                addonAfter="s"
                min={0}
                max={300}
                value={step.command.timeoutSec}
                onChange={(value) =>
                  onChange({ command: { ...step.command!, timeoutSec: value ?? 30 } })
                }
              />
            )}
          </div>
        </div>
      )}

      {step.type === 'delay' && (
        <div className="pl-6">
          <InputNumber
            addonAfter="ms"
            min={1}
            max={60000}
            step={100}
            value={step.delayMs}
            onChange={(value) => onChange({ delayMs: value ?? 500 })}
          />
        </div>
      )}
    </div>
  );
};
