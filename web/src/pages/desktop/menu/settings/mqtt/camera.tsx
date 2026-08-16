import { useEffect, useState } from 'react';
import { Alert, Input, message, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/camera.ts';
import { getBaseUrl } from '@/lib/service.ts';

export const Camera = () => {
  const { t } = useTranslation();

  const [enabled, setEnabled] = useState(false);
  const [token, setToken] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    api.getCamera().then((rsp: any) => {
      if (rsp.code !== 0) return;
      setEnabled(!!rsp.data?.enabled);
      setToken(rsp.data?.token || '');
    });
  }, []);

  function toggle(next: boolean) {
    setIsLoading(true);

    api
      .setCamera(next)
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.mqtt.cameraFailed'));
          return;
        }
        setEnabled(!!rsp.data?.enabled);
        setToken(rsp.data?.token || '');
      })
      .catch(() => message.error(t('settings.mqtt.cameraFailed')))
      .finally(() => setIsLoading(false));
  }

  const streamUrl = token ? `${getBaseUrl('http')}/api/camera/mjpeg?token=${token}` : '';

  return (
    <>
      <div className="flex items-center justify-between">
        <div className="flex flex-col">
          <span>{t('settings.mqtt.camera')}</span>
          <span className="text-xs text-neutral-500">{t('settings.mqtt.cameraDesc')}</span>
        </div>
        <Switch checked={enabled} loading={isLoading} onChange={toggle} />
      </div>

      {enabled && streamUrl && (
        <div className="mt-4 flex flex-col space-y-2">
          <span className="text-sm">{t('settings.mqtt.cameraUrl')}</span>
          <Input.TextArea value={streamUrl} readOnly autoSize spellCheck={false} />
          <span className="text-xs text-neutral-500">{t('settings.mqtt.cameraUrlDesc')}</span>

          <Alert
            type="info"
            showIcon
            message={t('settings.mqtt.cameraHint')}
            description={
              <code className="whitespace-pre-wrap break-all text-xs">
                {`streams:\n  nanokvm: ${streamUrl}`}
              </code>
            }
          />
        </div>
      )}
    </>
  );
};
