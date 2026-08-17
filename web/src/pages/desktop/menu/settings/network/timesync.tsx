import { useEffect, useState } from 'react';
import { Button, Divider, Input, message, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import { getTimeSync, setTimeSync } from '@/api/network.ts';

export const TimeSync = () => {
  const { t } = useTranslation();

  const [ntpEnabled, setNtpEnabled] = useState(true);
  const [servers, setServers] = useState('');
  const [stun, setStun] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    getTimeSync()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setNtpEnabled(!!rsp.data?.ntpEnabled);
        setServers((rsp.data?.ntpServers ?? []).join('\n'));
        // "disable" is the stored sentinel; an empty box means the same thing.
        setStun(rsp.data?.stun === 'disable' ? '' : (rsp.data?.stun ?? ''));
      })
      .catch(() => {
        // Nothing useful to show; the fields keep their defaults.
      });
  }, []);

  function save() {
    setIsLoading(true);

    setTimeSync({
      ntpEnabled,
      ntpServers: servers
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean),
      stun: stun.trim()
    })
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.network.timeSaveFailed'));
          return;
        }
        message.success(t('settings.network.timeSaved'));
      })
      .catch(() => message.error(t('settings.network.timeSaveFailed')))
      .finally(() => setIsLoading(false));
  }

  return (
    <>
      <Divider className="opacity-50" style={{ margin: '28px 0' }} />

      <div className="flex items-center justify-between">
        <div className="flex flex-col">
          <span>{t('settings.network.ntp')}</span>
          <span className="text-xs text-neutral-500">{t('settings.network.ntpDesc')}</span>
        </div>
        <Switch checked={ntpEnabled} onChange={setNtpEnabled} />
      </div>

      {ntpEnabled && (
        <div className="mt-3 flex flex-col space-y-1">
          <Input.TextArea
            rows={4}
            value={servers}
            spellCheck={false}
            placeholder={'0.pool.ntp.org\n1.pool.ntp.org'}
            onChange={(e) => setServers(e.target.value)}
          />
          <span className="text-xs text-neutral-500">{t('settings.network.ntpServersDesc')}</span>
        </div>
      )}

      <div className="mt-6 flex flex-col space-y-1">
        <span>{t('settings.network.stun')}</span>
        <Input
          value={stun}
          spellCheck={false}
          placeholder="stun.example.org:19302"
          onChange={(e) => setStun(e.target.value)}
        />
        <span className="text-xs text-neutral-500">{t('settings.network.stunDesc')}</span>
      </div>

      <div className="mt-5">
        <Button type="primary" loading={isLoading} onClick={save}>
          {t('settings.network.timeSave')}
        </Button>
      </div>
    </>
  );
};
