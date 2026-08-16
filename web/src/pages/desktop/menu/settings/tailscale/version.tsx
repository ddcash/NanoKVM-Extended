import { useEffect, useState } from 'react';
import { Button, message, Tag } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/extensions/tailscale.ts';
import type { TailscaleVersion } from '@/api/extensions/tailscale.ts';

export const Version = () => {
  const { t } = useTranslation();

  const [version, setVersion] = useState<TailscaleVersion | null>(null);
  const [isUpdating, setIsUpdating] = useState(false);

  useEffect(() => {
    refresh();
  }, []);

  function refresh() {
    api
      .getVersion()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setVersion(rsp.data);
      })
      .catch(() => {
        // The available version needs the network; failing to reach it is not
        // worth an error, the version simply shows as unknown.
      });
  }

  function update() {
    setIsUpdating(true);

    api
      .update()
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.tailscale.updateFailed'));
          return;
        }
        setVersion(rsp.data);
        message.success(t('settings.tailscale.updated'));
      })
      .catch(() => message.error(t('settings.tailscale.updateFailed')))
      .finally(() => setIsUpdating(false));
  }

  if (!version) return null;

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-col">
        <span className="text-sm">{t('settings.tailscale.version')}</span>
        <div className="flex items-center space-x-2">
          <span className="text-xs text-neutral-500">
            {version.installed || t('settings.tailscale.versionUnknown')}
          </span>
          {version.updateAvailable && <Tag color="blue">{version.latest}</Tag>}
        </div>
      </div>

      {version.updateAvailable ? (
        <Button type="primary" loading={isUpdating} onClick={update}>
          {t('settings.tailscale.update')}
        </Button>
      ) : (
        <Button loading={isUpdating} onClick={update}>
          {t('settings.tailscale.reinstall')}
        </Button>
      )}
    </div>
  );
};
