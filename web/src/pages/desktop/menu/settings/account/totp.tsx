import { useEffect, useState } from 'react';
import { Button, Input, message, Modal, QRCode, Tag } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/auth.ts';
import { encrypt } from '@/lib/encrypt.ts';

export const Totp = () => {
  const { t } = useTranslation();

  const [enabled, setEnabled] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const [setupUri, setSetupUri] = useState('');
  const [setupSecret, setSetupSecret] = useState('');
  const [code, setCode] = useState('');

  const [backupCodes, setBackupCodes] = useState<string[]>([]);

  const [isDisabling, setIsDisabling] = useState(false);
  const [password, setPassword] = useState('');

  useEffect(() => {
    refresh();
  }, []);

  function refresh() {
    api.getTotp().then((rsp: any) => {
      if (rsp.code !== 0) return;
      setEnabled(!!rsp.data?.enabled);
    });
  }

  function startSetup() {
    setIsLoading(true);
    api
      .setupTotp()
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.account.totpFailed'));
          return;
        }
        setSetupUri(rsp.data.uri);
        setSetupSecret(rsp.data.secret);
      })
      .finally(() => setIsLoading(false));
  }

  function confirmSetup() {
    setIsLoading(true);
    api
      .enableTotp(code.trim())
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.account.totpFailed'));
          return;
        }
        // Shown once and never retrievable again.
        setBackupCodes(rsp.data?.backupCodes || []);
        setSetupUri('');
        setSetupSecret('');
        setCode('');
        setEnabled(true);
        message.success(t('settings.account.totpEnabledMsg'));
      })
      .finally(() => setIsLoading(false));
  }

  function disable() {
    setIsLoading(true);
    api
      .disableTotp(encrypt(password), code.trim())
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.account.totpFailed'));
          return;
        }
        setIsDisabling(false);
        setPassword('');
        setCode('');
        setEnabled(false);
        message.success(t('settings.account.totpDisabledMsg'));
      })
      .finally(() => setIsLoading(false));
  }

  return (
    <>
      <div className="flex items-center justify-between">
        <div className="flex flex-col">
          <span>{t('settings.account.totp')}</span>
          <span className="text-xs text-neutral-500">{t('settings.account.totpDesc')}</span>
        </div>

        <div className="flex items-center space-x-3">
          <Tag color={enabled ? 'green' : 'default'}>
            {enabled ? t('settings.account.totpEnabled') : t('settings.account.totpDisabled')}
          </Tag>
          {enabled ? (
            <Button danger onClick={() => setIsDisabling(true)}>
              {t('settings.account.totpDisableBtn')}
            </Button>
          ) : (
            <Button loading={isLoading} onClick={startSetup}>
              {t('settings.account.totpEnable')}
            </Button>
          )}
        </div>
      </div>

      {/* Enrolment: the factor is not enforced until a code confirms it. */}
      <Modal
        open={!!setupUri}
        title={t('settings.account.totp')}
        onCancel={() => setSetupUri('')}
        onOk={confirmSetup}
        okText={t('settings.account.totpConfirm')}
        confirmLoading={isLoading}
        okButtonProps={{ disabled: code.trim().length === 0 }}
      >
        <div className="flex flex-col items-center space-y-3 py-3">
          <span className="text-sm">{t('settings.account.totpScan')}</span>
          <QRCode value={setupUri} size={180} />
          <span className="text-xs text-neutral-500">{t('settings.account.totpSecret')}</span>
          <code className="select-all break-all text-xs">{setupSecret}</code>
          <Input
            autoFocus
            inputMode="numeric"
            placeholder={t('settings.account.totpCode')}
            value={code}
            onChange={(e) => setCode(e.target.value)}
          />
        </div>
      </Modal>

      {/* Backup codes are displayed once, immediately after enabling. */}
      <Modal
        open={backupCodes.length > 0}
        title={t('settings.account.totpBackupTitle')}
        onCancel={() => setBackupCodes([])}
        onOk={() => setBackupCodes([])}
        okText={t('settings.account.totpDone')}
        cancelButtonProps={{ style: { display: 'none' } }}
      >
        <div className="flex flex-col space-y-3 py-3">
          <span className="text-xs text-neutral-500">{t('settings.account.totpBackupDesc')}</span>
          <div className="grid grid-cols-2 gap-2 rounded bg-neutral-800/60 p-3">
            {backupCodes.map((c) => (
              <code key={c} className="select-all text-sm">
                {c}
              </code>
            ))}
          </div>
        </div>
      </Modal>

      <Modal
        open={isDisabling}
        title={t('settings.account.totpDisableBtn')}
        onCancel={() => setIsDisabling(false)}
        onOk={disable}
        okText={t('settings.account.totpDisableBtn')}
        okButtonProps={{ danger: true, disabled: !password || !code.trim() }}
        confirmLoading={isLoading}
      >
        <div className="flex flex-col space-y-3 py-3">
          <Input.Password
            placeholder={t('settings.account.totpPassword')}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
          <Input
            inputMode="numeric"
            placeholder={t('settings.account.totpCode')}
            value={code}
            onChange={(e) => setCode(e.target.value)}
          />
        </div>
      </Modal>
    </>
  );
};
