import { useEffect, useState } from 'react';
import { Button, message, Popconfirm, Tag } from 'antd';
import { useTranslation } from 'react-i18next';

import { getSessions, revokeSession } from '@/api/auth.ts';
import type { SessionInfo } from '@/api/auth.ts';

function since(unix: number) {
  if (!unix) return '-';
  const seconds = Math.max(0, Math.floor(Date.now() / 1000 - unix));
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

// The full user agent is unreadable in a narrow row; the browser name is the
// part that identifies the session to a person.
function browser(userAgent: string) {
  if (!userAgent) return 'Unknown';
  if (/Edg\//.test(userAgent)) return 'Edge';
  if (/OPR\//.test(userAgent)) return 'Opera';
  if (/Firefox\//.test(userAgent)) return 'Firefox';
  if (/Chrome\//.test(userAgent)) return 'Chrome';
  if (/Safari\//.test(userAgent)) return 'Safari';
  return 'Other';
}

export const Sessions = () => {
  const { t } = useTranslation();

  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [busy, setBusy] = useState('');

  useEffect(() => {
    refresh();
  }, []);

  function refresh() {
    getSessions()
      .then((rsp: any) => {
        if (rsp.code !== 0) return;
        setSessions(rsp.data?.sessions ?? []);
      })
      .catch(() => {
        // Nothing useful to show if this fails.
      });
  }

  function revoke(payload: { id?: string; all?: boolean }, key: string) {
    setBusy(key);

    revokeSession(payload)
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          message.error(rsp.msg || t('settings.account.sessionRevokeFailed'));
          return;
        }
        message.success(t('settings.account.sessionRevoked'));
        refresh();
      })
      .catch(() => message.error(t('settings.account.sessionRevokeFailed')))
      .finally(() => setBusy(''));
  }

  const others = sessions.filter((session) => !session.current);

  return (
    <div className="flex flex-col space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex flex-col">
          <span>{t('settings.account.sessions')}</span>
          <span className="text-xs text-neutral-500">{t('settings.account.sessionsDesc')}</span>
        </div>

        {others.length > 0 && (
          <Popconfirm
            title={t('settings.account.sessionRevokeAllConfirm')}
            onConfirm={() => revoke({ all: true }, 'all')}
          >
            <Button danger loading={busy === 'all'}>
              {t('settings.account.sessionRevokeAll')}
            </Button>
          </Popconfirm>
        )}
      </div>

      <div className="flex flex-col space-y-2">
        {sessions.map((session) => (
          <div
            key={session.id}
            className="flex items-center justify-between rounded bg-neutral-800/40 px-3 py-2"
          >
            <div className="flex flex-col">
              <div className="flex items-center space-x-2">
                <span className="text-sm">{browser(session.userAgent)}</span>
                {session.current && <Tag color="blue">{t('settings.account.sessionCurrent')}</Tag>}
              </div>
              <span className="text-xs text-neutral-500">
                {session.ip || '-'} · {t('settings.account.sessionLastSeen')}{' '}
                {since(session.lastSeen)}
              </span>
            </div>

            {!session.current && (
              <Popconfirm
                title={t('settings.account.sessionRevokeConfirm')}
                onConfirm={() => revoke({ id: session.id }, session.id)}
              >
                <Button size="small" danger loading={busy === session.id}>
                  {t('settings.account.sessionRevoke')}
                </Button>
              </Popconfirm>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};
