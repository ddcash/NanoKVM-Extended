import { ReactElement, useEffect, useState } from 'react';
import { LockOutlined, SafetyOutlined, UserOutlined } from '@ant-design/icons';
import { Button, Form, Input } from 'antd';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import * as api from '@/api/auth.ts';
import { existToken, setToken } from '@/lib/cookie.ts';
import { encrypt } from '@/lib/encrypt.ts';
import { Head } from '@/components/head.tsx';

import { Tips } from './tips.tsx';

export const Login = (): ReactElement => {
  const navigate = useNavigate();
  const { t } = useTranslation();

  const [isLoading, setIsloading] = useState(false);
  const [msg, setMsg] = useState('');
  // Revealed only after the server reports a code is required, so the field
  // stays hidden for accounts that have not enabled two-factor.
  const [isCodeRequired, setIsCodeRequired] = useState(false);

  useEffect(() => {
    if (existToken()) {
      navigate('/', { replace: true });
    }
  }, []);

  useEffect(() => {
    if (msg) {
      setTimeout(() => setMsg(''), 3000);
    }
  }, [msg]);

  function login(values: any) {
    if (isLoading) return;
    setIsloading(true);

    const username = values.username;
    const password = encrypt(values.password);

    api
      .login(username, password, values.code)
      .then((rsp: any) => {
        // -6 means the password was accepted but a code is still needed.
        if (rsp.code === -6) {
          setIsCodeRequired(true);
          setMsg(t('auth.codeRequired'));
          return;
        }

        if (rsp.code !== 0) {
          let errorMsg = t('auth.error');
          if (rsp.code === -2) errorMsg = t('auth.invalidUser');
          else if (rsp.code === -5) errorMsg = t('auth.locked');
          else if (rsp.code === -4) errorMsg = t('auth.globalLocked');
          else if (rsp.code === -7) {
            errorMsg = t('auth.invalidCode');
            setIsCodeRequired(true);
          }

          setMsg(errorMsg);
          return;
        }

        setMsg('');
        setToken(rsp.data.token);

        navigate('/', { replace: true });
        window.location.reload();
      })
      .catch(() => {
        setMsg(t('auth.error'));
      })
      .finally(() => {
        setIsloading(false);
      });
  }

  return (
    <>
      <Head title={t('head.login')} />

      <div className="flex h-screen w-screen flex-col items-center justify-center">
        <Form
          style={{ minWidth: 300, maxWidth: 500 }}
          initialValues={{ remember: true }}
          onFinish={login}
        >
          <div className="flex flex-col items-center justify-center pb-4">
            <img
              id="logo"
              src="/sipeed.ico"
              alt="Sipeed"
              onClick={(evt) => {
                evt.preventDefault();
                (evt.target as HTMLImageElement).classList.add('animate-spin');
                setTimeout(() => {
                  (evt.target as HTMLImageElement).classList.remove('animate-spin');
                }, 1000);
              }}
            />
          </div>
          <Form.Item
            name="username"
            rules={[{ required: true, message: t('auth.noEmptyUsername'), min: 1 }]}
          >
            <Input prefix={<UserOutlined />} placeholder={t('auth.placeholderUsername')} />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: t('auth.noEmptyPassword'), min: 1 }]}
          >
            <Input
              prefix={<LockOutlined />}
              type="password"
              placeholder={t('auth.placeholderPassword')}
            />
          </Form.Item>

          {isCodeRequired && (
            <Form.Item
              name="code"
              rules={[{ required: true, message: t('auth.noEmptyCode'), min: 1 }]}
            >
              <Input
                prefix={<SafetyOutlined />}
                autoFocus
                autoComplete="one-time-code"
                inputMode="numeric"
                placeholder={t('auth.placeholderCode')}
              />
            </Form.Item>
          )}

          <div className="pb-1 text-red-500">{msg}</div>

          <Form.Item>
            <Button type="primary" htmlType="submit" className="w-full" loading={isLoading}>
              {t('auth.loginButtonText')}
            </Button>
          </Form.Item>

          <div className="flex justify-end pb-4 text-sm">
            <Tips />
          </div>
        </Form>
      </div>
    </>
  );
};
