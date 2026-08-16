import { useEffect, useState } from 'react';
import { Button, Input, message, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import type { OLEDDisplay } from '@/api/vm.ts';

const defaults: OLEDDisplay = {
  showIp: true,
  showRes: true,
  showType: true,
  showStream: true,
  showQuality: true,
  title: ''
};

export const OledDisplay = () => {
  const { t } = useTranslation();

  const [config, setConfig] = useState<OLEDDisplay>(defaults);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    api.getOLEDDisplay().then((rsp: any) => {
      if (rsp.code !== 0) return;
      setConfig({ ...defaults, ...rsp.data });
    });
  }, []);

  function update(patch: Partial<OLEDDisplay>) {
    setConfig((prev) => ({ ...prev, ...patch }));
  }

  function save() {
    setIsLoading(true);

    api
      .setOLEDDisplay(config)
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.device.oledSaveFailed'));
          return;
        }
        message.success(t('settings.device.oledSaved'));
      })
      .catch(() => message.error(t('settings.device.oledSaveFailed')))
      .finally(() => setIsLoading(false));
  }

  const rows: { key: keyof OLEDDisplay; label: string }[] = [
    { key: 'showIp', label: t('settings.device.oledIp') },
    { key: 'showRes', label: t('settings.device.oledRes') },
    { key: 'showType', label: t('settings.device.oledType') },
    { key: 'showStream', label: t('settings.device.oledStream') },
    { key: 'showQuality', label: t('settings.device.oledQuality') }
  ];

  return (
    <div className="flex flex-col space-y-4">
      <div className="flex flex-col">
        <span>{t('settings.device.oledDisplay')}</span>
        <span className="text-xs text-neutral-500">{t('settings.device.oledDisplayDesc')}</span>
      </div>

      {rows.map((row) => (
        <div key={row.key} className="flex items-center justify-between pl-2">
          <span className="text-sm">{row.label}</span>
          <Switch
            size="small"
            checked={config[row.key] as boolean}
            onChange={(v) => update({ [row.key]: v } as Partial<OLEDDisplay>)}
          />
        </div>
      ))}

      <div className="flex flex-col space-y-1">
        <span className="text-sm">{t('settings.device.oledTitle')}</span>
        <Input
          value={config.title}
          maxLength={20}
          placeholder={t('settings.device.oledTitlePlaceholder')}
          onChange={(e) => update({ title: e.target.value })}
        />
        <span className="text-xs text-neutral-500">{t('settings.device.oledTitleDesc')}</span>
      </div>

      <div>
        <Button type="primary" loading={isLoading} onClick={save}>
          {t('settings.device.oledSave')}
        </Button>
      </div>
    </div>
  );
};
