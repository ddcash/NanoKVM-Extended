import { useEffect, useRef, useState } from 'react';
import { Button, Divider, Input, InputNumber, message, Tag } from 'antd';
import { PlusIcon, Trash2Icon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/switcher.ts';
import { getKeycode, getModifierBit } from '@/lib/keymap.ts';
import type { SwitcherStep, SwitcherTarget } from '@/api/switcher.ts';

function keyLabel(event: KeyboardEvent) {
  const map: Record<string, string> = {
    ControlLeft: 'Ctrl',
    ControlRight: 'Ctrl',
    AltLeft: 'Alt',
    AltRight: 'Alt',
    ShiftLeft: 'Shift',
    ShiftRight: 'Shift',
    MetaLeft: 'Win',
    MetaRight: 'Win',
    ScrollLock: 'ScrLk',
    NumLock: 'NumLk',
    CapsLock: 'Caps',
    Escape: 'Esc',
    Delete: 'Del'
  };

  if (map[event.code]) return map[event.code];
  if (event.code.startsWith('Key')) return event.code.slice(3);
  if (event.code.startsWith('Digit')) return event.code.slice(5);
  if (event.code.startsWith('Numpad')) return `Num ${event.code.slice(6)}`;
  return event.code;
}

function describe(steps: SwitcherStep[]) {
  return steps.map((step) => step.keys.map((k) => k.label).join('+')).join(' , ');
}

export const Switcher = () => {
  const { t } = useTranslation();

  const [targets, setTargets] = useState<SwitcherTarget[]>([]);
  const [stepDelayMs, setStepDelayMs] = useState(120);
  const [isLoading, setIsLoading] = useState(false);

  // Index of the target currently capturing keys, or null.
  const [recordingIndex, setRecordingIndex] = useState<number | null>(null);
  const heldRef = useRef<Set<string>>(new Set());
  const stepRef = useRef<SwitcherStep | null>(null);

  useEffect(() => {
    api.getSwitcher().then((rsp: any) => {
      if (rsp.code !== 0) return;
      setTargets(rsp.data?.targets ?? []);
      setStepDelayMs(rsp.data?.stepDelayMs ?? 120);
    });
  }, []);

  useEffect(() => {
    if (recordingIndex === null) return;

    function onKeyDown(event: KeyboardEvent) {
      event.preventDefault();
      event.stopPropagation();

      // Keys held together form one step; a tap after everything is released
      // starts the next one. That distinguishes Ctrl+Alt+1 from ScrLk ScrLk 1.
      if (heldRef.current.size === 0) {
        stepRef.current = { keys: [] };
      }
      if (heldRef.current.has(event.code)) return;

      heldRef.current.add(event.code);
      stepRef.current?.keys.push({
        code: event.code,
        label: keyLabel(event),
        keycode: getKeycode(event.code) ?? 0,
        modifier: getModifierBit(event.code)
      });

      const step = stepRef.current;
      setTargets((prev) =>
        prev.map((target, i) => {
          if (i !== recordingIndex || !step) return target;
          const steps = [...target.steps];
          // Replace the in-progress step rather than appending a new one.
          if (steps.length > 0 && steps[steps.length - 1] === step) {
            steps[steps.length - 1] = { keys: [...step.keys] };
          } else {
            steps.push({ keys: [...step.keys] });
          }
          return { ...target, steps };
        })
      );
    }

    function onKeyUp(event: KeyboardEvent) {
      event.preventDefault();
      heldRef.current.delete(event.code);
      if (heldRef.current.size === 0) {
        stepRef.current = null;
      }
    }

    window.addEventListener('keydown', onKeyDown, true);
    window.addEventListener('keyup', onKeyUp, true);
    return () => {
      window.removeEventListener('keydown', onKeyDown, true);
      window.removeEventListener('keyup', onKeyUp, true);
    };
  }, [recordingIndex]);

  function update(index: number, patch: Partial<SwitcherTarget>) {
    setTargets((prev) => prev.map((tgt, i) => (i === index ? { ...tgt, ...patch } : tgt)));
  }

  function addTarget() {
    setTargets((prev) => [...prev, { id: `t${Date.now()}${prev.length}`, name: '', steps: [] }]);
  }

  function removeTarget(index: number) {
    if (recordingIndex === index) setRecordingIndex(null);
    setTargets((prev) => prev.filter((_, i) => i !== index));
  }

  function toggleRecording(index: number) {
    if (recordingIndex === index) {
      setRecordingIndex(null);
      return;
    }
    heldRef.current.clear();
    stepRef.current = null;
    update(index, { steps: [] });
    setRecordingIndex(index);
  }

  function save() {
    setRecordingIndex(null);
    setIsLoading(true);

    api
      .setSwitcher({ targets, stepDelayMs })
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.switcher.saveFailed'));
          return;
        }
        message.success(t('settings.switcher.saved'));
      })
      .catch(() => message.error(t('settings.switcher.saveFailed')))
      .finally(() => setIsLoading(false));
  }

  return (
    <>
      <div className="text-base">{t('settings.switcher.title')}</div>
      <Divider className="opacity-50" />

      <div className="flex flex-col space-y-1">
        <span className="text-xs text-neutral-500">{t('settings.switcher.desc')}</span>
      </div>

      <div className="mt-5 flex items-center justify-between">
        <div className="flex flex-col">
          <span>{t('settings.switcher.stepDelay')}</span>
          <span className="text-xs text-neutral-500">{t('settings.switcher.stepDelayDesc')}</span>
        </div>
        <InputNumber
          min={0}
          max={5000}
          step={20}
          value={stepDelayMs}
          onChange={(v) => setStepDelayMs(v ?? 120)}
        />
      </div>

      <Divider className="opacity-50" style={{ margin: '24px 0' }} />

      <div className="flex items-center justify-between">
        <span>{t('settings.switcher.targets')}</span>
        <Button icon={<PlusIcon size={14} />} onClick={addTarget}>
          {t('settings.switcher.addTarget')}
        </Button>
      </div>

      <div className="mt-4 flex flex-col space-y-3">
        {targets.map((target, index) => (
          <div key={target.id} className="flex flex-col space-y-2 rounded bg-neutral-800/40 p-3">
            <div className="flex items-center space-x-2">
              <Input
                value={target.name}
                placeholder={t('settings.switcher.namePlaceholder')}
                onChange={(e) => update(index, { name: e.target.value })}
              />
              <Button
                type={recordingIndex === index ? 'primary' : 'default'}
                danger={recordingIndex === index}
                onClick={() => toggleRecording(index)}
              >
                {recordingIndex === index
                  ? t('settings.switcher.stopRecording')
                  : t('settings.switcher.record')}
              </Button>
              <Button danger icon={<Trash2Icon size={14} />} onClick={() => removeTarget(index)} />
            </div>

            <div className="flex min-h-[24px] items-center space-x-2">
              {target.steps.length > 0 ? (
                <Tag color="blue">{describe(target.steps)}</Tag>
              ) : (
                <span className="text-xs text-neutral-500">
                  {recordingIndex === index
                    ? t('settings.switcher.recording')
                    : t('settings.switcher.noKeys')}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>

      <div className="mt-6">
        <Button type="primary" loading={isLoading} onClick={save}>
          {t('settings.switcher.save')}
        </Button>
      </div>
    </>
  );
};
